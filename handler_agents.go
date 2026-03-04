package chat

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/benaskins/axon"
)

const maxAgentSize = 1 << 20 // 1MB

// BuildSystemPrompt assembles the full system prompt from the freeform system_prompt,
// constraints, and skill guidance sections.
func BuildSystemPrompt(a Agent) string {
	var parts []string

	if sp := strings.TrimSpace(a.SystemPrompt); sp != "" {
		parts = append(parts, sp)
	}

	if c := strings.TrimSpace(a.Constraints); c != "" {
		parts = append(parts, "## Constraints\n"+c)
	}

	// Add tool-use guidance per skill
	for _, skill := range a.Skills {
		switch skill {
		case "web_search":
			parts = append(parts, "## Search\nYou can search the web for current information. Your search queries are automatically refined using your identity and expertise to produce more targeted results. When you use search results, cite your sources with inline markdown links like [Title](url). Search when you need recent facts, news, or details you're unsure about — not every question requires a search.")
		case "use_claude":
			parts = append(parts, "## Self-Modification\nYou can request code changes to yourself using the use_claude tool. Use it when the user asks you to change something about how you work, your UI, your behavior, or your capabilities.\n\nBe transparent: before calling the tool, tell the user what change you're about to request and why. After calling it, confirm what you submitted. Write a clear, specific description for the developer agent — include what to change, where, and the expected outcome. The change takes a few minutes to implement, test, and deploy.")
		case "current_time":
			parts = append(parts, "## Clock\nYou can check the current time. Use it when the user asks what time or date it is.")
		case "check_weather":
			parts = append(parts, "## Weather\nYou can check current weather conditions for any location. Use it when the user asks about the weather, temperature, or conditions somewhere.")
		case "recall_memory":
			parts = append(parts, "## Memory\nYou have a long-term memory that persists across conversations. Use the recall_memory tool when the user mentions something you don't recall, references a past conversation, or when remembering previous context would help you respond better. Your memory includes relationship metrics that reflect how your relationship with this user has evolved over time.")
		}
	}

	return strings.Join(parts, "\n\n")
}

// agentsListHandler serves GET /api/agents.
type agentsListHandler struct {
	store Store
}

func (h *agentsListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())

	summaries, err := h.store.ListAgentsByUser(userID)
	if err != nil {
		slog.Error("failed to list agents", "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}

	axon.WriteJSON(w, http.StatusOK, summaries)
}

// agentDetailHandler serves GET /api/agents/{slug}.
type agentDetailHandler struct {
	store Store
}

func (h *agentDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteError(w, http.StatusBadRequest, "slug is required")
		return
	}

	agent, err := h.store.GetAgentByUser(userID, slug)
	if err != nil {
		axon.WriteError(w, http.StatusNotFound, "agent not found")
		return
	}

	resp := AgentDetailResponse{
		Agent:      *agent,
		FullPrompt: BuildSystemPrompt(*agent),
	}

	axon.WriteJSON(w, http.StatusOK, resp)
}

// agentSaveHandler serves PUT /api/agents/{slug}.
type agentSaveHandler struct {
	store      Store
	taskRunner TaskRunner
}

func (h *agentSaveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())

	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteError(w, http.StatusBadRequest, "slug is required")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxAgentSize)
	var agent Agent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		axon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if agent.Slug != slug {
		axon.WriteError(w, http.StatusBadRequest, "slug in body must match URL")
		return
	}

	agent.UserID = userID

	if !axon.ValidSlug.MatchString(agent.Slug) {
		axon.WriteError(w, http.StatusBadRequest, "slug must be lowercase alphanumeric with hyphens")
		return
	}

	if agent.Name == "" {
		axon.WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	// Clamp temperature to 0.0-2.0
	if agent.Temperature != nil {
		t := *agent.Temperature
		if t < 0 {
			t = 0
		}
		if t > 2.0 {
			t = 2.0
		}
		agent.Temperature = &t
	}

	// Clamp top_p to 0.0-1.0
	if agent.TopP != nil {
		v := *agent.TopP
		if v < 0 {
			v = 0
		}
		if v > 1.0 {
			v = 1.0
		}
		agent.TopP = &v
	}

	// Clamp top_k to non-negative
	if agent.TopK != nil {
		v := *agent.TopK
		if v < 0 {
			v = 0
		}
		agent.TopK = &v
	}

	// Clamp min_p to 0.0-1.0
	if agent.MinP != nil {
		v := *agent.MinP
		if v < 0 {
			v = 0
		}
		if v > 1.0 {
			v = 1.0
		}
		agent.MinP = &v
	}

	// Clamp presence_penalty to 0.0-2.0
	if agent.PresencePenalty != nil {
		v := *agent.PresencePenalty
		if v < 0 {
			v = 0
		}
		if v > 2.0 {
			v = 2.0
		}
		agent.PresencePenalty = &v
	}

	// Clamp max_tokens to non-negative
	if agent.MaxTokens != nil {
		v := *agent.MaxTokens
		if v < 0 {
			v = 0
		}
		agent.MaxTokens = &v
	}

	if err := h.store.SaveAgent(agent); err != nil {
		slog.Error("failed to save agent", "slug", slug, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to save agent")
		return
	}

	// Issue agent identity credentials (non-blocking)
	username := axon.Username(r.Context())
	if h.taskRunner != nil && username != "" {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := h.taskRunner.IssueAgentCert(ctx, agent.Slug, username); err != nil {
				slog.Error("failed to issue agent cert", "slug", agent.Slug, "error", err)
			} else {
				slog.Info("agent cert issued", "slug", agent.Slug, "username", username)
			}
		}()
	}

	resp := AgentDetailResponse{
		Agent:      agent,
		FullPrompt: BuildSystemPrompt(agent),
	}

	axon.WriteJSON(w, http.StatusOK, resp)
}

// agentDeleteHandler serves DELETE /api/agents/{slug}.
type agentDeleteHandler struct {
	store Store
}

func (h *agentDeleteHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())

	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteError(w, http.StatusBadRequest, "slug is required")
		return
	}

	if err := h.store.DeleteAgent(userID, slug); err != nil {
		slog.Error("failed to delete agent", "slug", slug, "error", err)
		axon.WriteError(w, http.StatusInternalServerError, "failed to delete agent")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
