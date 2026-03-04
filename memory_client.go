package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// MemoryClient talks to the memory service over HTTP.
type MemoryClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewMemoryClient creates a client pointing at a memory service instance.
func NewMemoryClient(baseURL string) *MemoryClient {
	return &MemoryClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// RecallMemories implements MemoryRecaller by calling the memory service's recall endpoint.
func (c *MemoryClient) RecallMemories(ctx context.Context, agentSlug, userID, query string, limit int) (*MemoryRecallResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/memory/recall")
	if err != nil {
		return nil, fmt.Errorf("parse URL: %w", err)
	}

	q := u.Query()
	q.Set("agent", agentSlug)
	q.Set("user", userID)
	q.Set("query", query)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("memory recall request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("memory service returned %d", resp.StatusCode)
	}

	var result MemoryRecallResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// ExtractMemories triggers memory extraction for a conversation.
func (c *MemoryClient) ExtractMemories(ctx context.Context, conversationID, agentSlug, userID string) error {
	u := c.baseURL + "/api/memory/extract"

	payload := map[string]string{
		"conversation_id": conversationID,
		"agent_slug":      agentSlug,
		"user_id":         userID,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("memory extract request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("memory service returned %d", resp.StatusCode)
	}

	return nil
}
