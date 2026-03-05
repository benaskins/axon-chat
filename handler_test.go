package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	loop "github.com/benaskins/axon-loop"
)

func TestChatEndpoint_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestChatEndpoint_EmptyMessages(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"messages": []any{}})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChatEndpoint_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChatEndpoint_NoClient(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestChatEndpoint_StreamsViaAgentRun(t *testing.T) {
	mockClient := &mockStreamClient{
		responses: []loop.Response{
			{Content: "Hello "},
			{Content: "there!", Done: true},
		},
	}

	handler := newChatHandler("test-model", mockClient, nil, context.Background(), nil)

	body, _ := json.Marshal(chatRequest{
		Messages: []loop.Message{{Role: "user", Content: "Hi"}},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Should have SSE content-type
	ct := w.Header().Get("Content-Type")
	if ct != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}

	// Body should contain SSE data frames with content
	respBody := w.Body.String()
	if !strings.Contains(respBody, "Hello ") {
		t.Errorf("response body should contain 'Hello ', got:\n%s", respBody)
	}
	if !strings.Contains(respBody, `"done":true`) {
		t.Errorf("response body should contain done event, got:\n%s", respBody)
	}
}

type mockStreamClient struct {
	responses []loop.Response
}

func (m *mockStreamClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	for _, resp := range m.responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

func TestSanitizeTitle(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"Hello World"`, "Hello World"},
		{`'Hello World'`, "Hello World"},
		{`**Bold Title**`, "Bold Title"},
		{`  spaced  `, "spaced"},
		{`normal title`, "normal title"},
	}
	for _, tt := range tests {
		got := sanitizeTitle(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeTitle(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
