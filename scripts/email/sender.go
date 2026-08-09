package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strings"
)

type smtpSender struct {
	host     string
	port     int
	user     string
	pass     string
	from     string
	fromName string
	addr     string
}

func newSMTPSender(host string, port int, user, pass, from, fromName string) *smtpSender {
	return &smtpSender{
		host:     host,
		port:     port,
		user:     user,
		pass:     pass,
		from:     from,
		fromName: fromName,
		addr:     net.JoinHostPort(host, fmt.Sprintf("%d", port)),
	}
}

func (s *smtpSender) send(to, subject, plainText, htmlText string) error {
	msg, err := s.buildMessage(to, subject, plainText, htmlText)
	if err != nil {
		return err
	}
	conn, err := tls.Dial("tcp", s.addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	defer conn.Close()

	c, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer c.Close()

	if err := c.Auth(smtp.PlainAuth("", s.user, s.pass, s.host)); err != nil {
		return fmt.Errorf("auth: %w", err)
	}
	if err := c.Mail(s.from); err != nil {
		return fmt.Errorf("mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("rcpt %s: %w", to, err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("close data: %w", err)
	}
	return c.Quit()
}

// buildMessage renders a MIME message: text/plain alone, or
// multipart/alternative with a text/plain fallback plus the HTML part.
// Both parts are base64-encoded so non-ASCII (Chinese) survives any relay.
func (s *smtpSender) buildMessage(to, subject, plainText, htmlText string) ([]byte, error) {
	subject = strings.ReplaceAll(strings.ReplaceAll(subject, "\r", " "), "\n", " ")
	var b bytes.Buffer
	fromHeader := s.from
	if s.fromName != "" {
		fromHeader = fmt.Sprintf("%s <%s>", mime.QEncoding.Encode("utf-8", s.fromName), s.from)
	}
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\n", fromHeader, to, mime.QEncoding.Encode("utf-8", subject))
	if htmlText == "" {
		b.WriteString("MIME-Version: 1.0\r\n")
		b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
		b.WriteString(base64Body(plainText))
		b.WriteString("\r\n")
		return b.Bytes(), nil
	}
	boundary, err := newBoundary()
	if err != nil {
		return nil, err
	}
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(base64Body(plainText))
	b.WriteString("\r\n\r\n")
	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	b.WriteString("Content-Transfer-Encoding: base64\r\n\r\n")
	b.WriteString(base64Body(htmlText))
	b.WriteString("\r\n\r\n")
	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes(), nil
}

func newBoundary() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return "riffpad-" + hex.EncodeToString(raw), nil
}

func base64Body(s string) string {
	raw := base64.StdEncoding.EncodeToString([]byte(s))
	var b strings.Builder
	for len(raw) > 76 {
		b.WriteString(raw[:76])
		b.WriteString("\r\n")
		raw = raw[76:]
	}
	b.WriteString(raw)
	return b.String()
}
