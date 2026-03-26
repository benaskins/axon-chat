package chat

import (
	"crypto/subtle"
	"errors"
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
)

// requireInternalKey validates the X-Internal-API-Key header using
// constant-time comparison. Returns true if the request is authorised.
func requireInternalKey(w http.ResponseWriter, r *http.Request, key string) bool {
	if key == "" {
		axon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	provided := r.Header.Get("X-Internal-API-Key")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(key)) != 1 {
		axon.WriteError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	return true
}

// internalMessagesHandler serves conversation messages for
// internal service-to-service calls (e.g. memory service).
// Requires a valid X-Internal-API-Key header.
type internalMessagesHandler struct {
	store       Store
	internalKey string
}

func (h *internalMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requireInternalKey(w, r, h.internalKey) {
		return
	}

	conversationID := r.PathValue("id")
	if conversationID == "" {
		axon.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing conversation id"})
		return
	}

	messages, err := h.store.GetMessages(r.Context(), conversationID)
	if err != nil {
		slog.Error("internal messages: failed to get messages", "error", err, "conversation_id", conversationID)
		axon.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get messages"})
		return
	}

	axon.WriteJSON(w, http.StatusOK, messages)
}

// internalAgentHandler serves agent info by slug for
// internal service-to-service calls.
// Requires a valid X-Internal-API-Key header.
type internalAgentHandler struct {
	store       Store
	internalKey string
}

// internalAgentResponse is the JSON shape returned by the internal agent endpoint.
type internalAgentResponse struct {
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"`
}

func (h *internalAgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !requireInternalKey(w, r, h.internalKey) {
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent slug"})
		return
	}

	agent, err := h.store.GetAgentBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			axon.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		} else {
			axon.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to get agent"})
		}
		return
	}

	axon.WriteJSON(w, http.StatusOK, internalAgentResponse{
		Name:         agent.Name,
		SystemPrompt: agent.SystemPrompt,
	})
}
