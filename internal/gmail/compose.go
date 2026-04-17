// Package gmail provides Gmail API service functionality.
package gmail

import (
	"encoding/base64"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// BuildPlainMessage builds a simple text/plain email and returns it as a base64url encoded string.
func BuildPlainMessage(to, cc, bcc, subject, body string) string {
	var msg strings.Builder
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	if cc != "" {
		fmt.Fprintf(&msg, "Cc: %s\r\n", cc)
	}
	if bcc != "" {
		fmt.Fprintf(&msg, "Bcc: %s\r\n", bcc)
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
// and returns it as a base64url encoded string.
func BuildMessageWithAttachments(to, cc, bcc, subject, body string, attachments []string) (string, error) {
	boundary := "boundary_email_manager_attachment"

	var msg strings.Builder

	// Write top-level headers
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	if cc != "" {
		fmt.Fprintf(&msg, "Cc: %s\r\n", cc)
	}
	if bcc != "" {
		fmt.Fprintf(&msg, "Bcc: %s\r\n", bcc)
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
