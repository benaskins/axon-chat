package chat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAnalyticsClient_Emit(t *testing.T) {
	var mu sync.Mutex
	var receivedEvents []AnalyticsEvent

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var events []AnalyticsEvent
		json.NewDecoder(r.Body).Decode(&events)
		mu.Lock()
		receivedEvents = append(receivedEvents, events...)
		mu.Unlock()
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewAnalyticsClient(server.URL)
	client.Emit(
		MessageEvent("helper", "user1", "conv-1", "user", 0),
		MessageEvent("helper", "user1", "conv-1", "assistant", 3200),
	)

	// Wait for async send
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(receivedEvents) != 2 {
		t.Fatalf("expected 2 events, got %d", len(receivedEvents))
	}
	if receivedEvents[0].Role != "user" {
		t.Errorf("expected user role, got %s", receivedEvents[0].Role)
	}
	if receivedEvents[1].DurationMs != 3200 {
		t.Errorf("expected 3200ms duration, got %d", receivedEvents[1].DurationMs)
	}
}

func TestAnalyticsClient_EmitServerDown(t *testing.T) {
	// Client should not panic when server is unreachable
	client := NewAnalyticsClient("http://localhost:1")
	client.Emit(MessageEvent("helper", "user1", "conv-1", "user", 0))

	// Just verify it doesn't block or panic
	time.Sleep(100 * time.Millisecond)
}

func TestMessageEvent(t *testing.T) {
	e := MessageEvent("bot", "u1", "c1", "assistant", 5000)
	if e.Type != "message" {
		t.Errorf("expected type 'message', got %s", e.Type)
	}
	if e.DurationMs != 5000 {
		t.Errorf("expected 5000, got %d", e.DurationMs)
	}
}

func TestToolInvocationEvent(t *testing.T) {
	e := ToolInvocationEvent("bot", "u1", "c1", "web_search", true, 850)
	if e.Type != "tool_invocation" {
		t.Errorf("expected type 'tool_invocation', got %s", e.Type)
	}
	if e.ToolName != "web_search" {
		t.Errorf("expected web_search, got %s", e.ToolName)
	}
	if *e.Success != true {
		t.Error("expected success=true")
	}
}

func TestConversationEvent(t *testing.T) {
	e := ConversationEvent("bot", "u1", "c1", "started")
	if e.Type != "conversation_started" {
		t.Errorf("expected type 'conversation_started', got %s", e.Type)
	}
}
