# Email Manager - AI Development Guide

## Project Overview

**Type**: CLI Application
**Language**: Go 1.25+
**Purpose**: Gmail management via Gmail API v1 (multi-account)
**Authentication**: OAuth2 with Google (per-account tokens)
**CLI Framework**: Cobra

## Project Structure

Following golang skill conventions:

```
email-manager/
├── go.mod                    # Module at root
├── go.sum
├── Makefile                  # Build automation
├── README.md                 # User documentation
├── CLAUDE.md                 # AI development guide
├── cmd/
│   └── email-manager/
│       └── main.go           # Entry point (minimal)
├── internal/
│   ├── cli/
│   │   └── cli.go            # CLI commands, flags, multi-account logic
│   └── gmail/
│       ├── compose.go        # Email composition (plain text and multipart MIME)
│       └── service.go        # Gmail API service and helpers
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication (multi-account, shared with google-contacts)
```

## Architecture

### Core Packages

1. **cmd/email-manager/main.go** : Minimal entry point, initializes CLI and executes
2. **internal/cli/cli.go** : Command definitions, flag setup, command handlers, account resolution
3. **internal/gmail/compose.go** : Email message composition (BuildPlainMessage, BuildMessageWithAttachments)
4. **internal/gmail/service.go** : Gmail API service wrapper and helper functions
5. **pkg/auth/auth.go** : OAuth2 authentication with multi-account token storage (designed to be duplicated to google-contacts)

### Command Structure

```
email-manager [--account <email>]
├── accounts             # List authenticated accounts
├── auth --account <email> # Authenticate / re-authenticate a specific account
├── send                 # Send emails (with attachment support)
├── list                 # List messages
├── get                  # Get message by ID
├── search               # Search messages
├── read                 # Mark as read
├── unread               # Mark as unread
├── archive              # Archive (remove INBOX label)
├── trash                # Move message to trash
├── untrash              # Restore from trash
├── spam                 # Mark as spam
├── not-spam             # Remove spam label
├── download-attachments # Download message attachments
├── labels
│   ├── list             # List labels
│   ├── create           # Create label
│   ├── apply            # Apply label to message
│   └── remove           # Remove label from message
├── drafts
│   ├── list             # List drafts
│   ├── create           # Create draft (same flags as send)
│   └── delete           # Delete draft
└── skill                # Print agent skill (mode d'emploi for AI agents)
    └── learn            # Persist a learned rule (--rule "...")
```

The binary deliberately exposes only `trash` (reversible) and not a hard
delete: permanent deletion is left to the Gmail web UI.

### Multi-account Logic

- `--account` is a persistent flag on the root command
- `auth` command: `--account` is required, verifies that the OAuth authenticated email matches
- `accounts` command: no account resolution needed
- All other commands: auto-resolves when only one account exists, requires `--account` when multiple accounts exist
- Account resolution happens in `PersistentPreRunE` on the root command

## Key Dependencies

- `github.com/spf13/cobra` : CLI framework
- `google.golang.org/api/gmail/v1` : Gmail API client
- `golang.org/x/oauth2` : OAuth2 authentication
- `github.com/fatih/color` : Terminal colors

## Authentication Flow

1. Reads credentials from `GOOGLE_CREDENTIALS_FILE` env var (falls back to `~/.credentials/google_credentials.json`)
2. Checks for existing token at `~/.cache/email-manager/<account>.json`
3. If no token, initiates OAuth2 flow with browser on port 8002
4. Verifies authenticated email matches `--account` parameter (Inconsistent Authentication error if mismatch)
5. Saves token for future use
6. Creates Gmail service with authenticated HTTP client

### Re-authentication

When the token expires, run:
```bash
email-manager auth --account user@gmail.com
```
This removes the existing token and triggers a fresh OAuth2 flow.

## Credential Sharing Strategy

The `pkg/auth/auth.go` package is designed to be **duplicated** (not shared as a library) to the `google-contacts` project. Both applications:

- Use the same credentials file: `~/.credentials/google_credentials.json`
- Have the same scopes (Gmail + People API) for unified OAuth consent
- Store tokens per-account in `~/.cache/email-manager/`

### Unified OAuth2 Scopes

The auth package includes ALL scopes for both applications:

```go
// Gmail API scopes (for email-manager)
gmail.GmailModifyScope
gmail.GmailSendScope
gmail.GmailLabelsScope

// People API scopes (for google-contacts)
people.ContactsScope
people.ContactsOtherReadonlyScope
```

**Important**: Adding new scopes requires re-authorization per account:
```bash
email-manager auth --account user@gmail.com
```

## Helper Functions (internal/gmail/service.go)

```go
func GetService(ctx context.Context, account string) (*gmail.Service, error)
func ExtractHeaders(headers []*gmail.MessagePartHeader) (subject, from string)
func GetBody(part *gmail.MessagePart) string
func ListMessagesWithDetails(service *gmail.Service, messages []*gmail.Message) error
func ProcessAttachments(service *gmail.Service, messageID string, part *gmail.MessagePart, dir string, count *int) error
func ExpandTilde(path string) (string, error)
```

## Compose Functions (internal/gmail/compose.go)

```go
func BuildPlainMessage(to, cc, bcc, subject, body string) string
func BuildMessageWithAttachments(to, cc, bcc, subject, body string, attachments []string) (string, error)
```

## Auth Functions (pkg/auth/auth.go)

```go
func GetClient(ctx context.Context, account string) (*http.Client, error)
func GetCredentialsFilePath() string
func GetTokenDir() string                          // ~/.cache/email-manager/
func GetTokenPathForAccount(account string) string  // ~/.cache/email-manager/<account>.json
func ListAccounts() ([]string, error)
func RemoveToken(account string) error
```

## Development Workflow

### Build and Test

```bash
make build      # Build binary for current platform
make build-all  # Build for all platforms
make test       # Run tests
make fmt        # Format code
make vet        # Run linter
make check      # All checks
```

### Install/Uninstall

```bash
make install    # Install to /usr/local/bin
make uninstall  # Remove from system
```

### Common Tasks

**Add new command**:
1. Create command variable in `internal/cli/cli.go`
2. Implement `RunE` function
3. Register in `Init()` function with `RootCmd.AddCommand()`
4. All handlers receive `account` from the resolved global variable
5. **Update `internal/cli/skill.md`** so AI agents see the new command (see "Skill maintenance" below)

**Add OAuth scope**:
1. Update `Scopes` slice in `pkg/auth/auth.go`
2. Re-authenticate affected accounts

## File Locations

- **Credentials**: `GOOGLE_CREDENTIALS_FILE` env var or `~/.credentials/google_credentials.json`
- **Tokens**: `~/.cache/email-manager/<account>.json` (one per account)
- **User knowledge**: `~/.config/email-manager/*.md` (or `$XDG_CONFIG_HOME/email-manager/*.md`) — concatenated by `email-manager skill`
- **Binary**: `bin/email-manager-<os>-<arch>` (after build)
- **Installed**: `/usr/local/bin/email-manager` (after install)

## Testing

Recommended test structure:

```
internal/
├── cli/
│   └── cli_test.go
└── gmail/
    ├── compose_test.go
    └── service_test.go
pkg/
└── auth/
    └── auth_test.go
```

## Skill maintenance (CRITICAL)

The binary embeds an agent-facing skill document at
`internal/cli/skill.md` (loaded via `//go:embed`). It is printed by
`email-manager skill` and is what AI agents read to know how to drive the
tool.

**Whenever any of the following changes, the embedded `skill.md` MUST be
updated in the same commit:**

- Adding, renaming, or removing a command or subcommand
- Changing a command flag (name, default, semantics)
- Changing the multi-account resolution logic
- Changing default workflow rules (e.g. INBOX-only scanning)
- Changing destination labels or confirmation policies
- Changing token / credentials / config paths
- Changing the `skill learn` storage format or location

After editing `internal/cli/skill.md`, rebuild and verify with
`./bin/email-manager-<platform> skill | head -50`.

User-specific knowledge lives under `~/.config/email-manager/*.md` (or
`$XDG_CONFIG_HOME/email-manager/*.md`); `email-manager skill` concatenates
those files to the embedded doc at runtime, so they are NOT shipped in the
binary.

## Compliance Checklist

- [x] Remove code duplication (extract common functions)
- [x] Remove `init()` functions
- [x] Fix silent error handling
- [x] Remove dead code
- [x] Reorganize to golang skill structure (cmd/internal/pkg)
- [x] Create Makefile
- [x] Create README.md
- [x] Create CLAUDE.md
- [x] Add People API scopes for unified credentials
- [x] Multi-account support via --account flag
- [x] Functional attachment support in send command
- [x] Trash / untrash / spam / not-spam / labels remove / drafts CRUD
- [x] Self-documenting `skill` command for AI agents
- [ ] Add unit tests
- [ ] Add integration tests

## Notes for AI

- This is a CLI tool, avoid suggesting web/API frameworks
- OAuth2 flow requires user browser interaction
- Gmail API has rate limits, consider batch operations
- Token refresh is handled automatically by oauth2 library
- Always use proper error wrapping with `%w` format
- Follow Go coding standards defined in golang skill
- pkg/auth is designed to be duplicated, not shared as a library
- The `account` parameter flows: CLI flag -> PersistentPreRunE -> global var -> GetService -> GetClient
