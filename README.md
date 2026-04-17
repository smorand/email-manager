# Email Manager

A command-line interface (CLI) tool for managing Gmail emails using the Gmail API v1. Supports multiple Gmail accounts.

## Features

- Multi-account support: manage multiple Gmail accounts with `--account` flag
- Send emails with CC, BCC, and attachments
- List and search messages
- Mark messages as read/unread
- Archive and delete messages
- Download message attachments
- Manage Gmail labels
- OAuth2 authentication with Google

## Prerequisites

- Go 1.25 or higher
- Google Cloud Project with Gmail API enabled
- OAuth2 credentials (credentials.json)

## Installation

### Build from source

```bash
make build
```

### Build for all platforms

```bash
make build-all
```

### Install to system

```bash
make install
```

This will install the binary to `/usr/local/bin`. You can specify a custom location:

```bash
TARGET=/custom/path make install
```

### Uninstall

```bash
make uninstall
```

## Setup

1. Create a Google Cloud Project and enable Gmail API and People API
2. Create OAuth2 credentials (Web application) with redirect URI `http://localhost:8002/oauth2callback`
3. Download the credentials JSON file
4. Set the `GOOGLE_CREDENTIALS_FILE` environment variable to point to your credentials file:
   ```bash
   export GOOGLE_CREDENTIALS_FILE=~/.credentials/scm-pwd-web.json
   ```
   Alternatively, save credentials to the default location: `~/.credentials/google_credentials.json`
5. Authenticate your Gmail account:
   ```bash
   email-manager auth --account user@gmail.com
   ```
6. The token will be saved to `~/.cache/email-manager/user@gmail.com.json`

### Multi-account Setup

You can authenticate multiple Gmail accounts:

```bash
email-manager auth --account personal@gmail.com
email-manager auth --account work@gmail.com
```

When only one account is authenticated, it is selected automatically. When multiple accounts are authenticated, use `--account` to specify which one:

```bash
email-manager list --account personal@gmail.com
```

List all authenticated accounts:

```bash
email-manager accounts
```

### Re-authentication

If your token expires, re-authenticate the specific account:
```bash
email-manager auth --account user@gmail.com
```
This removes the old token and opens a browser for fresh authentication. The tool verifies that the account you authenticate with matches the `--account` value.

### Credential Sharing with google-contacts

This application shares OAuth credentials (client_id/secret) with the `google-contacts` project. Both applications use:
- Same credentials file: `~/.credentials/google_credentials.json`
- Combined scopes: Gmail API + People API

Token files are stored per-account in `~/.cache/email-manager/`.

## Usage

### Authenticate

```bash
email-manager auth --account user@gmail.com
```

### List Authenticated Accounts

```bash
email-manager accounts
```

### Send Email

```bash
# Simple email
email-manager send --to "recipient@example.com" --subject "Hello" --body "Message content"

# With CC and BCC
email-manager send --to "recipient@example.com" --subject "Test" --body "Message" --cc "cc@example.com" --bcc "bcc@example.com"

# With attachments
email-manager send --to "recipient@example.com" --subject "Report" --body "See attached" --attach /path/to/file.pdf
email-manager send --to "recipient@example.com" --subject "Files" --body "Multiple files" --attach file1.pdf --attach file2.png

# With specific account
email-manager send --account work@gmail.com --to "recipient@example.com" --subject "Hello" --body "Message"
```

### List Messages

```bash
# List recent messages
email-manager list

# List with query
email-manager list --query "is:unread"

# List with custom max results
email-manager list --max 20
```

### Search Messages

```bash
email-manager search "from:sender@example.com"
email-manager search "subject:meeting" --max 5
```

### Get Message

```bash
email-manager get <message-id>
```

### Mark as Read/Unread

```bash
email-manager read <message-id>
email-manager unread <message-id>
```

### Archive Message

```bash
email-manager archive <message-id>
```

### Delete Message

```bash
email-manager delete <message-id>
```

### Download Attachments

```bash
# Download attachments to default location (~/Downloads)
email-manager download-attachments <message-id>

# Download to custom directory
email-manager download-attachments <message-id> --dir /path/to/directory
```

### Manage Labels

```bash
# List all labels
email-manager labels list

# Create a label
email-manager labels create "MyLabel"

# Apply label to message
email-manager labels apply <message-id> <label-id>
```

## Development

### Run tests

```bash
make test
```

### Format code

```bash
make fmt
```

### Run linter

```bash
make vet
```

### Run all checks

```bash
make check
```

### Clean build artifacts

```bash
make clean
```

### Rebuild from scratch

```bash
make rebuild
```

## Project Structure

```
email-manager/
├── go.mod                    # Go module dependencies
├── go.sum                    # Dependency checksums
├── Makefile                  # Build automation
├── README.md                 # This file
├── CLAUDE.md                 # AI development guide
├── cmd/
│   └── email-manager/
│       └── main.go           # Entry point
├── internal/
│   ├── cli/
│   │   └── cli.go            # CLI command implementations
│   └── gmail/
│       ├── compose.go        # Email composition (plain and multipart MIME)
│       └── service.go        # Gmail API service
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication (multi-account)
```

## File Locations

- **Credentials**: `GOOGLE_CREDENTIALS_FILE` env var or `~/.credentials/google_credentials.json`
- **Tokens**: `~/.cache/email-manager/<account>.json` (one per account)
- **Binary**: `bin/email-manager-<os>-<arch>` (after build)

## License

Private project for internal use.

## Author

Sebastien MORAND (seb.morand@gmail.com)
