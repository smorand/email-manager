// Package cli provides the command-line interface for email-manager.
package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"email-manager/internal/mailer"
	"email-manager/pkg/auth"

	"github.com/spf13/cobra"
	gmailapi "google.golang.org/api/gmail/v1"
)

// Command line flags
var (
	account     string
	attach      []string
	bcc         string
	body        string
	cc          string
	downloadDir string
	htmlBody    string
	htmlFile    string
	maxResults  int64
	query       string
	subject     string
	to          string
)

// RootCmd is the root command for the CLI.
var RootCmd = &cobra.Command{
	Use:   "email-manager",
	Short: "Gmail Manager - Manage Gmail emails",
	Long:  "Send, receive, search, and manage Gmail emails using Gmail API v1",
}

// Command definitions
var (
	accountsCmd = &cobra.Command{
		Use:   "accounts",
		Short: "List authenticated accounts",
		RunE:  runAccounts,
	}

	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Gmail (re-authenticate if token expired)",
		RunE:  runAuth,
	}

	applyLabelCmd = &cobra.Command{
		Use:   "apply <message-id> <label-id>",
		Short: "Apply label to message",
		Args:  cobra.ExactArgs(2),
		RunE:  runApplyLabel,
	}

	archiveCmd = &cobra.Command{
		Use:   "archive <message-id>",
		Short: "Archive a message",
		Args:  cobra.ExactArgs(1),
		RunE:  runArchive,
	}

	createLabelCmd = &cobra.Command{
		Use:   "create <name>",
		Short: "Create a label",
		Args:  cobra.ExactArgs(1),
		RunE:  runCreateLabel,
	}

	trashCmd = &cobra.Command{
		Use:   "trash <message-id>",
		Short: "Move a message to trash",
		Args:  cobra.ExactArgs(1),
		RunE:  runTrash,
	}

	untrashCmd = &cobra.Command{
		Use:   "untrash <message-id>",
		Short: "Restore a message from trash",
		Args:  cobra.ExactArgs(1),
		RunE:  runUntrash,
	}

	spamCmd = &cobra.Command{
		Use:   "spam <message-id>",
		Short: "Mark message as spam",
		Args:  cobra.ExactArgs(1),
		RunE:  runSpam,
	}

	notSpamCmd = &cobra.Command{
		Use:   "not-spam <message-id>",
		Short: "Remove spam label from message",
		Args:  cobra.ExactArgs(1),
		RunE:  runNotSpam,
	}

	removeLabelCmd = &cobra.Command{
		Use:   "remove <message-id> <label-id>",
		Short: "Remove label from message",
		Args:  cobra.ExactArgs(2),
		RunE:  runRemoveLabel,
	}

	draftsCmd = &cobra.Command{
		Use:   "drafts",
		Short: "Manage drafts",
	}

	listDraftsCmd = &cobra.Command{
		Use:   "list",
		Short: "List drafts",
		RunE:  runListDrafts,
	}

	createDraftCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a draft",
		RunE:  runCreateDraft,
	}

	deleteDraftCmd = &cobra.Command{
		Use:   "delete <draft-id>",
		Short: "Delete a draft",
		Args:  cobra.ExactArgs(1),
		RunE:  runDeleteDraft,
	}

	downloadAttachmentsCmd = &cobra.Command{
		Use:   "download-attachments <message-id>",
		Short: "Download attachments from a message",
		Args:  cobra.ExactArgs(1),
		RunE:  runDownloadAttachments,
	}

	getCmd = &cobra.Command{
		Use:   "get <message-id>",
		Short: "Get a message by ID",
		Args:  cobra.ExactArgs(1),
		RunE:  runGet,
	}

	getBodyHTML bool

	labelsCmd = &cobra.Command{
		Use:   "labels",
		Short: "Manage labels",
	}

	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List messages",
		RunE:  runList,
	}

	listLabelsCmd = &cobra.Command{
		Use:   "list",
		Short: "List all labels",
		RunE:  runListLabels,
	}

	readCmd = &cobra.Command{
		Use:   "read <message-id>",
		Short: "Mark message as read",
		Args:  cobra.ExactArgs(1),
		RunE:  runRead,
	}

	searchCmd = &cobra.Command{
		Use:   "search <query>",
		Short: "Search messages",
		Args:  cobra.ExactArgs(1),
		RunE:  runSearch,
	}

	sendCmd = &cobra.Command{
		Use:   "send",
		Short: "Send an email",
		RunE:  runSend,
	}

	unreadCmd = &cobra.Command{
		Use:   "unread <message-id>",
		Short: "Mark message as unread",
		Args:  cobra.ExactArgs(1),
		RunE:  runUnread,
	}
)

// Commands that skip account resolution
var skipAccountResolution = map[string]bool{
	"accounts": true,
	"help":     true,
	"skill":    true,
	"learn":    true,
}

// ruleText holds the content for `skill learn --rule`
var ruleText string

// Init initializes the CLI commands and flags.
func Init() {
	// Persistent flag for account selection
	RootCmd.PersistentFlags().StringVar(&account, "account", "", "Gmail account email address")

	// Account resolution logic
	RootCmd.PersistentPreRunE = resolveAccount

	// Skip account resolution for accounts command
	accountsCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return nil }

	// Setup command flags
	setupSendFlags()
	setupListFlags()
	setupSearchFlags()
	setupDownloadAttachmentsFlags()
	setupLabelCommands()
	setupDraftsCommands()
	setupSkillCommand()
	getCmd.Flags().BoolVar(&getBodyHTML, "html", false, "Return raw HTML body instead of plain text")

	// Register all commands
	RootCmd.AddCommand(accountsCmd)
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(sendCmd)
	RootCmd.AddCommand(listCmd)
	RootCmd.AddCommand(getCmd)
	RootCmd.AddCommand(searchCmd)
	RootCmd.AddCommand(readCmd)
	RootCmd.AddCommand(unreadCmd)
	RootCmd.AddCommand(archiveCmd)
	RootCmd.AddCommand(trashCmd)
	RootCmd.AddCommand(untrashCmd)
	RootCmd.AddCommand(spamCmd)
	RootCmd.AddCommand(notSpamCmd)
	RootCmd.AddCommand(downloadAttachmentsCmd)
	RootCmd.AddCommand(labelsCmd)
	RootCmd.AddCommand(draftsCmd)
	RootCmd.AddCommand(skillCmd)
}

func resolveAccount(cmd *cobra.Command, args []string) error {
	// Skip resolution for commands that don't need an account
	if skipAccountResolution[cmd.Name()] {
		return nil
	}

	// Auth command requires --account explicitly
	if cmd.Name() == "auth" {
		if account == "" {
			return fmt.Errorf("--account flag is required for auth command")
		}
		return nil
	}

	// If account is already set via flag, use it
	if account != "" {
		return nil
	}

	// Auto-resolve: check how many accounts exist
	accounts, err := auth.ListAccounts()
	if err != nil {
		return fmt.Errorf("error listing accounts: %w", err)
	}

	switch len(accounts) {
	case 0:
		return fmt.Errorf("no authenticated accounts found; run 'email-manager auth --account <email>' first")
	case 1:
		account = accounts[0]
		return nil
	default:
		return fmt.Errorf("multiple accounts found, specify one with --account: %s", strings.Join(accounts, ", "))
	}
}

// Setup functions

func setupDownloadAttachmentsFlags() {
	downloadAttachmentsCmd.Flags().StringVar(&downloadDir, "dir", "~/Downloads", "Download directory")
}

func setupLabelCommands() {
	labelsCmd.AddCommand(listLabelsCmd)
	labelsCmd.AddCommand(createLabelCmd)
	labelsCmd.AddCommand(applyLabelCmd)
	labelsCmd.AddCommand(removeLabelCmd)
}

func setupDraftsCommands() {
	createDraftCmd.Flags().StringVar(&to, "to", "", "Recipient email (required)")
	createDraftCmd.Flags().StringVar(&subject, "subject", "", "Draft subject (required)")
	createDraftCmd.Flags().StringVar(&body, "body", "", "Plain-text draft body (required unless --html/--html-file is set)")
	createDraftCmd.Flags().StringVar(&htmlBody, "html", "", "HTML draft body (inline). Adds a text/plain fallback derived from the HTML unless --body is set")
	createDraftCmd.Flags().StringVar(&htmlFile, "html-file", "", "Path to an HTML file to use as the draft body (for large HTML)")
	createDraftCmd.Flags().StringVar(&cc, "cc", "", "CC recipients (comma-separated)")
	createDraftCmd.Flags().StringVar(&bcc, "bcc", "", "BCC recipients (comma-separated)")
	createDraftCmd.Flags().StringSliceVar(&attach, "attach", []string{}, "Attachment file paths")
	_ = createDraftCmd.MarkFlagRequired("to")
	_ = createDraftCmd.MarkFlagRequired("subject")

	draftsCmd.AddCommand(listDraftsCmd)
	draftsCmd.AddCommand(createDraftCmd)
	draftsCmd.AddCommand(deleteDraftCmd)
}

func setupListFlags() {
	listCmd.Flags().StringVar(&query, "query", "", "Gmail query string")
	listCmd.Flags().Int64Var(&maxResults, "max", 10, "Maximum results")
}

func setupSearchFlags() {
	searchCmd.Flags().Int64Var(&maxResults, "max", 10, "Maximum results")
}

func setupSendFlags() {
	sendCmd.Flags().StringVar(&to, "to", "", "Recipient email (required)")
	sendCmd.Flags().StringVar(&subject, "subject", "", "Email subject (required)")
	sendCmd.Flags().StringVar(&body, "body", "", "Plain-text email body (required unless --html/--html-file is set)")
	sendCmd.Flags().StringVar(&htmlBody, "html", "", "HTML email body (inline). Adds a text/plain fallback derived from the HTML unless --body is set")
	sendCmd.Flags().StringVar(&htmlFile, "html-file", "", "Path to an HTML file to use as the email body (for large HTML)")
	sendCmd.Flags().StringVar(&cc, "cc", "", "CC recipients (comma-separated)")
	sendCmd.Flags().StringVar(&bcc, "bcc", "", "BCC recipients (comma-separated)")
	sendCmd.Flags().StringSliceVar(&attach, "attach", []string{}, "Attachment file paths")
	_ = sendCmd.MarkFlagRequired("to")
	_ = sendCmd.MarkFlagRequired("subject")
}

// Command handler functions (alphabetically ordered)

func runAccounts(cmd *cobra.Command, args []string) error {
	accounts, err := auth.ListAccounts()
	if err != nil {
		return fmt.Errorf("error listing accounts: %w", err)
	}
	if len(accounts) == 0 {
		fmt.Fprintf(os.Stderr, "No authenticated accounts found.\n")
		fmt.Fprintf(os.Stderr, "Run 'email-manager auth --account <email>' to authenticate.\n")
		return nil
	}
	for _, a := range accounts {
		fmt.Println(a)
	}
	return nil
}

func runAuth(cmd *cobra.Command, args []string) error {
	// Remove existing token to force re-authentication
	if err := auth.RemoveToken(account); err != nil {
		return fmt.Errorf("error removing existing token: %w", err)
	}

	// Trigger authentication
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Verify by getting user profile
	profile, err := service.Users.GetProfile("me").Do()
	if err != nil {
		return fmt.Errorf("error verifying authentication: %w", err)
	}

	// Verify that the authenticated account matches the requested account
	if !strings.EqualFold(profile.EmailAddress, account) {
		// Remove the inconsistent token
		_ = auth.RemoveToken(account)
		return fmt.Errorf("inconsistent authentication: authenticated as %s but expected %s", profile.EmailAddress, account)
	}

	fmt.Fprintf(os.Stderr, "Authenticated as: %s\n", profile.EmailAddress)
	return nil
}

func runApplyLabel(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		AddLabelIds: []string{args[1]},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error applying label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Label applied\n")
	return nil
}

func runArchive(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error archiving: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message archived\n")
	return nil
}

func runCreateLabel(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	label := &gmailapi.Label{
		Name: args[0],
	}

	result, err := service.Users.Labels.Create("me", label).Do()
	if err != nil {
		return fmt.Errorf("error creating label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Label created: %s (ID: %s)\n", result.Name, result.Id)
	return nil
}

func runTrash(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	_, err = service.Users.Messages.Trash("me", args[0]).Do()
	if err != nil {
		return fmt.Errorf("error trashing message: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message moved to trash\n")
	return nil
}

func runUntrash(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	_, err = service.Users.Messages.Untrash("me", args[0]).Do()
	if err != nil {
		return fmt.Errorf("error untrashing message: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message restored from trash\n")
	return nil
}

func runSpam(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		AddLabelIds:    []string{"SPAM"},
		RemoveLabelIds: []string{"INBOX"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error marking as spam: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message marked as spam\n")
	return nil
}

func runNotSpam(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		RemoveLabelIds: []string{"SPAM"},
		AddLabelIds:    []string{"INBOX"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error removing spam label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message removed from spam\n")
	return nil
}

func runRemoveLabel(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		RemoveLabelIds: []string{args[1]},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error removing label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Label removed\n")
	return nil
}

func runListDrafts(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	response, err := service.Users.Drafts.List("me").Do()
	if err != nil {
		return fmt.Errorf("error listing drafts: %w", err)
	}

	if len(response.Drafts) == 0 {
		fmt.Fprintf(os.Stderr, "No drafts found\n")
		return nil
	}

	for _, draft := range response.Drafts {
		subjectHdr := ""
		toHdr := ""
		if draft.Message != nil && draft.Message.Payload != nil {
			subjectHdr, toHdr = mailer.ExtractDraftHeaders(draft.Message.Payload.Headers)
		}
		fmt.Printf("ID: %s\n  To: %s\n  Subject: %s\n\n", draft.Id, toHdr, subjectHdr)
	}
	return nil
}

func runCreateDraft(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	html, err := resolveHTMLBody()
	if err != nil {
		return err
	}
	if body == "" && html == "" {
		return errNoBody
	}

	raw, err := mailer.BuildMIMEMessage(mailer.MessageInput{
		From: resolveFrom(service), To: to, Cc: cc, Bcc: bcc,
		Subject: subject, Plain: body, HTML: html, Attachments: attach,
	})
	if err != nil {
		return fmt.Errorf("error building draft: %w", err)
	}

	draft := &gmailapi.Draft{Message: &gmailapi.Message{Raw: raw}}

	created, err := service.Users.Drafts.Create("me", draft).Do()
	if err != nil {
		return fmt.Errorf("error creating draft: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Draft created (ID: %s)\n", created.Id)
	return nil
}

func runDeleteDraft(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	if err := service.Users.Drafts.Delete("me", args[0]).Do(); err != nil {
		return fmt.Errorf("error deleting draft: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Draft deleted\n")
	return nil
}

func runDownloadAttachments(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	messageID := args[0]

	// Get the message
	msg, err := service.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return fmt.Errorf("error getting message: %w", err)
	}

	// Expand tilde in download directory
	dir, err := mailer.ExpandTilde(downloadDir)
	if err != nil {
		return err
	}

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(dir, mailer.AttachmentDirPerm); err != nil {
		return fmt.Errorf("error creating download directory: %w", err)
	}

	// Process attachments
	attachmentCount := 0
	if err := mailer.ProcessAttachments(service, messageID, msg.Payload, dir, &attachmentCount); err != nil {
		return err
	}

	if attachmentCount == 0 {
		fmt.Fprintf(os.Stderr, "No attachments found\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "Downloaded %d attachment(s) to %s\n", attachmentCount, dir)
	return nil
}

func runGet(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	msg, err := service.Users.Messages.Get("me", args[0]).Do()
	if err != nil {
		return fmt.Errorf("error getting message: %w", err)
	}

	// Print headers
	for _, header := range msg.Payload.Headers {
		if header.Name == "From" || header.Name == "To" || header.Name == "Subject" || header.Name == "Date" {
			fmt.Printf("%s: %s\n", header.Name, header.Value)
		}
	}

	// Print body
	fmt.Println("\n" + strings.Repeat("=", 80))
	var body string
	if getBodyHTML {
		body = mailer.GetHTMLBody(msg.Payload)
	} else {
		body = mailer.GetBody(msg.Payload)
	}
	fmt.Println(body)

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	call := service.Users.Messages.List("me").MaxResults(maxResults)
	if query != "" {
		call = call.Q(query)
	}

	response, err := call.Do()
	if err != nil {
		return fmt.Errorf("error listing messages: %w", err)
	}

	return mailer.ListMessagesWithDetails(service, response.Messages)
}

func runListLabels(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	response, err := service.Users.Labels.List("me").Do()
	if err != nil {
		return fmt.Errorf("error listing labels: %w", err)
	}

	for _, label := range response.Labels {
		fmt.Printf("%s (ID: %s)\n", label.Name, label.Id)
	}

	return nil
}

func runRead(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error marking as read: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message marked as read\n")
	return nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	response, err := service.Users.Messages.List("me").Q(args[0]).MaxResults(maxResults).Do()
	if err != nil {
		return fmt.Errorf("error searching: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d messages\n\n", len(response.Messages))

	return mailer.ListMessagesWithDetails(service, response.Messages)
}

func runSend(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	html, err := resolveHTMLBody()
	if err != nil {
		return err
	}
	if body == "" && html == "" {
		return errNoBody
	}

	raw, err := mailer.BuildMIMEMessage(mailer.MessageInput{
		From: resolveFrom(service), To: to, Cc: cc, Bcc: bcc,
		Subject: subject, Plain: body, HTML: html, Attachments: attach,
	})
	if err != nil {
		return fmt.Errorf("error building message: %w", err)
	}

	msg := &gmailapi.Message{Raw: raw}
	if _, err := service.Users.Messages.Send("me", msg).Do(); err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Email sent successfully to %s\n", to)
	return nil
}

// errNoBody is returned when neither a plain-text nor an HTML body is provided.
var errNoBody = errors.New("a body is required: pass --body, --html, or --html-file")

// resolveHTMLBody returns the HTML body from --html or --html-file. The two
// flags are mutually exclusive; --html-file is read from disk (with ~ expanded).
func resolveHTMLBody() (string, error) {
	if htmlBody != "" && htmlFile != "" {
		return "", errors.New("use either --html or --html-file, not both")
	}
	if htmlFile == "" {
		return htmlBody, nil
	}
	path, err := mailer.ExpandTilde(htmlFile)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("unable to read --html-file %s: %w", htmlFile, err)
	}
	return string(data), nil
}

// resolveFrom returns the From header value (raw "Name <email>") for the
// authenticated user, or an empty string if it cannot be retrieved (e.g.
// missing gmail.settings.basic scope on a token issued before the scope was
// added). When empty, the message is sent without an explicit From and Gmail
// auto-injects it (which can produce mojibake for non-ASCII display names —
// the user is warned and pointed at the re-auth command).
func resolveFrom(service *gmailapi.Service) string {
	from, err := mailer.GetDefaultFrom(service)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"warning: unable to fetch sender display name (%v).\n"+
				"         If the recipient sees garbled characters in your name,\n"+
				"         re-authenticate to pick up the new scope:\n"+
				"           email-manager auth --account %s\n",
			err, account)
		return ""
	}
	return from
}

func runUnread(cmd *cobra.Command, args []string) error {
	service, err := mailer.GetService(cmd.Context(), account)
	if err != nil {
		return err
	}

	req := &gmailapi.ModifyMessageRequest{
		AddLabelIds: []string{"UNREAD"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error marking as unread: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message marked as unread\n")
	return nil
}
