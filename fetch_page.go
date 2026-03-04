package chat

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	readability "github.com/go-shiori/go-readability"
	ollamaapi "github.com/ollama/ollama/api"
)

const (
	fetchTimeout      = 10 * time.Second
	fetchMaxBody      = 2 << 20 // 2MB
	extractionMaxLen  = 8000    // chars sent to extraction model
	extractionModel   = "qwen2.5:3b"
	fetchUserAgent    = "axon-chat/1.0"
	fetchDelayBetween = 1 * time.Second
)

// PageFetcher handles fetching web pages and extracting relevant content.
type PageFetcher struct {
	client    *http.Client
	llmClient ChatClient
	mu        sync.Mutex
	lastFetch time.Time
}

// NewPageFetcher creates a page fetcher with the given LLM client for extraction.
func NewPageFetcher(llmClient ChatClient) *PageFetcher {
	return &PageFetcher{
		client: &http.Client{
			Timeout: fetchTimeout,
		},
		llmClient: llmClient,
	}
}

// FetchAndExtract fetches a URL, extracts readable content, and uses an LLM
// to pull out the parts relevant to the given question.
func (f *PageFetcher) FetchAndExtract(ctx context.Context, rawURL, question string) (string, error) {
	// Rate limit: wait between fetches
	f.mu.Lock()
	if !f.lastFetch.IsZero() {
		elapsed := time.Since(f.lastFetch)
		if elapsed < fetchDelayBetween {
			f.mu.Unlock()
			time.Sleep(fetchDelayBetween - elapsed)
			f.mu.Lock()
		}
	}
	f.lastFetch = time.Now()
	f.mu.Unlock()

	// Fetch
	body, err := f.fetchPage(ctx, rawURL)
	if err != nil {
		return "", err
	}

	// Extract readable text
	text, err := extractReadableText(rawURL, body)
	if err != nil {
		return "", fmt.Errorf("could not extract readable content from this page")
	}

	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("could not extract readable content from this page")
	}

	// Truncate for LLM context
	if len(text) > extractionMaxLen {
		text = text[:extractionMaxLen]
	}

	// LLM extraction
	if f.llmClient == nil {
		return text, nil
	}

	extracted, err := f.llmExtract(ctx, text, question)
	if err != nil {
		slog.Warn("LLM extraction failed, returning raw text", "error", err)
		return text, nil
	}

	return extracted, nil
}

func (f *PageFetcher) fetchPage(ctx context.Context, rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %s", rawURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("could not fetch page: %v", err)
	}
	req.Header.Set("User-Agent", fetchUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not fetch page: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not fetch page: HTTP %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "text/html") && !strings.Contains(contentType, "application/xhtml") {
		return "", fmt.Errorf("URL does not point to a web page (content-type: %s)", contentType)
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBody))
	if err != nil {
		return "", fmt.Errorf("could not read page: %v", err)
	}

	return string(respBody), nil
}

func extractReadableText(pageURL, html string) (string, error) {
	parsed, err := url.Parse(pageURL)
	if err != nil {
		return "", err
	}

	article, err := readability.FromReader(strings.NewReader(html), parsed)
	if err != nil {
		return "", err
	}

	text := article.TextContent
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("no readable content extracted")
	}

	return text, nil
}

func (f *PageFetcher) llmExtract(ctx context.Context, pageText, question string) (string, error) {
	prompt := fmt.Sprintf(`You are a research assistant extracting information from a web page.

The user is researching: %s

Extract only the parts of this page that are relevant to that question.
Be concise — return key facts, findings, and quotes. Omit navigation,
ads, and unrelated content. If the page has nothing relevant, say so.

Page content:
%s`, question, pageText)

	messages := []ollamaapi.Message{
		{Role: "user", Content: prompt},
	}

	stream := false
	req := &ollamaapi.ChatRequest{
		Model:    extractionModel,
		Messages: messages,
		Stream:   &stream,
	}

	var result strings.Builder
	err := f.llmClient.Chat(ctx, req, func(resp ollamaapi.ChatResponse) error {
		result.WriteString(resp.Message.Content)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("extraction LLM call failed: %w", err)
	}

	extracted := strings.TrimSpace(result.String())
	if extracted == "" {
		return "", fmt.Errorf("extraction returned empty result")
	}

	return extracted, nil
}
