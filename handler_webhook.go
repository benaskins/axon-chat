package chat

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
)

type userCreatedHandler struct {
	store        Store
	defaultModel string
}

type userCreatedRequest struct {
	UserID string `json:"user_id"`
}

func (h *userCreatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req userCreatedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		axon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == "" {
		axon.WriteError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	// Create user record
	if err := h.store.CreateUser(req.UserID); err != nil {
		slog.Error("failed to create user", "user_id", req.UserID, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	// Create default agents
	if err := CreateDefaultAgents(h.store, req.UserID, h.defaultModel); err != nil {
		slog.Error("failed to create default agents", "user_id", req.UserID, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to create default agents")
		return
	}

	slog.Info("user created", "user_id", req.UserID)
	axon.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CreateDefaultAgents creates the initial set of agents for a new user.
func CreateDefaultAgents(store Store, userID, defaultModel string) error {
	defaultAgents := []Agent{
		{
			UserID:       userID,
			Slug:         "general",
			Name:         "General",
			Tagline:      "General purpose chat",
			AvatarEmoji:  "\U0001F916",
			SystemPrompt: "You handle a wide range of questions and tasks.\n\nClear, direct, and accurate.",
			DefaultModel: defaultModel,
			Skills:       []string{},
		},
		{
			UserID:       userID,
			Slug:         "writer",
			Name:         "Writer",
			Tagline:      "Long-form writing and editing",
			AvatarEmoji:  "\u270D\uFE0F",
			SystemPrompt: "You draft and edit long-form text — articles, documentation, proposals, and prose.\n\nStructured, clear, and expressive when appropriate.",
			DefaultModel: defaultModel,
			Skills:       []string{},
		},
	}

	for _, agent := range defaultAgents {
		if err := store.SaveAgent(agent); err != nil {
			return err
		}
	}

	return nil
}
