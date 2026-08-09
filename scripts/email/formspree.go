package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

type formspreeResponse struct {
	Submissions []map[string]any `json:"submissions"`
}

func fetchFormspree(formID, apiKey string) ([]recipient, error) {
	u := fmt.Sprintf("https://formspree.io/api/0/forms/%s/submissions", formID)
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth("", apiKey)
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("formspree status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var out formspreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode formspree response: %w", err)
	}

	seen := map[string]bool{}
	var recs []recipient
	for _, sub := range out.Submissions {
		email := submissionEmail(sub)
		email = normalizeEmail(email)
		if email == "" || !validEmail(email) || seen[email] {
			continue
		}
		seen[email] = true
		recs = append(recs, recipient{Email: email, Date: submissionDate(sub)})
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Email < recs[j].Email })
	return recs, nil
}

func submissionEmail(sub map[string]any) string {
	if v, ok := sub["email"].(string); ok {
		return v
	}
	if data, ok := sub["data"].(map[string]any); ok {
		if v, ok := data["email"].(string); ok {
			return v
		}
	}
	return ""
}

func submissionDate(sub map[string]any) string {
	for _, k := range []string{"_date", "created_at", "date"} {
		if v, ok := sub[k].(string); ok {
			return v
		}
	}
	return ""
}
