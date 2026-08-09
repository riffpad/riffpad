package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func fetchOptOuts(adminKey string) ([]string, error) {
	u := strings.TrimSuffix(envOr("RIFFPAD_API_URL", "https://api.riffpad.ai"), "/") + "/api/waitlist/optouts"
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
		return nil, fmt.Errorf("optouts status %d", resp.StatusCode)
	}
	var out struct {
		Emails []string `json:"emails"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Emails, nil
}
