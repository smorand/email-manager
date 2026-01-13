// Package gmail provides Gmail API service functionality.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"email-manager/pkg/auth"

	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GetService returns a Gmail service instance.
func GetService(ctx context.Context) (*gmail.Service, error) {
	client, err := auth.GetClient(ctx)
	if err != nil {
		return nil, err
	}

	service, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to create Gmail service: %w", err)
	}

	return service, nil
}

// ExtractHeaders extracts subject and from headers from a message.
func ExtractHeaders(headers []*gmail.MessagePartHeader) (subject, from string) {
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

// GetBody extracts the body text from a message part.
func GetBody(part *gmail.MessagePart) string {
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

// ListMessagesWithDetails prints detailed information about messages.
func ListMessagesWithDetails(service *gmail.Service, messages []*gmail.Message) error {
	for _, msg := range messages {
		fullMsg, err := service.Users.Messages.Get("me", msg.Id).Do()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get message %s: %v\n", msg.Id, err)
			continue
		}

		subject, from := ExtractHeaders(fullMsg.Payload.Headers)
		fmt.Printf("ID: %s\n", msg.Id)
		fmt.Printf("From: %s\n", from)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Println("---")
	}
	return nil
}

// ProcessAttachments recursively processes and downloads attachments.
func ProcessAttachments(service *gmail.Service, messageID string, part *gmail.MessagePart, dir string, count *int) error {
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
		if err := ProcessAttachments(service, messageID, subPart, dir, count); err != nil {
			return err
		}
	}

	return nil
}

// ExpandTilde expands ~ to user's home directory.
func ExpandTilde(path string) (string, error) {
	dir := os.ExpandEnv(path)
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error getting home directory: %w", err)
		}
		dir = strings.Replace(dir, "~", home, 1)
	}
	return dir, nil
}
