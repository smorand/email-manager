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

// BuildPlainMessage builds a simple text/plain email and returns it as a
// base64url encoded string. The from parameter, when non-empty, is written as
// the From header with RFC 2047 encoding of any non-ASCII display name; if
// empty, Gmail's API will auto-inject From (which currently emits raw UTF-8
// and causes mojibake on the recipient side).
func BuildPlainMessage(from, to, cc, bcc, subject, body string) string {
	var msg strings.Builder
	if from != "" {
		fmt.Fprintf(&msg, "From: %s\r\n", encodeAddress(from))
	}
	fmt.Fprintf(&msg, "To: %s\r\n", encodeAddressList(to))
	if cc != "" {
		fmt.Fprintf(&msg, "Cc: %s\r\n", encodeAddressList(cc))
	}
	if bcc != "" {
		fmt.Fprintf(&msg, "Bcc: %s\r\n", encodeAddressList(bcc))
	}
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(base64.StdEncoding.EncodeToString([]byte(body)))

	return base64.URLEncoding.EncodeToString([]byte(msg.String()))
}

// BuildMessageWithAttachments builds a multipart/mixed email with attachments
// and returns it as a base64url encoded string. See BuildPlainMessage for
// the semantics of the from parameter.
func BuildMessageWithAttachments(from, to, cc, bcc, subject, body string, attachments []string) (string, error) {
	boundary := "boundary_email_manager_attachment"

	var msg strings.Builder

	// Write top-level headers
	if from != "" {
		fmt.Fprintf(&msg, "From: %s\r\n", encodeAddress(from))
	}
	fmt.Fprintf(&msg, "To: %s\r\n", encodeAddressList(to))
	if cc != "" {
		fmt.Fprintf(&msg, "Cc: %s\r\n", encodeAddressList(cc))
	}
	if bcc != "" {
		fmt.Fprintf(&msg, "Bcc: %s\r\n", encodeAddressList(bcc))
	}
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	msg.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary)
	msg.WriteString("\r\n")

	// Body part
	fmt.Fprintf(&msg, "--%s\r\n", boundary)
	msg.WriteString("Content-Type: text/plain; charset=\"utf-8\"\r\n")
	msg.WriteString("Content-Transfer-Encoding: base64\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(encodeBase64Wrapped([]byte(body)))
	msg.WriteString("\r\n")

	// Attachment parts
	for _, filePath := range attachments {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("unable to read attachment %s: %w", filePath, err)
		}

		mimeType := detectMIMEType(filePath, data)
		filename := filepath.Base(filePath)

		fmt.Fprintf(&msg, "--%s\r\n", boundary)
		fmt.Fprintf(&msg, "Content-Type: %s; name=\"%s\"\r\n", mimeType, filename)
		fmt.Fprintf(&msg, "Content-Disposition: attachment; filename=\"%s\"\r\n", filename)
		msg.WriteString("Content-Transfer-Encoding: base64\r\n")
		msg.WriteString("\r\n")
		msg.WriteString(encodeBase64Wrapped(data))
		msg.WriteString("\r\n")
	}

	// Closing boundary
	fmt.Fprintf(&msg, "--%s--\r\n", boundary)

	return base64.URLEncoding.EncodeToString([]byte(msg.String())), nil
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
