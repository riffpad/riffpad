package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/riffpad/riffpad/apps/api/internal/sandbox"
	"github.com/sashabaranov/go-openai"
)

// SearchResult is a single web search result.
type SearchResult struct {
	Index   int    `json:"index"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchProvider searches the web for a query.
type WebSearchProvider interface {
	Search(ctx context.Context, query string) ([]SearchResult, error)
}

// DuckDuckGoProvider uses DuckDuckGo's HTML search endpoint.
// It requires no API key and is suitable as a fallback adapter.
type DuckDuckGoProvider struct {
	client *http.Client
}

// NewDuckDuckGoProvider creates a DuckDuckGo search provider.
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Search performs a DuckDuckGo HTML search and parses the results.
func (p *DuckDuckGoProvider) Search(ctx context.Context, query string) ([]SearchResult, error) {
	u := "https://html.duckduckgo.com/html/"
	data := url.Values{}
	data.Set("q", query)
	data.Set("kl", "us-en")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var results []SearchResult
	doc.Find(".result").Each(func(i int, s *goquery.Selection) {
		if i >= 5 {
			return
		}
		linkElem := s.Find(".result__a").First()
		title := strings.TrimSpace(linkElem.Text())
		href, _ := linkElem.Attr("href")
		snippet := strings.TrimSpace(s.Find(".result__snippet").First().Text())
		if title == "" || href == "" {
			return
		}
		results = append(results, SearchResult{
			Index:   i + 1,
			Title:   title,
			URL:     href,
			Snippet: snippet,
		})
	})

	if len(results) == 0 {
		return nil, fmt.Errorf("no search results found")
	}
	return results, nil
}

// WebSearchTool lets the agent search the web.
type WebSearchTool struct {
	Provider WebSearchProvider
}

// NewWebSearchTool creates a web search tool with the default provider.
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{Provider: NewDuckDuckGoProvider()}
}

func (t *WebSearchTool) Definition(lang string) openai.Tool {
	desc := "Search the web for current information, facts, or documentation. Returns up to 5 results with title, URL, and snippet. When answering, cite sources with [1], [2], etc. matching the result index."
	paramDesc := "Search query"
	if lang == "zh" {
		desc = "在网络上搜索最新信息、事实或文档。返回最多 5 条结果，每条结果带有 [1]、[2] 等角标。回答时请用对应的角标标注信息来源。"
		paramDesc = "搜索关键词"
	}
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "web_search",
			Description: desc,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]string{
						"type":        "string",
						"description": paramDesc,
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, _ sandbox.Sandbox, _ string, args json.RawMessage) (string, bool, error) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", true, fmt.Errorf("parse args: %w", err)
	}
	if t.Provider == nil {
		return "", true, fmt.Errorf("web search provider not configured")
	}
	results, err := t.Provider.Search(ctx, params.Query)
	if err != nil {
		return "", true, err
	}
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", true, err
	}
	return string(data) + "\n\n---\nCITE EVERY FACTUAL CLAIM using the exact bracketed format [1], [2], etc. matching the result index.\n- Correct: \"Beijing is sunny today [1].\" \"AccuWeather and EaseWeather both report clear skies [1][2].\"\n- Wrong: \"Beijing is sunny today 1.\" \"Beijing is sunny today 12.\" \"Beijing is sunny today [12].\"\nAlways keep each citation as a separate bracketed marker.", false, nil
}
