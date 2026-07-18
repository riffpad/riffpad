package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
)

func TestBashExecTool_BackgroundServer(t *testing.T) {
	sb, err := sandbox.NewLocal("test-bg-" + fmt.Sprint(time.Now().UnixNano()))
	if err != nil {
		t.Fatalf("create sandbox: %v", err)
	}
	defer sb.Destroy()

	// Write a simple index.html so the server has something to serve.
	if err := sb.WriteFile(context.Background(), "index.html", "<h1>ok</h1>"); err != nil {
		t.Fatalf("write file: %v", err)
	}

	tool := &BashExecTool{}
	args, _ := json.Marshal(map[string]any{
		"command":    "python3 -m http.server 18082",
		"background": true,
	})

	output, isError, err := tool.Execute(context.Background(), sb, "call-1", args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if isError {
		t.Fatalf("expected success, got error: %s", output)
	}
	if !strings.Contains(output, "Background process started") {
		t.Fatalf("unexpected output: %s", output)
	}

	// Give the server a moment to start, then verify it responds.
	time.Sleep(2 * time.Second)
	resp, err := http.Get("http://localhost:18082/index.html")
	if err != nil {
		t.Fatalf("server did not start: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected status: %d", resp.StatusCode)
	}

	// Clean up the background process.
	_, _ = sb.Exec(context.Background(), "pkill -f 'http.server 18082' || true", sandbox.ExecOptions{Timeout: 5 * time.Second})
}

func TestWebSearchTool_Execute(t *testing.T) {
	// Stub provider to avoid external network calls in tests.
	stub := &stubSearchProvider{
		results: []SearchResult{
			{Title: "Go Hello World", URL: "https://go.dev/hello", Snippet: "A simple Go program."},
		},
	}

	tool := &WebSearchTool{Provider: stub}
	args, _ := json.Marshal(map[string]any{"query": "golang hello world"})

	output, isError, err := tool.Execute(context.Background(), nil, "call-1", args)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if isError {
		t.Fatalf("expected success, got error: %s", output)
	}
	if !strings.Contains(output, "Go Hello World") {
		t.Fatalf("expected search result in output, got: %s", output)
	}
}

func TestDuckDuckGoProvider_Search(t *testing.T) {
	// Serve a minimal DuckDuckGo HTML result page.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`
			<div class="result">
				<a class="result__a" href="https://example.com/one">First Result</a>
				<a class="result__snippet">This is the first snippet.</a>
			</div>
			<div class="result">
				<a class="result__a" href="https://example.com/two">Second Result</a>
				<a class="result__snippet">This is the second snippet.</a>
			</div>
		`))
	}))
	defer ts.Close()

	// Point provider at the test server by replacing the HTTP client base URL behavior.
	// We do this by overriding the provider's client with a custom RoundTripper.
	provider := NewDuckDuckGoProvider()
	provider.client = &http.Client{
		Transport: &fixedHostTransport{base: ts.URL},
		Timeout:   5 * time.Second,
	}

	results, err := provider.Search(context.Background(), "test query")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Title != "First Result" {
		t.Fatalf("unexpected first result title: %s", results[0].Title)
	}
	if results[1].URL != "https://example.com/two" {
		t.Fatalf("unexpected second result URL: %s", results[1].URL)
	}
}

type stubSearchProvider struct {
	results []SearchResult
	err     error
}

func (s *stubSearchProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

// fixedHostTransport redirects all requests to the test server URL while preserving paths.
type fixedHostTransport struct {
	base string
}

func (f *fixedHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the request URL host with the test server host.
	baseURL := f.base + req.URL.Path
	if req.URL.RawQuery != "" {
		baseURL += "?" + req.URL.RawQuery
	}
	newReq, err := http.NewRequestWithContext(req.Context(), req.Method, baseURL, req.Body)
	if err != nil {
		return nil, err
	}
	newReq.Header = req.Header
	return http.DefaultTransport.RoundTrip(newReq)
}

// Ensure test workspace temp roots do not leak on failure.
func TestMain(m *testing.M) {
	code := m.Run()
	_ = os.RemoveAll("/tmp/riffpad-sandbox/test-*")
	os.Exit(code)
}
