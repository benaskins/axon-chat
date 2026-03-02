package chat

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalMessagesHandler(t *testing.T) {
	store := newMemoryStore()
	store.CreateUser("u1")
	store.SaveAgent(Agent{UserID: "u1", Slug: "bot", Name: "Bot"})
	convo, _ := store.CreateConversationForUser("u1", "bot")
	store.AppendMessage(convo.ID, Message{Role: "user", Content: "hello"})
	store.AppendMessage(convo.ID, Message{Role: "assistant", Content: "hi"})

	handler := &internalMessagesHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/internal/conversations/"+convo.ID+"/messages", nil)
	req.SetPathValue("id", convo.ID)
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
	handler := &internalMessagesHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/internal/conversations/nonexistent/messages", nil)
	req.SetPathValue("id", "nonexistent")
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
	store.SaveAgent(Agent{
		UserID:       "u1",
		Slug:         "bot",
		Name:         "Bot",
		SystemPrompt: "You are a helpful bot.",
	})

	handler := &internalAgentHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/internal/agents/bot", nil)
	req.SetPathValue("slug", "bot")
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

func TestInternalAgentHandler_NotFound(t *testing.T) {
	store := newMemoryStore()
	handler := &internalAgentHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/internal/agents/missing", nil)
	req.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
