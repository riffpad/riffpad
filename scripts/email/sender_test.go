package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMessagePlainTextOnly(t *testing.T) {
	s := newSMTPSender("mail.test", 465, "u@test", "p", "from@test", "Riffpad")
	msg, err := s.buildMessage("to@test", "hello world", "plain body", "")
	if err != nil {
		t.Fatal(err)
	}
	text := string(msg)
	if !strings.Contains(text, "Content-Type: text/plain; charset=UTF-8") {
		t.Fatalf("missing plain content type:\n%s", text)
	}
	if strings.Contains(text, "multipart/alternative") {
		t.Fatal("unexpected multipart for plain-only message")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.Split(text, "\r\n\r\n")[1]))
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "plain body" {
		t.Fatalf("unexpected body: %q", decoded)
	}
}

func TestBuildMessageMultipartAlternative(t *testing.T) {
	s := newSMTPSender("mail.test", 465, "u@test", "p", "from@test", "Riffpad")
	plain := "纯文本兜底 with unsubscribe https://x/unsub"
	html := "<html><body>中文 HTML <a href=\"https://x/unsub\">退订</a></body></html>"
	msg, err := s.buildMessage("to@test", "Riffpad is in beta — 测试", plain, html)
	if err != nil {
		t.Fatal(err)
	}
	text := string(msg)
	if !strings.Contains(text, "multipart/alternative; boundary=\"riffpad-") {
		t.Fatalf("missing multipart header:\n%s", text)
	}
	if !strings.Contains(text, "Content-Type: text/plain; charset=UTF-8") ||
		!strings.Contains(text, "Content-Type: text/html; charset=UTF-8") {
		t.Fatalf("missing parts:\n%s", text)
	}
	// Decode and verify both parts survived round-trip.
	parts := map[string]string{}
	boundary := strings.Split(strings.Split(text, "boundary=\"")[1], "\"")[0]
	chunks := strings.Split(text, "--"+boundary)
	for _, chunk := range chunks {
		chunk = strings.TrimPrefix(chunk, "\r\n")
		headerEnd := strings.Index(chunk, "\r\n\r\n")
		if headerEnd < 0 {
			continue
		}
		var contentType string
		for _, line := range strings.Split(chunk[:headerEnd], "\r\n") {
			if strings.HasPrefix(line, "Content-Type: ") {
				contentType = strings.TrimSpace(strings.TrimPrefix(line, "Content-Type: "))
				break
			}
		}
		if contentType != "" {
			b64 := chunk[headerEnd+4:]
			b64 = strings.TrimSuffix(strings.TrimSpace(b64), "--")
			raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
			if err != nil {
				t.Fatalf("decode %s: %v", contentType, err)
			}
			parts[contentType] = string(raw)
		}
	}
	if parts["text/plain; charset=UTF-8"] != plain {
		t.Fatalf("plain mismatch: %q", parts["text/plain; charset=UTF-8"])
	}
	if parts["text/html; charset=UTF-8"] != html {
		t.Fatalf("html mismatch: %q", parts["text/html; charset=UTF-8"])
	}
}

func TestRenderBodyHTMLAppendsUnsubscribeFooter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.html")
	if err := os.WriteFile(path, []byte("<p>Hello {{.Email}}</p>"), 0o600); err != nil {
		t.Fatal(err)
	}
	tmpl, err := loadTemplate(path)
	if err != nil {
		t.Fatal(err)
	}
	u := "https://riffpad.ai/unsubscribe?email=a%40b.c&token=x"
	body, err := renderBodyHTML(tmpl, recipient{Email: "a@b.c"}, u)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, u) || !strings.Contains(body, "<p>Hello a@b.c</p>") {
		t.Fatalf("html body incomplete: %s", body)
	}
}
