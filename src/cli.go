package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"google.golang.org/api/gmail/v1"
)

// Color functions
var (
	cyan  = color.New(color.FgCyan).SprintFunc()
	green = color.New(color.FgGreen).SprintFunc()
	red   = color.New(color.FgRed).SprintFunc()
)

// Command line flags
var (
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

// Command definitions

var (
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

func runApplyLabel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	req := &gmail.ModifyMessageRequest{
		AddLabelIds: []string{args[1]},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error applying label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Label applied\n")
	return nil
}

func runArchive(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"INBOX"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error archiving: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Message archived\n")
	return nil
}

func runCreateLabel(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	label := &gmail.Label{
		Name: args[0],
	}

	result, err := service.Users.Labels.Create("me", label).Do()
	if err != nil {
		return fmt.Errorf("error creating label: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Label created: %s (ID: %s)\n", result.Name, result.Id)
	return nil
}

func runDelete(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	_, err = service.Users.Messages.Trash("me", args[0]).Do()
	if err != nil {
		return fmt.Errorf("error deleting: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Message deleted\n")
	return nil
}

func runDownloadAttachments(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
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
	dir := os.ExpandEnv(downloadDir)
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("error getting home directory: %w", err)
		}
		dir = strings.Replace(dir, "~", home, 1)
	}

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating download directory: %w", err)
	}

	// Process attachments
	attachmentCount := 0
	if err := processAttachments(service, messageID, msg.Payload, dir, &attachmentCount); err != nil {
		return err
	}

	if attachmentCount == 0 {
		fmt.Fprintf(os.Stderr, "No attachments found\n")
		return nil
	}

	fmt.Fprintf(os.Stderr, "✅ Downloaded %d attachment(s) to %s\n", attachmentCount, dir)
	return nil
}

func runGet(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
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
	body := getBody(msg.Payload)
	fmt.Println(body)

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
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

	return listMessagesWithDetails(service, response.Messages)
}

func runListLabels(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
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
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	req := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error marking as read: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Message marked as read\n")
	return nil
}

func runSearch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	response, err := service.Users.Messages.List("me").Q(args[0]).MaxResults(maxResults).Do()
	if err != nil {
		return fmt.Errorf("error searching: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d messages\n\n", len(response.Messages))

	return listMessagesWithDetails(service, response.Messages)
}

func runSend(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	var message strings.Builder
	message.WriteString(fmt.Sprintf("To: %s\r\n", to))
	if cc != "" {
		message.WriteString(fmt.Sprintf("Cc: %s\r\n", cc))
	}
	if bcc != "" {
		message.WriteString(fmt.Sprintf("Bcc: %s\r\n", bcc))
	}
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("\r\n")
	message.WriteString(body)

	raw := base64.URLEncoding.EncodeToString([]byte(message.String()))

	msg := &gmail.Message{
		Raw: raw,
	}

	_, err = service.Users.Messages.Send("me", msg).Do()
	if err != nil {
		return fmt.Errorf("error sending email: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Email sent successfully to %s\n", to)
	return nil
}

func runUnread(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	service, err := getGmailService(ctx)
	if err != nil {
		return err
	}

	req := &gmail.ModifyMessageRequest{
		AddLabelIds: []string{"UNREAD"},
	}

	_, err = service.Users.Messages.Modify("me", args[0], req).Do()
	if err != nil {
		return fmt.Errorf("error marking as unread: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Message marked as unread\n")
	return nil
}

// Helper functions (alphabetically ordered)

func extractHeaders(headers []*gmail.MessagePartHeader) (subject, from string) {
	for _, header := range headers {
		switch header.Name {
		case "Subject":
			subject = header.Value
		case "From":
			from = header.Value
		}
	}
	return
}

func getBody(part *gmail.MessagePart) string {
	if part.Body != nil && part.Body.Data != "" {
		data, err := base64.URLEncoding.DecodeString(part.Body.Data)
		if err == nil {
			return string(data)
		}
	}

	for _, p := range part.Parts {
		if p.MimeType == "text/plain" {
			if p.Body != nil && p.Body.Data != "" {
				data, err := base64.URLEncoding.DecodeString(p.Body.Data)
				if err == nil {
					return string(data)
				}
			}
		}
	}

	return "[No text content]"
}

func listMessagesWithDetails(service *gmail.Service, messages []*gmail.Message) error {
	for _, msg := range messages {
		fullMsg, err := service.Users.Messages.Get("me", msg.Id).Do()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get message %s: %v\n", msg.Id, err)
			continue
		}

		subject, from := extractHeaders(fullMsg.Payload.Headers)
		fmt.Printf("ID: %s\n", msg.Id)
		fmt.Printf("From: %s\n", from)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Println("---")
	}
	return nil
}

func processAttachments(service *gmail.Service, messageID string, part *gmail.MessagePart, dir string, count *int) error {
	// Check if this part has a filename (is an attachment)
	if part.Filename != "" && part.Body != nil {
		attachmentID := part.Body.AttachmentId

		if attachmentID != "" {
			// Download the attachment
			fmt.Fprintf(os.Stderr, "Downloading: %s\n", part.Filename)

			attachment, err := service.Users.Messages.Attachments.Get("me", messageID, attachmentID).Do()
			if err != nil {
				return fmt.Errorf("error downloading attachment %s: %w", part.Filename, err)
			}

			// Decode the attachment data
			data, err := base64.URLEncoding.DecodeString(attachment.Data)
			if err != nil {
				return fmt.Errorf("error decoding attachment %s: %w", part.Filename, err)
			}

			// Write to file
			filepath := fmt.Sprintf("%s/%s", dir, part.Filename)
			if err := os.WriteFile(filepath, data, 0644); err != nil {
				return fmt.Errorf("error writing file %s: %w", filepath, err)
			}

			fmt.Fprintf(os.Stderr, "Saved: %s\n", filepath)
			*count++
		}
	}

	// Recursively process parts
	for _, subPart := range part.Parts {
		if err := processAttachments(service, messageID, subPart, dir, count); err != nil {
			return err
		}
	}

	return nil
}
