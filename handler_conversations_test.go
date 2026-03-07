package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benaskins/axon"
)

func ctxWithUser(userID string) context.Context {
	return context.WithValue(context.Background(), axon.UserIDKey, userID)
}

func TestConversationListHandler_ReturnsConversations(t *testing.T) {
	store := newMemoryStore()
	store.CreateUser("u1")
	store.SaveAgent(Agent{UserID: "u1", Slug: "bot", Name: "Bot"})
	store.CreateConversationForUser("u1", "bot")
	store.CreateConversationForUser("u1", "bot")

	handler := &conversationListHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/bot/conversations", nil)
	req = req.WithContext(ctxWithUser("u1"))
	req.SetPathValue("slug", "bot")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var convos []ConversationSummary
	if err := json.NewDecoder(w.Body).Decode(&convos); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(convos) != 2 {
		t.Errorf("expected 2 conversations, got %d", len(convos))
	}
}

func TestConversationListHandler_EmptyList(t *testing.T) {
	store := newMemoryStore()

	handler := &conversationListHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/bot/conversations", nil)
	req = req.WithContext(ctxWithUser("u1"))
	req.SetPathValue("slug", "bot")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var convos []ConversationSummary
	json.NewDecoder(w.Body).Decode(&convos)
	if len(convos) != 0 {
		t.Errorf("expected 0 conversations, got %d", len(convos))
	}
}

func TestConversationListHandler_WrongMethod(t *testing.T) {
	store := newMemoryStore()
	handler := &conversationListHandler{store: store}

	req := httptest.NewRequest(http.MethodPost, "/api/agents/bot/conversations", nil)
	req = req.WithContext(ctxWithUser("u1"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestConversationCreateHandler_CreatesConversation(t *testing.T) {
	store := newMemoryStore()
	store.CreateUser("u1")
	store.SaveAgent(Agent{UserID: "u1", Slug: "bot", Name: "Bot"})

	handler := &conversationCreateHandler{store: store}

	req := httptest.NewRequest(http.MethodPost, "/api/agents/bot/conversations", nil)
	req = req.WithContext(ctxWithUser("u1"))
	req.SetPathValue("slug", "bot")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var conv Conversation
	if err := json.NewDecoder(w.Body).Decode(&conv); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if conv.ID == "" {
		t.Error("expected conversation ID to be set")
	}
	if conv.AgentSlug != "bot" {
		t.Errorf("expected agent_slug bot, got %s", conv.AgentSlug)
	}
}

func TestConversationCreateHandler_AgentNotFound(t *testing.T) {
	store := newMemoryStore()
	store.CreateUser("u1")

	handler := &conversationCreateHandler{store: store}

	req := httptest.NewRequest(http.MethodPost, "/api/agents/missing/conversations", nil)
	req = req.WithContext(ctxWithUser("u1"))
	req.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestConversationCreateHandler_WrongMethod(t *testing.T) {
	store := newMemoryStore()
	handler := &conversationCreateHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/api/agents/bot/conversations", nil)
	req = req.WithContext(ctxWithUser("u1"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestConversationDeleteHandler_DeletesConversation(t *testing.T) {
	store := newMemoryStore()
	store.CreateUser("u1")
	store.SaveAgent(Agent{UserID: "u1", Slug: "bot", Name: "Bot"})
	conv, _ := store.CreateConversationForUser("u1", "bot")

	handler := &conversationDeleteHandler{store: store}

	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/"+conv.ID, nil)
	req = req.WithContext(ctxWithUser("u1"))
	req.SetPathValue("id", conv.ID)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}

	// Verify it's gone
	_, err := store.GetConversationByUser("u1", conv.ID)
	if err == nil {
		t.Error("expected conversation to be deleted")
	}
}

func TestConversationDeleteHandler_NonexistentReturns404(t *testing.T) {
	store := newMemoryStore()
	handler := &conversationDeleteHandler{store: store}

	req := httptest.NewRequest(http.MethodDelete, "/api/conversations/nonexistent", nil)
	req = req.WithContext(ctxWithUser("u1"))
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestConversationDeleteHandler_WrongMethod(t *testing.T) {
	store := newMemoryStore()
	handler := &conversationDeleteHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/api/conversations/abc", nil)
	req = req.WithContext(ctxWithUser("u1"))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}
