# Email Manager - AI Development Guide

## Project Overview

**Type**: CLI Application
**Language**: Go 1.25+
**Purpose**: Gmail and Google Calendar management via Gmail API v1 and Calendar API v3 (multi-account)
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
│   │   ├── cli.go            # Mail CLI commands, flags, multi-account logic
│   │   ├── calendar.go       # `cal` command tree (Google Calendar)
│   │   ├── skill.go          # 'skill' / 'skill learn' commands
│   │   └── skill.md          # Embedded mode d'emploi (//go:embed)
│   ├── mailer/
│   │   ├── compose.go        # Email composition (plain text and multipart MIME)
│   │   └── service.go        # Gmail API service and helpers
│   └── calendar/
│       ├── event.go          # EventInput -> *calendar.Event, partial-patch application
│       └── service.go        # Calendar API service and helpers (formatting, notify validation)
└── pkg/
    └── auth/
        └── auth.go           # OAuth2 authentication (multi-account, shared with google-contacts)
```

## Architecture

### Core Packages

1. **cmd/email-manager/main.go** : Minimal entry point, initializes CLI and executes
2. **internal/cli/cli.go** : Mail command definitions, flag setup, command handlers, account resolution
3. **internal/cli/calendar.go** : `cal` command tree (calendars/list/get/instances/add/update/delete/respond/quick-add/freebusy)
4. **internal/mailer/compose.go** : Email message composition (BuildMIMEMessage: text/plain, HTML multipart/alternative, and multipart/mixed with attachments)
5. **internal/mailer/service.go** : Gmail API service wrapper and helper functions
6. **internal/calendar/event.go** : `EventInput.ToEvent()` builds a new `*calendar.Event` (recurrence, attendees, reminders, transparency, Meet conferenceData); `ApplyPatch()` mutates only the cobra-`Changed()` fields for partial PATCH semantics
7. **internal/calendar/service.go** : `GetService`, `ValidateNotify`, and display helpers (`FormatEventLine`, `MeetLink`, `EventTime`, `FindSelfAttendee`)
8. **pkg/auth/auth.go** : OAuth2 authentication with multi-account token storage (designed to be duplicated to google-contacts)

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
├── cal [--calendar-id primary]
│   ├── calendars list   # List calendars visible to the account
│   ├── list             # List events in a time window (--start/--end RFC3339)
│   ├── get <event-id>   # Event details (attendees, Meet link, recurrence, reminders)
│   ├── instances <event-id>  # Occurrences of a recurring event
│   ├── add              # Create an event (recurrence, attendees, reminders, --meet, ...)
│   ├── update <event-id>     # Partial PATCH: only explicitly-passed flags are applied
│   ├── delete <event-id>     # Permanent delete, no undo (Calendar API has none)
│   ├── respond <event-id>    # Accept/decline/tentative on an invitation
│   ├── quick-add        # Create an event from natural language text (Google NLP)
│   └── freebusy          # Query busy time slots across one or more calendars
└── skill                # Print agent skill (mode d'emploi for AI agents)
    └── learn            # Persist a learned rule (--rule "...")
```

The binary deliberately exposes only `trash` (reversible) and not a hard
delete for mail: permanent deletion is left to the Gmail web UI. `cal delete`
has no such reversible alternative (the Calendar API itself has no undo), so
it is a real permanent delete — treat it with the same caution as a hard
delete would warrant.

`cal add`/`update`/`delete`/`respond` default `--notify` to `none`: no
invitation/notification email is sent to attendees unless `--notify all` (or
`externalOnly`) is passed explicitly.

Every `cal` subcommand also accepts `--json` (persistent flag on `calCmd`):
prints the raw Calendar API struct(s) via `printCalJSON` (encoding/json on
the `google.golang.org/api/calendar/v3` types directly, which already carry
`json:"..."` tags) instead of the human-readable text format. `list`/
`instances`/`calendars list` emit a JSON array; `get`/`add`/`update`/
`respond`/`quick-add` emit a single event object; `freebusy` emits the raw
freebusy response; `delete` (which has no response body) emits
`{"id":..., "deleted": true}`.

### Multi-account Logic

- `--account` is a persistent flag on the root command
- `auth` command: `--account` is required, verifies that the OAuth authenticated email matches
- `accounts` command: no account resolution needed
- All other commands: auto-resolves when only one account exists, requires `--account` when multiple accounts exist
- Account resolution happens in `PersistentPreRunE` on the root command

## Key Dependencies

- `github.com/spf13/cobra` : CLI framework
- `google.golang.org/api/gmail/v1` : Gmail API client
- `google.golang.org/api/calendar/v3` : Calendar API client
- `github.com/google/uuid` : Conference (Google Meet) create-request IDs
- `golang.org/x/oauth2` : OAuth2 authentication

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

The auth package includes ALL scopes for both applications, plus Calendar:

```go
// Gmail API scopes (for email-manager mail commands)
gmail.GmailModifyScope
gmail.GmailSendScope
gmail.GmailLabelsScope
gmail.GmailSettingsBasicScope

// People API scopes (for google-contacts)
people.ContactsScope
people.ContactsOtherReadonlyScope

// Calendar API scope (for email-manager cal commands)
calendar.CalendarScope
```

`calendar.CalendarScope` (full access, not events-only) is required because
`freebusy`, `quick-add`, and `calendarList` all fall outside the narrower
events scope.

**Important**: Adding new scopes requires re-authorization per account:
```bash
email-manager auth --account user@gmail.com
```

## Helper Functions (internal/mailer/service.go)

```go
func GetService(ctx context.Context, account string) (*gmail.Service, error)
func ExtractHeaders(headers []*gmail.MessagePartHeader) (subject, from string)
func GetBody(part *gmail.MessagePart) string
func ListMessagesWithDetails(service *gmail.Service, messages []*gmail.Message) error
func ProcessAttachments(service *gmail.Service, messageID string, part *gmail.MessagePart, dir string, count *int) error
func ExpandTilde(path string) (string, error)
```

## Compose Functions (internal/mailer/compose.go)

```go
// MessageInput carries From/To/Cc/Bcc/Subject, Plain and HTML bodies, and Attachments.
// BuildMIMEMessage picks the simplest MIME shape: text/plain; multipart/alternative
// {text/plain, text/html} when HTML is set (a plain fallback is derived from the HTML
// via HTMLToText when Plain is empty); wrapped in multipart/mixed when attachments exist.
func BuildMIMEMessage(in MessageInput) (string, error)
```

`send` and `drafts create` expose `--body` (plain), `--html` (inline HTML) and
`--html-file` (HTML from a path); at least one body is required, `--html`/`--html-file`
are mutually exclusive.

## Auth Functions (pkg/auth/auth.go)

```go
func GetClient(ctx context.Context, account string) (*http.Client, error)
func GetCredentialsFilePath() string
func GetTokenDir() string                          // ~/.cache/email-manager/
func GetTokenPathForAccount(account string) string  // ~/.cache/email-manager/<account>.json
func ListAccounts() ([]string, error)
func RemoveToken(account string) error
```

## Calendar Functions (internal/calendar)

```go
// service.go
func GetService(ctx context.Context, account string) (*calendar.Service, error)
func ValidateNotify(notify string) error   // "none" | "all" | "externalOnly"
func EventTime(dt *calendar.EventDateTime) string
func FormatEventLine(ev *calendar.Event) string
func MeetLink(ev *calendar.Event) string
func FindSelfAttendee(ev *calendar.Event, account string) *calendar.EventAttendee

// event.go
type EventInput struct { /* Summary, Start/End, AllDay, TimeZone, Attendees, Recurrence, ReminderMinutes, ColorID, Visibility, Busy *bool, ConferenceMeet */ }
func (in EventInput) ToEvent() *calendar.Event                                  // used by `cal add`
func ApplyPatch(ev *calendar.Event, in EventInput, changed Changed) *calendar.Event  // used by `cal update`, only mutates cobra-Changed() fields
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

**Add new mail command**:
1. Create command variable in `internal/cli/cli.go`
2. Implement `RunE` function
3. Register in `Init()` function with `RootCmd.AddCommand()`
4. All handlers receive `account` from the resolved global variable
5. **Update `internal/cli/skill.md`** so AI agents see the new command (see "Skill maintenance" below)

**Add new `cal` subcommand**:
1. Create command variable + flags in `internal/cli/calendar.go`
2. Implement `RunE` function using `gcalsvc.GetService(cmd.Context(), account)`
3. Register under `calCmd` in `setupCalendarCommands()`
4. If it builds/patches an event, extend `EventInput`/`ToEvent`/`ApplyPatch` in `internal/calendar/event.go` rather than constructing `*calendar.Event` inline
5. **Update `internal/cli/skill.md`** ("Agenda (Google Calendar)" section) in the same commit

**Add OAuth scope**:
1. Update `Scopes` slice in `pkg/auth/auth.go`
2. Re-authenticate affected accounts: `email-manager auth --account <email>`

## File Locations

- **Credentials**: `GOOGLE_CREDENTIALS_FILE` env var or `~/.credentials/google_credentials.json`
- **Tokens**: `~/.cache/email-manager/<account>.json` (one per account)
- **User knowledge**: `~/.config/email-manager/*.md` (or `$XDG_CONFIG_HOME/email-manager/*.md`) — concatenated by `email-manager skill`
- **Binary**: `bin/email-manager-<os>-<arch>` (after build)
- **Installed**: `/usr/local/bin/email-manager` (after install)

## Testing

Current test coverage:

```
internal/
├── mailer/
│   └── compose_test.go     # MIME building (plain/HTML/attachments)
└── calendar/
    ├── event_test.go       # EventInput.ToEvent(), ApplyPatch(), FindSelfAttendee()
    └── service_test.go     # ValidateNotify, EventTime, MeetLink, FormatEventLine
```

No live network mocking exists for `gmail/v1` or `calendar/v3` (unlike
outlook-tool's `respx`); tests target pure functions only (MIME/event
building, formatting, validation). `internal/cli` has no unit tests — verify
CLI wiring manually via `--help` and against a real account.

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
- [x] Canonical Makefile (golang skill reference)
- [x] LICENSE (MIT)
- [x] golangci-lint configured (`.golangci.yml`)
- [x] Package renamed `internal/gmail` -> `internal/mailer` (collision fix)
- [x] Magic file modes extracted to constants
- [x] OAuth callback race fixed (listener-based, no Sleep)
- [x] Use `cmd.Context()` instead of `context.Background()`
- [x] Add unit tests (pure functions: compose, calendar event building/patching)
- [ ] Add integration tests
- [x] Google Calendar support (`cal` command tree, see Command Structure above)

## Notes for AI

- This is a CLI tool, avoid suggesting web/API frameworks
- OAuth2 flow requires user browser interaction
- Gmail and Calendar APIs both have rate limits, consider batch operations
- Token refresh is handled automatically by oauth2 library (same token file/scopes cover both APIs)
- `cal delete` and `--notify all` are irreversible/user-visible side effects: always confirm with the user first (see skill.md)
- Always use proper error wrapping with `%w` format
- Follow Go coding standards defined in golang skill
- pkg/auth is designed to be duplicated, not shared as a library
- The `account` parameter flows: CLI flag -> PersistentPreRunE -> global var -> GetService -> GetClient
