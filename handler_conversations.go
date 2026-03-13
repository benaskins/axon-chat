package chat

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
	fact "github.com/benaskins/axon-fact"
	"github.com/google/uuid"
)

type conversationListHandler struct {
	store Store
}

func (h *conversationListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	userID := axon.UserID(ctx)
	slug := r.PathValue("slug")
	convos, err := h.store.ListConversationsByUser(ctx, userID, slug)
	if err != nil {
		slog.Error("failed to list conversations", "slug", slug, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to list conversations")
		return
	}

	axon.WriteJSON(w, http.StatusOK, convos)
}

type conversationCreateHandler struct {
	store      Store
	eventStore fact.EventStore
}

func (h *conversationCreateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	userID := axon.UserID(ctx)
	slug := r.PathValue("slug")

	// Verify agent exists
	_, err := h.store.GetAgentByUser(ctx, userID, slug)
	if err != nil {
		if errors.Is(err, ErrAgentNotFound) {
			axon.WriteError(w, http.StatusNotFound, "agent not found")
		} else {
			slog.Error("failed to verify agent", "slug", slug, "error", err)
			axon.WriteError(w, http.StatusInternalServerError, "failed to verify agent")
		}
		return
	}

	convID := uuid.New().String()
	stream := "conversation-" + convID
	if err := emitEvent(r.Context(), h.eventStore, stream, ConversationCreated{ID: convID, AgentSlug: slug, UserID: userID}, nil); err != nil {
		slog.Error("failed to create conversation", "slug", slug, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to create conversation")
		return
	}
	// Read back from projected read model
	conv, err := h.store.GetConversationByUser(ctx, userID, convID)
	if err != nil {
		slog.Error("failed to read conversation", "id", convID, "error", err)
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

	ctx := r.Context()
	userID := axon.UserID(ctx)
	id := r.PathValue("id")

	conv, err := h.store.GetConversationByUser(ctx, userID, id)
	if err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			axon.WriteError(w, http.StatusNotFound, "conversation not found")
		} else {
			slog.Error("failed to get conversation", "id", id, "error", err)
			axon.WriteError(w, http.StatusInternalServerError, "failed to get conversation")
		}
		return
	}

	msgs, err := h.store.GetMessages(ctx, id)
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
	store      Store
	eventStore fact.EventStore
}

func (h *conversationDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	ctx := r.Context()
	userID := axon.UserID(ctx)
	id := r.PathValue("id")

	if _, err := h.store.GetConversationByUser(ctx, userID, id); err != nil {
		if errors.Is(err, ErrConversationNotFound) {
			axon.WriteError(w, http.StatusNotFound, "conversation not found")
		} else {
			slog.Error("failed to get conversation", "id", id, "error", err)
			axon.WriteError(w, http.StatusInternalServerError, "failed to get conversation")
		}
		return
	}

	stream := "conversation-" + id
	meta := map[string]string{"user_id": userID}
	if err := emitEvent(r.Context(), h.eventStore, stream, ConversationDeleted{}, meta); err != nil {
		slog.Error("failed to delete conversation", "id", id, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to delete conversation")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
