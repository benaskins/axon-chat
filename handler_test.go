package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChatEndpoint_InvalidMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/chat", nil)
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestChatEndpoint_EmptyMessages(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"messages": []any{}})
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChatEndpoint_InvalidBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/chat", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()

	handler := newChatHandler(testModel, nil, nil, nil, nil, context.Background(), nil)
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

	handler := newChatHandler(testModel, nil, nil, nil, nil, context.Background(), nil)
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
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
