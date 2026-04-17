// Package cli provides the command-line interface for email-manager.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"email-manager/internal/gmail"
	"email-manager/pkg/auth"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	gmailapi "google.golang.org/api/gmail/v1"
)

// Color functions
var (
	cyan  = color.New(color.FgCyan).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	red   = color.New(color.FgRed).SprintFunc()
)

// Command line flags
var (
	account     string
	attach      []string
	bcc         string
	body        string
	cc          string
	downloadDir string
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

	deleteCmd = &cobra.Command{
		Use:   "delete <message-id>",
		Short: "Delete a message",
		Args:  cobra.ExactArgs(1),
		RunE:  runDelete,
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
}

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
	RootCmd.AddCommand(deleteCmd)
	RootCmd.AddCommand(downloadAttachmentsCmd)
	RootCmd.AddCommand(labelsCmd)
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
	sendCmd.Flags().StringVar(&body, "body", "", "Email body (required)")
	sendCmd.Flags().StringVar(&cc, "cc", "", "CC recipients (comma-separated)")
	sendCmd.Flags().StringVar(&bcc, "bcc", "", "BCC recipients (comma-separated)")
	sendCmd.Flags().StringSliceVar(&attach, "attach", []string{}, "Attachment file paths")
	sendCmd.MarkFlagRequired("to")
	sendCmd.MarkFlagRequired("subject")
	sendCmd.MarkFlagRequired("body")
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
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
		return fmt.Errorf("Inconsistent Authentication: authenticated as %s but expected %s", profile.EmailAddress, account)
	}

	fmt.Fprintf(os.Stderr, "Authenticated as: %s\n", profile.EmailAddress)
	return nil
}

func runApplyLabel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
	if err != nil {
		return err
	}

	_, err = service.Users.Messages.Trash("me", args[0]).Do()
	if err != nil {
		return fmt.Errorf("error deleting: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Message deleted\n")
	return nil
}

func runDownloadAttachments(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
	dir, err := gmail.ExpandTilde(downloadDir)
	if err != nil {
		return err
	}

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating download directory: %w", err)
	}

	// Process attachments
	attachmentCount := 0
	if err := gmail.ProcessAttachments(service, messageID, msg.Payload, dir, &attachmentCount); err != nil {
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
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
	body := gmail.GetBody(msg.Payload)
	fmt.Println(body)

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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

	return gmail.ListMessagesWithDetails(service, response.Messages)
}

func runListLabels(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
	if err != nil {
		return err
	}

	response, err := service.Users.Messages.List("me").Q(args[0]).MaxResults(maxResults).Do()
	if err != nil {
		return fmt.Errorf("error searching: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d messages\n\n", len(response.Messages))

	return gmail.ListMessagesWithDetails(service, response.Messages)
}

func runSend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
	if err != nil {
		return err
	}

	var raw string
	if len(attach) > 0 {
		raw, err = gmail.BuildMessageWithAttachments(to, cc, bcc, subject, body, attach)
		if err != nil {
			return fmt.Errorf("error building message: %w", err)
		}
	} else {
		raw = gmail.BuildPlainMessage(to, cc, bcc, subject, body)
	}

	msg := &gmailapi.Message{Raw: raw}

	_, err = service.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Email sent successfully to %s\n", to)
	return nil
}

func runUnread(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := gmail.GetService(ctx, account)
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

// Suppress unused variable warnings for color functions
var _ = cyan
var _ = green
var _ = red
