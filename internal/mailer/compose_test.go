package mailer

import (
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// parseRaw decodes a base64url message produced by BuildMIMEMessage and returns
// the parsed *mail.Message.
func parseRaw(t *testing.T, raw string) *mail.Message {
	t.Helper()
	data, err := base64.URLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("base64url decode: %v", err)
	}
	m, err := mail.ReadMessage(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("parse message: %v", err)
	}
	return m
}

// decodePart reads a MIME part, base64-decoding it when so encoded.
func decodePart(t *testing.T, p *multipart.Part) string {
	t.Helper()
	body, err := io.ReadAll(p)
	if err != nil {
		t.Fatalf("read part: %v", err)
	}
	if strings.EqualFold(p.Header.Get("Content-Transfer-Encoding"), "base64") {
		dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(string(body), "\r\n", ""))
		if err != nil {
			t.Fatalf("base64 decode part: %v", err)
		}
		return string(dec)
	}
	return string(body)
}

// mediaType returns the lowercased media type of a Content-Type header value.
func mediaType(t *testing.T, ct string) (string, map[string]string) {
	t.Helper()
	mt, params, err := mime.ParseMediaType(ct)
	if err != nil {
		t.Fatalf("parse media type %q: %v", ct, err)
	}
	return mt, params
}

// collectLeafParts walks a message body and returns leaf parts keyed by media
// type, with their decoded content (attachments keyed by filename).
type leaf struct {
	mediaType string
	filename  string
	content   string
}

func walkParts(t *testing.T, ct, cte string, body io.Reader) []leaf {
	t.Helper()
	mt, params := mediaType(t, ct)
	if !strings.HasPrefix(mt, "multipart/") {
		data, _ := io.ReadAll(body)
		content := string(data)
		if strings.EqualFold(cte, "base64") {
			dec, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(content, "\r\n", ""))
			if err != nil {
				t.Fatalf("base64 decode top-level: %v", err)
			}
			content = string(dec)
		}
		return []leaf{{mediaType: mt, content: content}}
	}
	var out []leaf
	mr := multipart.NewReader(body, params["boundary"])
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next part: %v", err)
		}
		pct := p.Header.Get("Content-Type")
		pmt, _ := mediaType(t, pct)
		if strings.HasPrefix(pmt, "multipart/") {
			// A nested multipart is not itself content-transfer-encoded.
			raw, _ := io.ReadAll(p)
			out = append(out, walkParts(t, pct, "", strings.NewReader(string(raw)))...)
			continue
		}
		out = append(out, leaf{mediaType: pmt, filename: p.FileName(), content: decodePart(t, p)})
	}
	return out
}

func findLeaf(leaves []leaf, mediaType string) (leaf, bool) {
	for _, l := range leaves {
		if l.mediaType == mediaType && l.filename == "" {
			return l, true
		}
	}
	return leaf{}, false
}

func TestBuildMIMEMessage_PlainOnly(t *testing.T) {
	raw, err := BuildMIMEMessage(MessageInput{To: "a@b.com", Subject: "Hi", Plain: "hello world"})
	if err != nil {
		t.Fatal(err)
	}
	m := parseRaw(t, raw)
	mt, _ := mediaType(t, m.Header.Get("Content-Type"))
	if mt != "text/plain" {
		t.Fatalf("want text/plain, got %s", mt)
	}
	leaves := walkParts(t, m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body)
	if p, ok := findLeaf(leaves, "text/plain"); !ok || p.content != "hello world" {
		t.Fatalf("plain content = %q (ok=%v)", p.content, ok)
	}
}

func TestBuildMIMEMessage_HTMLOnlyDerivesPlain(t *testing.T) {
	html := "<html><body><h2>Titre</h2><p>Bonjour <b>gras</b></p></body></html>"
	raw, err := BuildMIMEMessage(MessageInput{To: "a@b.com", Subject: "Hi", HTML: html})
	if err != nil {
		t.Fatal(err)
	}
	m := parseRaw(t, raw)
	mt, _ := mediaType(t, m.Header.Get("Content-Type"))
	if mt != "multipart/alternative" {
		t.Fatalf("want multipart/alternative, got %s", mt)
	}
	leaves := walkParts(t, m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body)
	htmlLeaf, ok := findLeaf(leaves, "text/html")
	if !ok || htmlLeaf.content != html {
		t.Fatalf("html part mismatch: %q", htmlLeaf.content)
	}
	plainLeaf, ok := findLeaf(leaves, "text/plain")
	if !ok || plainLeaf.content == "" {
		t.Fatalf("expected derived text/plain, got %q", plainLeaf.content)
	}
	// The derived plain text must carry the visible words, not HTML tags.
	if !strings.Contains(plainLeaf.content, "Bonjour") || strings.Contains(plainLeaf.content, "<b>") {
		t.Fatalf("derived plain looks wrong: %q", plainLeaf.content)
	}
}

func TestBuildMIMEMessage_HTMLWithExplicitPlain(t *testing.T) {
	raw, err := BuildMIMEMessage(MessageInput{
		To: "a@b.com", Subject: "Hi", Plain: "plain fallback", HTML: "<p>rich</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parseRaw(t, raw)
	leaves := walkParts(t, m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body)
	if p, ok := findLeaf(leaves, "text/plain"); !ok || p.content != "plain fallback" {
		t.Fatalf("explicit plain not preserved: %q", p.content)
	}
}

func TestBuildMIMEMessage_PlainWithAttachment(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(fp, []byte("file-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := BuildMIMEMessage(MessageInput{
		To: "a@b.com", Subject: "Hi", Plain: "see attached", Attachments: []string{fp},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parseRaw(t, raw)
	mt, _ := mediaType(t, m.Header.Get("Content-Type"))
	if mt != "multipart/mixed" {
		t.Fatalf("want multipart/mixed, got %s", mt)
	}
	leaves := walkParts(t, m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body)
	if p, ok := findLeaf(leaves, "text/plain"); !ok || p.content != "see attached" {
		t.Fatalf("plain part missing/mismatch: %q (ok=%v)", p.content, ok)
	}
	var found bool
	for _, l := range leaves {
		if l.filename == "note.txt" && l.content == "file-bytes" {
			found = true
		}
	}
	if !found {
		t.Fatalf("attachment not found in %+v", leaves)
	}
}

func TestBuildMIMEMessage_HTMLWithAttachment(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "img.png")
	// minimal PNG signature bytes
	if err := os.WriteFile(fp, []byte("\x89PNG\r\n\x1a\nrest"), 0o600); err != nil {
		t.Fatal(err)
	}
	raw, err := BuildMIMEMessage(MessageInput{
		To: "a@b.com", Subject: "Hi", HTML: "<p>rich</p>", Attachments: []string{fp},
	})
	if err != nil {
		t.Fatal(err)
	}
	m := parseRaw(t, raw)
	mt, _ := mediaType(t, m.Header.Get("Content-Type"))
	if mt != "multipart/mixed" {
		t.Fatalf("want multipart/mixed, got %s", mt)
	}
	leaves := walkParts(t, m.Header.Get("Content-Type"), m.Header.Get("Content-Transfer-Encoding"), m.Body)
	if _, ok := findLeaf(leaves, "text/html"); !ok {
		t.Fatalf("html part missing in %+v", leaves)
	}
	if _, ok := findLeaf(leaves, "text/plain"); !ok {
		t.Fatalf("derived plain part missing in %+v", leaves)
	}
	var attOK bool
	for _, l := range leaves {
		if l.filename == "img.png" {
			attOK = true
		}
	}
	if !attOK {
		t.Fatalf("png attachment missing in %+v", leaves)
	}
}
