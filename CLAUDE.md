# Email Manager - AI Development Guide

## Project Overview

**Type**: CLI Application
**Language**: Go 1.25+
**Purpose**: Gmail management via Gmail API v1
**Authentication**: OAuth2 with Google
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
│   │   └── cli.go            # CLI commands and flags
│   └── gmail/
│       └── service.go        # Gmail API service and helpers
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication (shared with google-contacts)
```

## Architecture

### Core Packages

1. **cmd/email-manager/main.go** - Minimal entry point, initializes CLI and executes
2. **internal/cli/cli.go** - Command definitions, flag setup, command handlers
3. **internal/gmail/service.go** - Gmail API service wrapper and helper functions
4. **pkg/auth/auth.go** - OAuth2 authentication (designed to be duplicated to google-contacts)

### Command Structure

```
email-manager
├── send                 # Send emails
├── list                 # List messages
├── get                  # Get message by ID
├── search               # Search messages
├── read                 # Mark as read
├── unread               # Mark as unread
├── archive              # Archive message
├── delete               # Delete message
├── download-attachments # Download message attachments
└── labels
    ├── list             # List labels
    ├── create           # Create label
    └── apply            # Apply label to message
```

## Key Dependencies

- `github.com/spf13/cobra` - CLI framework
- `google.golang.org/api/gmail/v1` - Gmail API client
- `golang.org/x/oauth2` - OAuth2 authentication
- `github.com/fatih/color` - Terminal colors

## Authentication Flow

1. Reads credentials from `~/.credentials/google_credentials.json`
2. Checks for existing token at `~/.credentials/google_token.json`
3. If no token, initiates OAuth2 flow with browser
4. Saves token for future use
5. Creates Gmail service with authenticated HTTP client

## Credential Sharing Strategy

The `pkg/auth/auth.go` package is designed to be **duplicated** (not shared as a library) to the `google-contacts` project. Both applications will:

- Use the same token file: `~/.credentials/google_token.json`
- Use the same credentials file: `~/.credentials/google_credentials.json`
- Have the same scopes (Gmail + People API) for unified OAuth consent

This enables users to authorize once and use both applications.

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

**Important**: Adding new scopes requires re-authorization. Delete the token file to force re-auth:
```bash
rm ~/.credentials/google_token.json
```

## Helper Functions (internal/gmail/service.go)

```go
// GetService - Returns Gmail API service instance
func GetService(ctx context.Context) (*gmail.Service, error)

// ExtractHeaders - Extracts subject and from headers from message
func ExtractHeaders(headers []*gmail.MessagePartHeader) (subject, from string)

// GetBody - Extracts text body from message payload
func GetBody(part *gmail.MessagePart) string

// ListMessagesWithDetails - Lists messages with full details (from, subject)
func ListMessagesWithDetails(service *gmail.Service, messages []*gmail.Message) error

// ProcessAttachments - Recursively processes message parts to download attachments
func ProcessAttachments(service *gmail.Service, messageID string, part *gmail.MessagePart, dir string, count *int) error

// ExpandTilde - Expands ~ to user's home directory
func ExpandTilde(path string) (string, error)
```

## CLI Setup Functions (internal/cli/cli.go)

```go
func Init()                          // Initializes all commands and flags
func setupSendFlags()                // Configures send command flags
func setupListFlags()                // Configures list command flags
func setupSearchFlags()              // Configures search command flags
func setupDownloadAttachmentsFlags() // Configures download-attachments flags
func setupLabelCommands()            // Registers label subcommands
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

**Add OAuth scope**:
1. Update `Scopes` slice in `pkg/auth/auth.go`
2. Delete existing token to re-authenticate

## File Locations

- **Credentials**: `~/.credentials/google_credentials.json`
- **Token**: `~/.credentials/google_token.json`
- **Binary**: `bin/email-manager-<os>-<arch>` (after build)
- **Installed**: `/usr/local/bin/email-manager` (after install)

## Testing

Recommended test structure:

```
internal/
├── cli/
│   └── cli_test.go
└── gmail/
    └── service_test.go
pkg/
└── auth/
    └── auth_test.go
```

## Compliance Checklist

- [x] Remove code duplication (extract common functions)
- [x] Remove `init()` functions
- [x] Fix silent error handling
- [x] Remove dead code
- [x] Reorganize to golang skill structure (cmd/internal/pkg)
- [x] Create Makefile
- [x] Create README.md
- [x] Create CLAUDE.md
- [ ] Add unit tests
- [ ] Add integration tests
- [x] Add People API scopes for unified credentials (US-00002)

## Notes for AI

- This is a CLI tool, avoid suggesting web/API frameworks
- OAuth2 flow requires user browser interaction
- Gmail API has rate limits - consider batch operations
- Token refresh is handled automatically by oauth2 library
- Always use proper error wrapping with `%w` format
- Follow Go coding standards defined in golang skill
- pkg/auth is designed to be duplicated, not shared as a library
