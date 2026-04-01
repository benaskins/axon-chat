package chat

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
	fact "github.com/benaskins/axon-fact"
)

type userCreatedHandler struct {
	eventStore   fact.EventStore
	defaultModel string
}

type userCreatedRequest struct {
	UserID string `json:"user_id"`
}

func (h *userCreatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req, ok := axon.DecodeJSON[userCreatedRequest](w, r)
	if !ok {
		return
	}

	if req.UserID == "" {
		axon.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	if err := emitEvent(r.Context(), h.eventStore, "user-"+req.UserID, UserCreated(req), nil); err != nil {
		slog.Error("failed to create user", "user_id", req.UserID, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	if err := EmitDefaultAgents(r.Context(), h.eventStore, req.UserID, h.defaultModel); err != nil {
		slog.Error("failed to create default agents", "user_id", req.UserID, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to create default agents")
		return
	}

	slog.Info("user created", "user_id", req.UserID)
	axon.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// EmitDefaultAgents emits agent.created events for the default agent set.
func EmitDefaultAgents(ctx context.Context, es fact.EventStore, userID, defaultModel string) error {
	for _, agent := range defaultAgentSet(userID, defaultModel) {
		stream := "agent-" + userID + "-" + agent.Slug
		evt, err := NewEvent(stream, AgentCreated{Agent: agent})
		if err != nil {
			return err
		}
		if err := es.Append(ctx, stream, []fact.Event{evt}); err != nil {
			return err
		}
	}
	return nil
}

func defaultAgentSet(userID, defaultModel string) []Agent {
	return []Agent{
		{
			UserID:       userID,
			Slug:         "general",
			Name:         "General",
			Tagline:      "General purpose chat",
			AvatarEmoji:  "\U0001F916",
			SystemPrompt: "You handle a wide range of questions and tasks.\n\nClear, direct, and accurate.",
			DefaultModel: defaultModel,
			Tools:        []string{},
		},
		{
			UserID:       userID,
			Slug:         "writer",
			Name:         "Writer",
			Tagline:      "Long-form writing and editing",
			AvatarEmoji:  "\u270D\uFE0F",
			SystemPrompt: "You draft and edit long-form text — articles, documentation, proposals, and prose.\n\nStructured, clear, and expressive when appropriate.",
			DefaultModel: defaultModel,
			Tools:        []string{},
		},
	}
}
