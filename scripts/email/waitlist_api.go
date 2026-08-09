package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// fetchWaitlist pulls the collected waitlist entries from the relay
// (POST /api/waitlist/subscribe writes them; this is the announcement tool's
// read side, protected by WAITLIST_ADMIN_KEY).
func fetchWaitlist(adminKey string) ([]recipient, error) {
	u := strings.TrimSuffix(envOr("RIFFPAD_API_URL", "https://api.riffpad.ai"), "/") + "/api/waitlist/emails"
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Admin-Key", adminKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("waitlist emails status %d", resp.StatusCode)
	}
	var out struct {
		Entries []struct {
			Email     string `json:"email"`
			CreatedAt string `json:"createdAt"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	recs := make([]recipient, 0, len(out.Entries))
	for _, e := range out.Entries {
		email := normalizeEmail(e.Email)
		if email == "" || !validEmail(email) {
			continue
		}
		recs = append(recs, recipient{Email: email, Date: e.CreatedAt})
	}
	return recs, nil
}
