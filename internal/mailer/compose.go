// Package mailer provides Gmail API service functionality.
// Named 'mailer' (not 'gmail') to avoid collision with the external
// google.golang.org/api/gmail/v1 package.
package mailer

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
)

// encodeAddress returns the RFC 2047 encoded form of a single address.
// Accepts either a bare email address ("foo@bar.com") or a name-address form
// ("Name <foo@bar.com>"). When the display name contains non-ASCII characters,
// (*mail.Address).String() encodes it as a RFC 2047 encoded-word automatically.
// On parse failure, the input is returned unchanged.
func encodeAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return addr
	}
	return parsed.String()
}

// encodeAddressList returns the RFC 2047 encoded form of a comma-separated
// address list. On parse failure, falls back to per-address encoding.
func encodeAddressList(list string) string {
	list = strings.TrimSpace(list)
	if list == "" {
		return ""
	}
	addrs, err := mail.ParseAddressList(list)
	if err != nil {
		parts := strings.Split(list, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			out = append(out, encodeAddress(p))
		}
		return strings.Join(out, ", ")
	}
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.String()
	}
	return strings.Join(out, ", ")
}

// MIME boundary strings. Distinct per nesting level so a multipart/mixed can
// contain a multipart/alternative without boundary collision.
const (
	mixedBoundary = "boundary_email_manager_mixed"
	altBoundary   = "boundary_email_manager_alternative"
)

// MessageInput holds every field needed to build an outgoing MIME message.
// Plain is the text/plain body; HTML is the text/html body. At least one of
// them must be non-empty. When HTML is set and Plain is empty, a text/plain
// fallback is derived from the HTML so the message stays a well-formed
// multipart/alternative (better deliverability and text-client rendering).
type MessageInput struct {
	From        string
	To          string
	Cc          string
	Bcc         string
	Subject     string
	Plain       string
	HTML        string
	Attachments []string
	InReplyTo   string
	References  string
}

// BuildMIMEMessage builds an outgoing message and returns it base64url encoded
// for the Gmail API. It selects the simplest MIME structure that fits the input:
//
//   - plain only:                text/plain
//   - html (+ optional plain):   multipart/alternative { text/plain, text/html }
//   - any of the above + files:  multipart/mixed { <body>, attachment... }
//
// The from field, when non-empty, is RFC 2047 encoded; when empty, Gmail
// auto-injects From (which can mojibake a non-ASCII display name).
func BuildMIMEMessage(in MessageInput) (string, error) {
	plain := in.Plain
	if in.HTML != "" && strings.TrimSpace(plain) == "" {
		plain = HTMLToText(in.HTML)
	}

	var msg strings.Builder
	writeHeaders(&msg, in)

	body := buildBodyPart(plain, in.HTML)

	if len(in.Attachments) == 0 {
		msg.WriteString(body)
		return base64.URLEncoding.EncodeToString([]byte(msg.String())), nil
	}

	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n", mixedBoundary)
	fmt.Fprintf(&msg, "--%s\r\n", mixedBoundary)
	msg.WriteString(body)
	msg.WriteString("\r\n")

	for _, filePath := range in.Attachments {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("unable to read attachment %s: %w", filePath, err)
		}
		mimeType := detectMIMEType(filePath, data)
		filename := filepath.Base(filePath)

		fmt.Fprintf(&msg, "--%s\r\n", mixedBoundary)
		fmt.Fprintf(&msg, "Content-Type: %s; name=\"%s\"\r\n", mimeType, filename)
		fmt.Fprintf(&msg, "Content-Disposition: attachment; filename=\"%s\"\r\n", filename)
		msg.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		msg.WriteString(encodeBase64Wrapped(data))
		msg.WriteString("\r\n")
	}

	fmt.Fprintf(&msg, "--%s--\r\n", mixedBoundary)
	return base64.URLEncoding.EncodeToString([]byte(msg.String())), nil
}

// writeHeaders writes the shared top-level headers (From/To/Cc/Bcc/Subject and
// MIME-Version) but NOT the Content-Type, which depends on the body shape.
func writeHeaders(msg *strings.Builder, in MessageInput) {
	if in.From != "" {
		fmt.Fprintf(msg, "From: %s\r\n", encodeAddress(in.From))
	}
	fmt.Fprintf(msg, "To: %s\r\n", encodeAddressList(in.To))
	if in.Cc != "" {
		fmt.Fprintf(msg, "Cc: %s\r\n", encodeAddressList(in.Cc))
	}
	if in.Bcc != "" {
		fmt.Fprintf(msg, "Bcc: %s\r\n", encodeAddressList(in.Bcc))
	}
	fmt.Fprintf(msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", in.Subject))
	if in.InReplyTo != "" {
		fmt.Fprintf(msg, "In-Reply-To: %s\r\n", in.InReplyTo)
	}
	if in.References != "" {
		fmt.Fprintf(msg, "References: %s\r\n", in.References)
	}
	msg.WriteString("MIME-Version: 1.0\r\n")
}

// buildBodyPart returns the message body as a self-contained MIME entity: a
// Content-Type header, its headers, a blank line and the encoded content. It is
// either a lone text/plain part or a multipart/alternative (plain + html) when
// html is non-empty. The returned string is embeddable both as the top-level
// body and as a sub-part of a multipart/mixed.
func buildBodyPart(plain, htmlBody string) string {
	if htmlBody == "" {
		return textPart("text/plain", plain)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n\r\n", altBoundary)
	fmt.Fprintf(&b, "--%s\r\n", altBoundary)
	b.WriteString(textPart("text/plain", plain))
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s\r\n", altBoundary)
	b.WriteString(textPart("text/html", htmlBody))
	b.WriteString("\r\n")
	fmt.Fprintf(&b, "--%s--\r\n", altBoundary)
	return b.String()
}

// textPart renders a single text/* MIME part (headers + blank line + base64
// content) for the given content type.
func textPart(contentType, content string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Content-Type: %s; charset=\"utf-8\"\r\n", contentType)
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(encodeBase64Wrapped([]byte(content)))
	return b.String()
}

// detectMIMEType detects the MIME type of a file using extension first, then content sniffing.
func detectMIMEType(filePath string, data []byte) string {
	ext := filepath.Ext(filePath)
	if ext != "" {
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			return mimeType
		}
	}

	mimeType := http.DetectContentType(data)
	if mimeType != "" {
		return mimeType
	}

	return "application/octet-stream"
}

// encodeBase64Wrapped encodes data as base64 with line wrapping at 76 characters per RFC 2045.
func encodeBase64Wrapped(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var wrapped strings.Builder
	for i := 0; i < len(encoded); i += 76 {
		end := min(i+76, len(encoded))
		wrapped.WriteString(encoded[i:end])
		if end < len(encoded) {
			wrapped.WriteString("\r\n")
		}
	}
	return wrapped.String()
}
