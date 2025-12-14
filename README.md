# Email Manager

A command-line interface (CLI) tool for managing Gmail emails using the Gmail API v1.

## Features

- Send emails with CC, BCC, and attachments
- List and search messages
- Mark messages as read/unread
- Archive and delete messages
- Download message attachments
- Manage Gmail labels
- OAuth2 authentication with Google

## Prerequisites

- Go 1.21 or higher
- Google Cloud Project with Gmail API enabled
- OAuth2 credentials (credentials.json)

## Installation

### Build from source

```bash
make build
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

1. Create a Google Cloud Project and enable Gmail API
2. Create OAuth2 credentials (Desktop application)
3. Download the credentials and save to `~/.credentials/google_credentials.json`
4. Run any command - you'll be prompted to authorize the application
5. The token will be saved to `~/.credentials/google_token.json`

## Usage

### Send Email

```bash
email-manager send --to "recipient@example.com" --subject "Hello" --body "Message content"
email-manager send --to "recipient@example.com" --subject "Test" --body "Message" --cc "cc@example.com" --bcc "bcc@example.com"
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
├── src/
│   ├── main.go      # Main entry point and command registration
│   ├── cli.go       # CLI command implementations
│   ├── auth.go      # OAuth2 authentication logic
│   ├── go.mod       # Go module dependencies
│   └── go.sum       # Dependency checksums
├── Makefile         # Build automation
├── README.md        # This file
└── CLAUDE.md        # AI-oriented documentation
```

## License

Private project for internal use.

## Author

Sebastien MORAND (sebastien.morand@loreal.com)
