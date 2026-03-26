package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalMessagesHandler(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	store.CreateUser(ctx, "u1")
	store.SaveAgent(ctx, Agent{UserID: "u1", Slug: "bot", Name: "Bot"})
	convo, _ := store.CreateConversationForUser(ctx, "u1", "bot")
	store.AppendMessage(ctx, convo.ID, Message{Role: "user", Content: "hello"})
	store.AppendMessage(ctx, convo.ID, Message{Role: "assistant", Content: "hi"})

	handler := &internalMessagesHandler{store: store, internalKey: "test-key"}

	req := httptest.NewRequest(http.MethodGet, "/internal/conversations/"+convo.ID+"/messages", nil)
	req.SetPathValue("id", convo.ID)
	req.Header.Set("X-Internal-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var msgs []Message
	if err := json.NewDecoder(w.Body).Decode(&msgs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(msgs) != 2 {
		t.Errorf("expected 2 messages, got %d", len(msgs))
	}
}

func TestInternalMessagesHandler_EmptyConversation(t *testing.T) {
	store := newMemoryStore()
	handler := &internalMessagesHandler{store: store, internalKey: "test-key"}

	req := httptest.NewRequest(http.MethodGet, "/internal/conversations/nonexistent/messages", nil)
	req.SetPathValue("id", "nonexistent")
	req.Header.Set("X-Internal-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var msgs []Message
	json.NewDecoder(w.Body).Decode(&msgs)
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages, got %d", len(msgs))
	}
}

func TestInternalAgentHandler(t *testing.T) {
	store := newMemoryStore()
	store.SaveAgent(context.Background(), Agent{
		UserID:       "u1",
		Slug:         "bot",
		Name:         "Bot",
		SystemPrompt: "You are a helpful bot.",
	})

	handler := &internalAgentHandler{store: store, internalKey: "test-key"}

	req := httptest.NewRequest(http.MethodGet, "/internal/agents/bot", nil)
	req.SetPathValue("slug", "bot")
	req.Header.Set("X-Internal-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp internalAgentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Name != "Bot" {
		t.Errorf("expected name Bot, got %s", resp.Name)
	}
	if resp.SystemPrompt != "You are a helpful bot." {
		t.Errorf("expected system prompt, got %s", resp.SystemPrompt)
	}
}

func TestInternalMessagesHandler_Unauthorized(t *testing.T) {
	store := newMemoryStore()
	handler := &internalMessagesHandler{store: store, internalKey: "test-key"}

	// No key
	req := httptest.NewRequest(http.MethodGet, "/internal/conversations/c1/messages", nil)
	req.SetPathValue("id", "c1")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no key: expected 401, got %d", w.Code)
	}

	// Wrong key
	req = httptest.NewRequest(http.MethodGet, "/internal/conversations/c1/messages", nil)
	req.SetPathValue("id", "c1")
	req.Header.Set("X-Internal-API-Key", "wrong-key")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong key: expected 401, got %d", w.Code)
	}
}

func TestInternalAgentHandler_Unauthorized(t *testing.T) {
	store := newMemoryStore()
	handler := &internalAgentHandler{store: store, internalKey: "test-key"}

	req := httptest.NewRequest(http.MethodGet, "/internal/agents/bot", nil)
	req.SetPathValue("slug", "bot")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no key: expected 401, got %d", w.Code)
	}
}

func TestInternalAgentHandler_NotFound(t *testing.T) {
	store := newMemoryStore()
	handler := &internalAgentHandler{store: store, internalKey: "test-key"}

	req := httptest.NewRequest(http.MethodGet, "/internal/agents/missing", nil)
	req.SetPathValue("slug", "missing")
	req.Header.Set("X-Internal-API-Key", "test-key")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
