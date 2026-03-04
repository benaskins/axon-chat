package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ollamaapi "github.com/ollama/ollama/api"
)

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
