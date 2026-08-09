package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"text/template"
)

func loadTemplate(path string) (*template.Template, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	t, err := template.New("body").Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	return t, nil
}

// renderBody renders the template and guarantees the unsubscribe link is
// present, appending a plain footer if the template forgot it.
func renderBody(t *template.Template, r recipient, u string) (string, error) {
	var buf bytes.Buffer
	if err := t.Execute(&buf, map[string]string{
		"Email":          r.Email,
		"Name":           r.Name,
		"UnsubscribeURL": u,
	}); err != nil {
		return "", err
	}
	body := strings.TrimRight(buf.String(), "\n")
	if !strings.Contains(body, u) {
		body += "\n\n---\n" + u
	}
	return body + "\n", nil
}
