// Package auth provides OAuth2 authentication for Google APIs.
// This package contains unified scopes for both Gmail and People APIs,
// enabling a single OAuth consent for multiple applications.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

const (
	// CredentialsFile is the name of the OAuth credentials file.
	CredentialsFile = "google_credentials.json"
	// TokenFile is the name of the token file.
	TokenFile = "google_token.json"
)

// Scopes contains all OAuth2 scopes for Gmail API.
var Scopes = []string{
	gmail.GmailModifyScope,
	gmail.GmailSendScope,
	gmail.GmailLabelsScope,
}

// GetCredentialsPath returns the path to the credentials directory.
func GetCredentialsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".credentials")
}

// GetClient returns an HTTP client with OAuth2 authentication.
func GetClient(ctx context.Context) (*http.Client, error) {
	credPath := filepath.Join(GetCredentialsPath(), CredentialsFile)
	tokenPath := filepath.Join(GetCredentialsPath(), TokenFile)

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
		token, err = getTokenFromWeb(config)
		if err != nil {
			return nil, err
		}
		if err := saveToken(tokenPath, token); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: unable to save token: %v\n", err)
		}
	}

	return config.Client(ctx, token), nil
}

func getTokenFromWeb(config *oauth2.Config) (*oauth2.Token, error) {
	// Use localhost with configured port
	config.RedirectURL = "http://localhost:8080/oauth2callback"

	// Create channels for communication
	codeChan := make(chan string)
	errChan := make(chan error)

	// Start local HTTP server
	server := &http.Server{Addr: ":8080"}
	http.HandleFunc("/oauth2callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			return
		}

		// Send success message to browser
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<body>
				<h1>Authentication successful!</h1>
				<p>You can close this window and return to the terminal.</p>
			</body>
			</html>
		`)

		codeChan <- code
	})

	// Start server in background
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// Ignore server closed error
			if err != http.ErrServerClosed {
				errChan <- err
			}
		}
	}()

	// Wait a moment for server to start
	time.Sleep(100 * time.Millisecond)

	// Generate auth URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Opening browser for authentication...\n")
	fmt.Printf("If browser doesn't open, visit:\n%v\n\n", authURL)

	// Try to open browser automatically
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", authURL)
	case "linux":
		cmd = exec.Command("xdg-open", authURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", authURL)
	}

	if cmd != nil {
		_ = cmd.Start()
	}

	// Wait for auth code or error
	var code string
	select {
	case code = <-codeChan:
		// Success
	case err := <-errChan:
		return nil, err
	case <-time.After(3 * time.Minute):
		return nil, fmt.Errorf("authentication timeout after 3 minutes")
	}

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)

	// Exchange code for token
	tok, err := config.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve token from web: %w", err)
	}

	fmt.Println("\nAuthentication successful!")
	return tok, nil
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	token := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(token)
	return token, err
}

func saveToken(path string, token *oauth2.Token) error {
	fmt.Fprintf(os.Stderr, "Saving credentials to: %s\n", path)

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return json.NewEncoder(f).Encode(token)
}
