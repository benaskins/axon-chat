package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// AnalyticsEvent is a typed event sent to the analytics service.
type AnalyticsEvent struct {
	Type             string    `json:"type"`
	Timestamp        time.Time `json:"timestamp"`
	AgentSlug        string    `json:"agent_slug,omitempty"`
	UserID           string    `json:"user_id,omitempty"`
	ConversationID   string    `json:"conversation_id,omitempty"`
	RunID            string    `json:"run_id,omitempty"`
	Role             string    `json:"role,omitempty"`
	PromptTokens     uint32    `json:"prompt_tokens,omitempty"`
	CompletionTokens uint32    `json:"completion_tokens,omitempty"`
	DurationMs       uint32    `json:"duration_ms,omitempty"`
	ToolName         string    `json:"tool_name,omitempty"`
	Success          *bool     `json:"success,omitempty"`
	EventName        string    `json:"event_name,omitempty"`
}

// AnalyticsEmitter sends analytics events to the analytics service.
type AnalyticsEmitter interface {
	Emit(events ...AnalyticsEvent)
}

// AnalyticsClient sends events to the analytics service over HTTP.
// Events are sent asynchronously — failures are logged but never block.
type AnalyticsClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAnalyticsClient creates a client for the analytics service.
func NewAnalyticsClient(baseURL string) *AnalyticsClient {
	return &AnalyticsClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// Emit sends events to the analytics service asynchronously.
func (c *AnalyticsClient) Emit(events ...AnalyticsEvent) {
	go func() {
		body, err := json.Marshal(events)
		if err != nil {
			slog.Error("analytics: failed to marshal events", "error", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/events", bytes.NewReader(body))
		if err != nil {
			slog.Error("analytics: failed to create request", "error", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			slog.Warn("analytics: failed to send events", "error", err)
			return
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			slog.Warn("analytics: unexpected status", "status", resp.StatusCode)
		}
	}()
}

// NoopAnalytics discards all events. Used when analytics is not configured.
type NoopAnalytics struct{}

func (NoopAnalytics) Emit(events ...AnalyticsEvent) {}

// MessageEvent creates a message analytics event.
func MessageEvent(agentSlug, userID, conversationID, role string, durationMs int64) AnalyticsEvent {
	return AnalyticsEvent{
		Type:           "message",
		Timestamp:      time.Now(),
		AgentSlug:      agentSlug,
		UserID:         userID,
		ConversationID: conversationID,
		Role:           role,
		DurationMs:     uint32(durationMs),
	}
}

// ToolInvocationEvent creates a tool invocation analytics event.
func ToolInvocationEvent(agentSlug, userID, conversationID, toolName string, success bool, durationMs int64) AnalyticsEvent {
	return AnalyticsEvent{
		Type:           "tool_invocation",
		Timestamp:      time.Now(),
		AgentSlug:      agentSlug,
		UserID:         userID,
		ConversationID: conversationID,
		ToolName:       toolName,
		Success:        &success,
		DurationMs:     uint32(durationMs),
	}
}

// ConversationEvent creates a conversation lifecycle analytics event.
func ConversationEvent(agentSlug, userID, conversationID, eventName string) AnalyticsEvent {
	return AnalyticsEvent{
		Type:           fmt.Sprintf("conversation_%s", eventName),
		Timestamp:      time.Now(),
		AgentSlug:      agentSlug,
		UserID:         userID,
		ConversationID: conversationID,
		EventName:      eventName,
	}
}
