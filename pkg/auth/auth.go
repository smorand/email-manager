// Package auth provides OAuth2 authentication for Google APIs.
// This package contains unified scopes for both Gmail and People APIs,
// enabling a single OAuth consent for multiple applications.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
	people "google.golang.org/api/people/v1"
)

const (
	// DefaultCredentialsFile is the fallback name of the OAuth credentials file.
	DefaultCredentialsFile = "google_credentials.json"
	// TokenDir is the directory name for per-account token files.
	TokenDir = "email-manager"

	// tokenDirPerm is the permission for the token directory (owner only).
	tokenDirPerm = 0o700
	// tokenFilePerm is the permission for token files (owner only, contains secrets).
	tokenFilePerm = 0o600

	// oauthCallbackPort is the local port used for the OAuth2 redirect.
	// Bind to 127.0.0.1 explicitly so we don't expose the callback to the network.
	oauthCallbackPort = "127.0.0.1:8002"
	// oauthCallbackPath is the URL path that receives the OAuth2 callback.
	oauthCallbackPath = "/oauth2callback"
	// oauthRedirectURL is the full redirect URL (must match the OAuth client).
	// Always uses localhost (loopback) regardless of the literal bind address.
	oauthRedirectURL = "http://localhost:8002" + oauthCallbackPath

	// oauthReadHeaderTimeout protects the local callback server against
	// slowloris-style header-read stalls.
	oauthReadHeaderTimeout = 5 * time.Second

	// oauthTimeout is the maximum duration to wait for the user to complete
	// the OAuth flow in their browser.
	oauthTimeout = 3 * time.Minute
	// oauthShutdownTimeout is the grace period for shutting down the local
	// callback HTTP server after receiving the code.
	oauthShutdownTimeout = 5 * time.Second
)

// Scopes contains all OAuth2 scopes for Gmail and People APIs.
// These unified scopes enable a single OAuth consent for both email-manager
// and google-contacts applications, using the same token file.
var Scopes = []string{
	// Gmail API scopes (for email-manager)
	gmail.GmailModifyScope,
	gmail.GmailSendScope,
	gmail.GmailLabelsScope,
	// gmail.settings.basic is needed to read the user's sendAs display name
	// so that an RFC 2047 encoded From header can be set on outgoing messages
	// (avoids mojibake on names with non-ASCII characters).
	gmail.GmailSettingsBasicScope,
	// People API scopes (for google-contacts)
	people.ContactsScope,
	people.ContactsOtherReadonlyScope,
}

// GetCredentialsPath returns the path to the credentials directory.
func GetCredentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".credentials")
}

// GetCredentialsFilePath returns the path to the credentials file.
// It checks the GOOGLE_CREDENTIALS_FILE environment variable first,
// then falls back to the default location.
func GetCredentialsFilePath() string {
	if envPath := os.Getenv("GOOGLE_CREDENTIALS_FILE"); envPath != "" {
		return envPath
	}
	return filepath.Join(GetCredentialsPath(), DefaultCredentialsFile)
}

// GetTokenDir returns the directory for per-account token files.
func GetTokenDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", TokenDir)
}

// GetTokenPathForAccount returns the token file path for a specific account.
func GetTokenPathForAccount(account string) string {
	return filepath.Join(GetTokenDir(), account+".json")
}

// ListAccounts returns all authenticated account email addresses.
func ListAccounts() ([]string, error) {
	tokenDir := GetTokenDir()
	entries, err := os.ReadDir(tokenDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("unable to read token directory: %w", err)
	}
	var accounts []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			account := strings.TrimSuffix(entry.Name(), ".json")
			accounts = append(accounts, account)
		}
	}
	return accounts, nil
}

// RemoveToken removes the token file for a specific account.
func RemoveToken(account string) error {
	tokenPath := GetTokenPathForAccount(account)
	if _, err := os.Stat(tokenPath); err == nil {
		return os.Remove(tokenPath)
	}
	return nil
}

// GetClient returns an HTTP client with OAuth2 authentication for the given
// account. The returned client transparently refreshes the access token on
// every request and, if the refresh itself fails (refresh token revoked,
// missing, or otherwise invalid), opens the browser to re-run the OAuth
// consent flow and persists the new token before retrying.
func GetClient(ctx context.Context, account string) (*http.Client, error) {
	credPath := GetCredentialsFilePath()
	tokenPath := GetTokenPathForAccount(account)

	b, err := os.ReadFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read credentials file %s: %w", credPath, err)
	}

	config, err := google.ConfigFromJSON(b, Scopes...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse credentials: %w", err)
	}

	token, err := tokenFromFile(tokenPath)
	if err != nil {
		token, err = getTokenFromWeb(ctx, config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, token); err != nil {
			return nil, fmt.Errorf("unable to save token to %s: %w", tokenPath, err)
		}
	}

	src := &reauthTokenSource{
		ctx:       ctx,
		config:    config,
		base:      config.TokenSource(ctx, token),
		tokenPath: tokenPath,
		lastSaved: token.AccessToken,
	}
	return oauth2.NewClient(ctx, src), nil
}

// reauthTokenSource wraps oauth2's standard refreshing TokenSource so that:
//  1. when the underlying refresh fails (e.g. the refresh token has been
//     revoked or the saved token is missing one), the user's browser is
//     re-launched through the same OAuth flow used by the `auth` command,
//     instead of bubbling up a cryptic "invalid_grant" error to the caller;
//  2. when the refresh succeeds and a new access token is issued, that token
//     is persisted to disk so subsequent processes do not need to refresh
//     again immediately.
type reauthTokenSource struct {
	ctx       context.Context
	config    *oauth2.Config
	base      oauth2.TokenSource
	tokenPath string
	lastSaved string
}

func (r *reauthTokenSource) Token() (*oauth2.Token, error) {
	tok, err := r.base.Token()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Auth: token refresh failed (%v); opening browser to re-authenticate...\n", err)
		if rmErr := os.Remove(r.tokenPath); rmErr != nil && !os.IsNotExist(rmErr) {
			return nil, fmt.Errorf("unable to remove stale token at %s: %w", r.tokenPath, rmErr)
		}
		newTok, werr := getTokenFromWeb(r.ctx, r.config)
		if werr != nil {
			return nil, fmt.Errorf("re-authentication failed: %w", werr)
		}
		if serr := saveToken(r.tokenPath, newTok); serr != nil {
			return nil, fmt.Errorf("unable to save refreshed token: %w", serr)
		}
		r.base = r.config.TokenSource(r.ctx, newTok)
		r.lastSaved = newTok.AccessToken
		return newTok, nil
	}
	if tok.AccessToken != r.lastSaved {
		if serr := saveToken(r.tokenPath, tok); serr != nil {
			fmt.Fprintf(os.Stderr, "warning: unable to persist refreshed token: %v\n", serr)
		} else {
			r.lastSaved = tok.AccessToken
		}
	}
	return tok, nil
}

const oauthSuccessHTML = `<!doctype html>
<html><body>
  <h1>Authentication successful!</h1>
  <p>You can close this window and return to the terminal.</p>
</body></html>`

func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	config.RedirectURL = oauthRedirectURL

	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	mux := http.NewServeMux()
	listener, err := net.Listen("tcp", oauthCallbackPort)
	if err != nil {
		return nil, fmt.Errorf("unable to listen on %s: %w", oauthCallbackPort, err)
	}
	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: oauthReadHeaderTimeout,
	}

	mux.HandleFunc(oauthCallbackPath, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in OAuth callback")
			return
		}

		w.Header().Set("Content-Type", "text/html")
		if _, werr := fmt.Fprint(w, oauthSuccessHTML); werr != nil {
			fmt.Fprintf(os.Stderr, "warning: unable to write success page: %v\n", werr)
		}

		codeChan <- code
	})

	// Start server. Listener is already bound, so the server is ready as soon
	// as Serve is called — no need for a sleep-based race.
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If browser doesn't open, visit:\n%v\n\n", authURL)

	openBrowser(authURL)

	var code string
	select {
	case code = <-codeChan:
	case err := <-errChan:
		return nil, err
	case <-time.After(oauthTimeout):
		return nil, fmt.Errorf("authentication timeout after %s", oauthTimeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, oauthShutdownTimeout)
	defer cancel()
	_ = server.Shutdown(shutdownCtx)

	tok, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}

	fmt.Println("\nAuthentication successful!")
	return tok, nil
}

// openBrowser launches the user's default browser to view the given OAuth URL.
// The URL is built by golang.org/x/oauth2 from the application's static config,
// so it is trusted input.
func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url) //nolint:gosec // trusted OAuth URL
	case "linux":
		cmd = exec.Command("xdg-open", url) //nolint:gosec // trusted OAuth URL
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url) //nolint:gosec // trusted OAuth URL
	default:
		return
	}
	_ = cmd.Start()
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file) //nolint:gosec // path is built from a controlled cache directory
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Fprintf(os.Stderr, "Saving credentials to: %s\n", path)

	if err := os.MkdirAll(filepath.Dir(path), tokenDirPerm); err != nil {
		return fmt.Errorf("unable to create token directory: %w", err)
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, tokenFilePerm)
	if err != nil {
		return fmt.Errorf("unable to open token file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// Persisting the access/refresh token is the explicit purpose of this
	// function; gosec's secret-pattern heuristic does not apply.
	if err := json.NewEncoder(f).Encode(token); err != nil { //nolint:gosec // intentional token persistence
		return fmt.Errorf("unable to encode token: %w", err)
	}
	return nil
}
