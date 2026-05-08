// Package gmail provides Gmail API service functionality.
package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"email-manager/pkg/auth"

	"golang.org/x/net/html"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// GetService returns a Gmail service instance for the given account.
func GetService(ctx context.Context, account string) (*gmail.Service, error) {
	client, err := auth.GetClient(ctx, account)
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

// collectBodies walks the MIME tree recursively and returns the first text/plain
// and text/html parts it finds (decoded).
func collectBodies(part *gmail.MessagePart) (plain, htmlBody string) {
	if part == nil {
		return
	}

	mime := strings.ToLower(strings.SplitN(part.MimeType, ";", 2)[0])

	if part.Body != nil && part.Body.Data != "" && part.Filename == "" {
		if data, err := base64.URLEncoding.DecodeString(part.Body.Data); err == nil {
			switch mime {
			case "text/plain":
				if plain == "" {
					plain = string(data)
				}
			case "text/html":
				if htmlBody == "" {
					htmlBody = string(data)
				}
			case "":
				if plain == "" && len(part.Parts) == 0 {
					plain = string(data)
				}
			}
		}
	}

	for _, p := range part.Parts {
		subPlain, subHTML := collectBodies(p)
		if plain == "" {
			plain = subPlain
		}
		if htmlBody == "" {
			htmlBody = subHTML
		}
	}
	return
}

// GetBody extracts the body text from a message. Prefers text/plain, falls
// back to text/html converted to readable text (with links preserved).
func GetBody(part *gmail.MessagePart) string {
	plain, htmlBody := collectBodies(part)
	if plain != "" {
		return plain
	}
	if htmlBody != "" {
		return HTMLToText(htmlBody)
	}
	return "[No text content]"
}

// GetHTMLBody returns the raw HTML body if present, otherwise the plain body.
func GetHTMLBody(part *gmail.MessagePart) string {
	plain, htmlBody := collectBodies(part)
	if htmlBody != "" {
		return htmlBody
	}
	if plain != "" {
		return plain
	}
	return "[No content]"
}

// HTMLToText converts HTML to readable plain text, preserving links as
// "text (URL)" and rendering block-level elements with line breaks.
func HTMLToText(input string) string {
	doc, err := html.Parse(strings.NewReader(input))
	if err != nil {
		return input
	}

	var b strings.Builder
	var walk func(n *html.Node)
	skip := map[string]bool{"script": true, "style": true, "head": true, "noscript": true}
	block := map[string]bool{
		"p": true, "div": true, "br": true, "tr": true, "li": true,
		"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
		"section": true, "article": true, "header": true, "footer": true,
		"blockquote": true, "pre": true, "ul": true, "ol": true, "table": true,
	}

	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && skip[n.Data] {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			var href string
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
					break
				}
			}
			var inner strings.Builder
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				collectText(c, &inner)
			}
			text := strings.TrimSpace(inner.String())
			if href != "" && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "mailto:") {
				if text == "" || text == href {
					b.WriteString(href)
				} else {
					b.WriteString(text)
					b.WriteString(" (")
					b.WriteString(href)
					b.WriteString(")")
				}
			} else {
				b.WriteString(text)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			if n.Data == "br" || block[n.Data] {
				b.WriteString("\n")
			}
		}
	}
	walk(doc)

	out := b.String()
	lines := strings.Split(out, "\n")
	cleaned := make([]string, 0, len(lines))
	blank := 0
	for _, l := range lines {
		t := strings.TrimRight(l, " \t\r")
		if strings.TrimSpace(t) == "" {
			blank++
			if blank > 1 {
				continue
			}
		} else {
			blank = 0
		}
		cleaned = append(cleaned, t)
	}
	return strings.TrimSpace(strings.Join(cleaned, "\n"))
}

func collectText(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
		return
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, b)
	}
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
