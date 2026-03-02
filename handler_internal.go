package chat

import (
	"net/http"

	"github.com/benaskins/axon"
)

// internalMessagesHandler serves conversation messages without auth for
// internal service-to-service calls (e.g. memory service).
type internalMessagesHandler struct {
	store Store
}

func (h *internalMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conversationID := r.PathValue("id")
	if conversationID == "" {
		axon.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing conversation id"})
		return
	}

	messages, err := h.store.GetMessages(conversationID)
	if err != nil {
		axon.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	axon.WriteJSON(w, http.StatusOK, messages)
}

// internalAgentHandler serves agent info by slug without auth for
// internal service-to-service calls.
type internalAgentHandler struct {
	store Store
}

// internalAgentResponse is the JSON shape returned by the internal agent endpoint.
type internalAgentResponse struct {
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"`
}

func (h *internalAgentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent slug"})
		return
	}

	agent, err := h.store.GetAgentBySlug(slug)
	if err != nil {
		axon.WriteJSON(w, http.StatusNotFound, map[string]string{"error": "agent not found"})
		return
	}

	axon.WriteJSON(w, http.StatusOK, internalAgentResponse{
		Name:         agent.Name,
		SystemPrompt: agent.SystemPrompt,
	})
}
