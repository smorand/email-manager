# Email Manager - AI Development Guide

## Project Overview

**Type**: CLI Application
**Language**: Go 1.21+
**Purpose**: Gmail management via Gmail API v1
**Authentication**: OAuth2 with Google
**CLI Framework**: Cobra

## Architecture

### Core Components

1. **main.go** - Entry point, command registration
2. **cli.go** - Command implementations (send, list, get, search, read, unread, archive, delete, download-attachments, labels)
3. **auth.go** - OAuth2 authentication and Gmail service initialization

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

## Compliance Status

### ✅ All Critical Issues Resolved

The codebase is now fully compliant with Go coding standards:

1. **✅ Code Duplication Eliminated**
   - Extracted `extractHeaders()` helper function
   - Extracted `listMessagesWithDetails()` for common message listing logic
   - Both `runList()` and `runSearch()` now use shared helper

2. **✅ init() Functions Removed**
   - Replaced with explicit setup functions: `setupSendFlags()`, `setupListFlags()`, `setupSearchFlags()`, `setupLabelCommands()`
   - All setup called explicitly in `main()`

3. **✅ Error Handling Improved**
   - Silent errors now logged with `fmt.Fprintf(os.Stderr, "Warning: ...")`
   - Proper error context provided to users

4. **✅ Dead Code Removed**
   - Deleted unused `printJSON()` function

5. **✅ File Organization Compliant**
   - Proper ordering: package → imports → constants/vars → command definitions → setup functions → handlers (alphabetical) → helpers (alphabetical)
   - All functions and variables alphabetically ordered within their sections

6. **✅ Context Usage**
   - Using `context.Background()` is acceptable for CLI tools
   - Each command handler creates its own context as needed

## Code Structure

### Helper Functions

The codebase uses the following helper functions to eliminate duplication:

```go
// extractHeaders - Extracts subject and from headers from message
func extractHeaders(headers []*gmail.MessagePartHeader) (subject, from string)

// listMessagesWithDetails - Lists messages with full details (from, subject)
func listMessagesWithDetails(service *gmail.Service, messages []*gmail.Message) error

// getBody - Extracts text body from message payload
func getBody(part *gmail.MessagePart) string

// processAttachments - Recursively processes message parts to download attachments
func processAttachments(service *gmail.Service, messageID string, part *gmail.MessagePart, dir string, count *int) error
```

### Setup Functions

All command configuration is done through explicit setup functions:

```go
func setupSendFlags()                // Configures send command flags
func setupListFlags()                // Configures list command flags
func setupSearchFlags()              // Configures search command flags
func setupDownloadAttachmentsFlags() // Configures download-attachments flags
func setupLabelCommands()            // Registers label subcommands
```

## Development Workflow

### Build and Test

```bash
make build      # Build binary
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
1. Create command variable in `cli.go`
2. Implement `RunE` function
3. Register in `main.go` with `rootCmd.AddCommand()`

**Add OAuth scope**:
1. Update `scopes` slice in `auth.go`
2. Delete existing token to re-authenticate

## File Locations

- **Credentials**: `~/.credentials/google_credentials.json`
- **Token**: `~/.credentials/google_token.json`
- **Binary**: `/usr/local/bin/email-manager` (after install)

## Testing

Currently no tests implemented. Recommended test structure:

```
src/
├── main_test.go
├── cli_test.go
└── auth_test.go
```

## Future Improvements

1. Add unit tests for all commands
2. Implement attachment support in send command (upload)
3. Add batch operations (bulk read/archive/delete)
4. Add configuration file support (~/.email-manager.yaml)
5. Implement proper logging with levels
6. Add progress bars for batch operations
7. Support HTML email bodies
8. Add draft management commands
9. List attachments without downloading them
10. Download specific attachments by index or name

## Compliance Checklist

- [x] Remove code duplication (extract common functions)
- [x] Remove `init()` functions
- [x] Fix silent error handling
- [x] Remove dead code (`printJSON`)
- [x] Reorganize file element ordering
- [x] Improve context usage
- [x] Create Makefile
- [x] Create README.md
- [x] Create CLAUDE.md
- [ ] Add unit tests
- [ ] Add integration tests

## Notes for AI

- This is a CLI tool, avoid suggesting web/API frameworks
- OAuth2 flow requires user browser interaction
- Gmail API has rate limits - consider batch operations
- Token refresh is handled automatically by oauth2 library
- Always use proper error wrapping with `%w` format
- Follow Go coding standards defined in golang skill
