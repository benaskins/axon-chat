package chat

import (
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
)

type conversationListHandler struct {
	store Store
}

func (h *conversationListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	slug := r.PathValue("slug")
	convos, err := h.store.ListConversationsByUser(userID, slug)
	if err != nil {
		slog.Error("failed to list conversations", "slug", slug, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}

	axon.WriteJSON(w, http.StatusOK, convos)
}

type conversationCreateHandler struct {
	store Store
}

func (h *conversationCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	slug := r.PathValue("slug")

	// Verify agent exists
	_, err := h.store.GetAgentByUser(userID, slug)
	if err != nil {
		axon.WriteError(w, http.StatusNotFound, "agent not found")
		return
	}

	conv, err := h.store.CreateConversationForUser(userID, slug)
	if err != nil {
		slog.Error("failed to create conversation", "slug", slug, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}

	axon.WriteJSON(w, http.StatusCreated, conv)
}

type conversationGetHandler struct {
	store Store
}

func (h *conversationGetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	id := r.PathValue("id")

	conv, err := h.store.GetConversationByUser(userID, id)
	if err != nil {
		axon.WriteError(w, http.StatusNotFound, "conversation not found")
		return
	}

	msgs, err := h.store.GetMessages(id)
	if err != nil {
		slog.Error("failed to get messages", "id", id, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to get messages")
		return
	}

	resp := struct {
		*Conversation
		Messages []Message `json:"messages"`
	}{conv, msgs}

	axon.WriteJSON(w, http.StatusOK, resp)
}

type conversationDeleteHandler struct {
	store Store
}

func (h *conversationDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	id := r.PathValue("id")

	if _, err := h.store.GetConversationByUser(userID, id); err != nil {
		axon.WriteError(w, http.StatusNotFound, "conversation not found")
		return
	}

	if err := h.store.DeleteConversation(userID, id); err != nil {
		slog.Error("failed to delete conversation", "id", id, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to delete conversation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
