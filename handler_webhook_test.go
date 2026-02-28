package chat

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserCreatedWebhook(t *testing.T) {
	store := newMemoryStore()
	handler := &userCreatedHandler{store: store, defaultModel: testModel}

	body, _ := json.Marshal(map[string]string{"user_id": "test-user"})
	req := httptest.NewRequest(http.MethodPost, "/internal/user-created", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify default agents were created
	agents, err := store.ListAgentsByUser("test-user")
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 default agents, got %d", len(agents))
	}
}

func TestUserCreatedWebhook_MissingUserID(t *testing.T) {
	store := newMemoryStore()
	handler := &userCreatedHandler{store: store, defaultModel: testModel}

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/internal/user-created", bytes.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
