# Email Manager

A command-line interface (CLI) tool for managing Gmail emails using the Gmail API v1. Supports multiple Gmail accounts.

## Features

- Multi-account support: manage multiple Gmail accounts with `--account` flag
- Send emails with CC, BCC, and attachments
- Manage drafts (list, create with attachments, delete)
- List and search messages
- Mark messages as read/unread
- Archive, trash, untrash messages
- Mark / unmark spam
- Download message attachments
- Manage Gmail labels (list, create, apply, remove)
- OAuth2 authentication with Google
- Self-documenting via `email-manager skill` for AI agent integration

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
   export GOOGLE_CREDENTIALS_FILE=~/.credentials/google_credentials.json
   ```
   Alternatively, the binary falls back to `~/.credentials/google_credentials.json`.
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

### Trash / Untrash

```bash
email-manager trash <message-id>
email-manager untrash <message-id>
```

The binary deliberately does not expose a permanent delete: trash is reversible
via `untrash`, hard-delete is left to the Gmail web UI.

### Spam / Not-spam

```bash
email-manager spam <message-id>
email-manager not-spam <message-id>
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

# Remove label from message
email-manager labels remove <message-id> <label-id>
```

### Manage Drafts

```bash
# List drafts
email-manager drafts list

# Create a draft (same flags as send)
email-manager drafts create --to "recipient@example.com" --subject "Hello" --body "Draft content"
email-manager drafts create --to "..." --subject "..." --body "..." --attach file.pdf

# Delete a draft
email-manager drafts delete <draft-id>
```

## Use with AI agent coding tools

`email-manager` is self-documenting for AI agents (Claude Code, Cursor,
Aider, ...). The binary embeds a complete usage guide accessible via:

```bash
email-manager skill
```

To make any project's AI agent aware of `email-manager`, add a single line
to that project's `CLAUDE.md` (or `AGENTS.md`):

```markdown
For Gmail email management, run `email-manager skill` to get the full usage
guide and workflows.
```

The agent will execute `email-manager skill` on demand, retrieving:

- The current command list
- The multi-account workflow
- Default rules (INBOX-only scanning, destination labels, confirmation
  policies)
- Any user-specific knowledge stored under `~/.config/email-manager/*.md`

### Teaching new rules to the agent

When the user expresses a permanent preference ("emails from X should always
be archived"), the agent should persist it via:

```bash
email-manager skill learn --rule "emails from noreply@example.com -> archive + label personal/commandes"
```

This appends the rule to `~/.config/email-manager/regles-tri.md`, which is
automatically concatenated to subsequent `email-manager skill` calls so the
agent picks up the new rule on the next invocation.

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
│   │   ├── cli.go            # CLI command implementations
│   │   ├── skill.go          # 'skill' / 'skill learn' commands
│   │   └── skill.md          # Embedded agent skill (//go:embed)
│   └── mailer/
│       ├── compose.go        # Email composition (plain and multipart MIME)
│       └── service.go        # Gmail API service
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication (multi-account)
```

## File Locations

- **Credentials**: `GOOGLE_CREDENTIALS_FILE` env var or `~/.credentials/google_credentials.json`
- **Tokens**: `~/.cache/email-manager/<account>.json` (one per account)
- **User knowledge** (read by `email-manager skill`): `~/.config/email-manager/*.md` (or `$XDG_CONFIG_HOME/email-manager/*.md`)
- **Binary**: `bin/email-manager-<os>-<arch>` (after build)

## License

Private project for internal use.

## Author

Sebastien MORAND (seb.morand@gmail.com)
