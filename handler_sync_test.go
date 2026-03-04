package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/benaskins/axon"
	ollamaapi "github.com/ollama/ollama/api"
)

// spyAnalytics captures emitted analytics events for testing.
type spyAnalytics struct {
	mu     sync.Mutex
	events []AnalyticsEvent
}

func (s *spyAnalytics) Emit(events ...AnalyticsEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
}

func (s *spyAnalytics) Events() []AnalyticsEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AnalyticsEvent, len(s.events))
	copy(out, s.events)
	return out
}

func TestSyncChatEndpoint_InvalidMethod(t *testing.T) {
	handler := &syncChatHandler{chat: newChatHandler(testModel, nil, nil, context.Background(), nil)}
	req := httptest.NewRequest(http.MethodGet, "/api/chat/sync", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestSyncChatEndpoint_EmptyMessages(t *testing.T) {
	handler := &syncChatHandler{chat: newChatHandler(testModel, nil, nil, context.Background(), nil)}
	body, _ := json.Marshal(map[string]any{"messages": []any{}})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestSyncChatEndpoint_ReturnsJSON(t *testing.T) {
	mockClient := &mockStreamClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Content: "Hello "}, Done: false},
			{Message: ollamaapi.Message{Content: "there!"}, Done: true},
		},
	}

	handler := &syncChatHandler{chat: newChatHandler("test-model", mockClient, nil, context.Background(), nil)}

	body, _ := json.Marshal(chatRequest{
		Messages: []ollamaapi.Message{{Role: "user", Content: "Hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var result SyncChatResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Response != "Hello there!" {
		t.Errorf("Response = %q, want %q", result.Response, "Hello there!")
	}
	if result.DurationMs < 0 {
		t.Errorf("DurationMs = %d, want >= 0", result.DurationMs)
	}
}

func TestSyncChatEndpoint_TracksToolUse(t *testing.T) {
	mockClient := &mockToolClient{
		toolName: "check_weather",
		finalContent: "The weather is sunny.",
	}

	handler := &syncChatHandler{chat: newChatHandler("test-model", mockClient, nil, context.Background(), nil)}

	body, _ := json.Marshal(chatRequest{
		Messages: []ollamaapi.Message{{Role: "user", Content: "What's the weather?"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sync", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var result SyncChatResponse
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if result.Response != "The weather is sunny." {
		t.Errorf("Response = %q, want %q", result.Response, "The weather is sunny.")
	}
	if len(result.ToolsUsed) != 1 || result.ToolsUsed[0] != "check_weather" {
		t.Errorf("ToolsUsed = %v, want [check_weather]", result.ToolsUsed)
	}
}

func TestSyncChatEndpoint_PropagatesRunID(t *testing.T) {
	mockClient := &mockStreamClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Content: "Hi"}, Done: true},
		},
	}

	spy := &spyAnalytics{}
	chat := newChatHandler("test-model", mockClient, nil, context.Background(), nil)
	chat.analytics = spy

	handler := axon.MetaHeaders(&syncChatHandler{chat: chat})

	body, _ := json.Marshal(chatRequest{
		AgentSlug: "xagent",
		Messages:  []ollamaapi.Message{{Role: "user", Content: "Hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sync", bytes.NewReader(body))
	req.Header.Set("X-Axon-Run-Id", "run-test-123")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	events := spy.Events()
	if len(events) == 0 {
		t.Fatal("expected analytics events, got none")
	}

	for _, ev := range events {
		if ev.RunID != "run-test-123" {
			t.Errorf("expected run_id 'run-test-123' on %s event, got %q", ev.Type, ev.RunID)
		}
	}
}

func TestSyncChatEndpoint_PersistsMessages(t *testing.T) {
	mockClient := &mockStreamClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Content: "I'm great!"}, Done: true},
		},
	}

	store := newMemoryStore()
	store.CreateUser("test-user")
	store.SaveAgent(Agent{UserID: "test-user", Slug: "helper", Name: "Helper"})
	conv, _ := store.CreateConversationForUser("test-user", "helper")

	chat := newChatHandler("test-model", mockClient, store, context.Background(), nil)
	handler := &syncChatHandler{chat: chat}

	body, _ := json.Marshal(chatRequest{
		AgentSlug:      "helper",
		ConversationID: conv.ID,
		Messages:       []ollamaapi.Message{{Role: "user", Content: "How are you?"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat/sync", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), axon.UserIDKey, "test-user")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should have persisted both user and assistant messages
	msgs, err := store.GetMessages(conv.ID)
	if err != nil {
		t.Fatalf("GetMessages: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 persisted messages, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "How are you?" {
		t.Errorf("user message: role=%q content=%q", msgs[0].Role, msgs[0].Content)
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "I'm great!" {
		t.Errorf("assistant message: role=%q content=%q", msgs[1].Role, msgs[1].Content)
	}
}

// mockToolClient simulates a tool call then a final response.
type mockToolClient struct {
	toolName     string
	finalContent string
}

func (m *mockToolClient) Chat(ctx context.Context, req *ollamaapi.ChatRequest, fn ollamaapi.ChatResponseFunc) error {
	// First call: return a tool call
	hasToolCall := false
	for _, msg := range req.Messages {
		if msg.Role == "tool" {
			hasToolCall = true
			break
		}
	}

	if !hasToolCall {
		return fn(ollamaapi.ChatResponse{
			Message: ollamaapi.Message{
				Role: "assistant",
				ToolCalls: []ollamaapi.ToolCall{
					{
						Function: ollamaapi.ToolCallFunction{
							Name:      m.toolName,
							Arguments: func() ollamaapi.ToolCallFunctionArguments {
							a := ollamaapi.NewToolCallFunctionArguments()
							a.Set("location", "Melbourne")
							return a
						}(),
						},
					},
				},
			},
			Done: true,
		})
	}

	// Second call: return final content
	return fn(ollamaapi.ChatResponse{
		Message: ollamaapi.Message{
			Role:    "assistant",
			Content: m.finalContent,
		},
		Done: true,
	})
}
