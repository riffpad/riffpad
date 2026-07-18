package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/riffpad/riffpad/apps/api/internal/config"
)

// NewLLMHTTPClient returns an http.Client that optionally injects provider-specific
// extension fields (e.g. SiliconFlow thinking controls) into chat-completion requests.
func NewLLMHTTPClient(cfg config.LLMConfig) *http.Client {
	transport := http.DefaultTransport
	if cfg.EnableThinking != nil || cfg.ThinkingBudget != nil {
		transport = &thinkingInjector{
			base: transport,
			cfg:  cfg,
		}
	}
	return &http.Client{Transport: transport}
}

type thinkingInjector struct {
	base http.RoundTripper
	cfg  config.LLMConfig
}

func (t *thinkingInjector) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method != http.MethodPost || !strings.HasSuffix(req.URL.Path, "/chat/completions") {
		return t.base.RoundTrip(req)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		// Not JSON we can extend; forward unchanged.
		req.Body = io.NopCloser(bytes.NewReader(body))
		return t.base.RoundTrip(req)
	}

	if t.cfg.EnableThinking != nil {
		payload["enable_thinking"] = *t.cfg.EnableThinking
	}
	if t.cfg.ThinkingBudget != nil {
		payload["thinking_budget"] = *t.cfg.ThinkingBudget
	}

	newBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req.Body = io.NopCloser(bytes.NewReader(newBody))
	req.ContentLength = int64(len(newBody))
	req.Header.Set("Content-Length", strconv.Itoa(len(newBody)))
	req.Header.Set("Content-Type", "application/json")

	return t.base.RoundTrip(req)
}
