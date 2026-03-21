package chat

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/benaskins/axon"
	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// SyncChatResponse is the JSON response from the synchronous chat endpoint.
type SyncChatResponse struct {
	Response   string   `json:"response"`
	Thinking   string   `json:"thinking,omitempty"`
	DurationMs int64    `json:"duration_ms"`
	ToolsUsed  []string `json:"tools_used"`
}

// syncChatHandler handles POST /api/chat/sync — a non-streaming version of /api/chat
// that returns a complete JSON response.
type syncChatHandler struct {
	chat *chatHandler
}

func (h *syncChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxChatRequestSize)
	var req chatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		axon.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Messages) == 0 {
		axon.WriteError(w, http.StatusBadRequest, "messages must not be empty")
		return
	}

	// Verify agent ownership
	if req.AgentSlug != "" && h.chat.store != nil {
		userID := axon.UserID(r.Context())
		if _, err := h.chat.store.GetAgentByUser(r.Context(), userID, req.AgentSlug); err != nil {
			if errors.Is(err, ErrAgentNotFound) {
				axon.WriteError(w, http.StatusNotFound, "agent not found")
			} else {
				slog.Error("failed to verify agent", "slug", req.AgentSlug, "error", err)
				axon.WriteError(w, http.StatusInternalServerError, "failed to verify agent")
			}
			return
		}
	}

	model := req.Model
	if model == "" {
		model = h.chat.defaultModel
	}

	if h.chat.llm == nil {
		axon.WriteError(w, http.StatusInternalServerError, "LLM client not configured")
		return
	}

	start := time.Now()

	var systemPrompt string
	if len(req.Messages) > 0 && req.Messages[0].Role == "system" {
		systemPrompt = req.Messages[0].Content
	}

	// Extract last user message content
	var userContent string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userContent = req.Messages[i].Content
			break
		}
	}

	// Build tool map from requested tool names
	tools := h.chat.buildToolMap(req.Tools, func(sseEvent) {}, r, req.AgentSlug, req.ConversationID, systemPrompt)

	// Use tool router if available
	if h.chat.toolRouter != nil && len(tools) > 0 && userContent != "" {
		toolDefs := toolDefsFromMap(tools)
		routed, err := h.chat.toolRouter.Route(r.Context(), userContent, toolDefs)
		if err != nil {
			slog.Warn("tool router error, proceeding with all tools", "error", err)
		} else {
			tools = filterToolNames(tools, routed)
		}
	}

	// Build agent request
	agentReq := &loop.Request{
		Model:    model,
		Messages: req.Messages,
		Stream:   false,
		Think:    req.Think,
		Options:  req.Options,
	}

	// Build tool context
	userID := axon.UserID(r.Context())
	username := axon.Username(r.Context())
	toolCtx := &tool.ToolContext{
		Ctx:            r.Context(),
		UserID:         userID,
		Username:       username,
		AgentSlug:      req.AgentSlug,
		ConversationID: req.ConversationID,
		SystemPrompt:   systemPrompt,
	}

	// Accumulate response instead of streaming
	var content strings.Builder
	var thinking strings.Builder
	var toolsUsed []string

	cb := loop.Callbacks{
		OnToken: func(token string) {
			content.WriteString(token)
		},
		OnThinking: func(token string) {
			thinking.WriteString(token)
		},
		OnToolUse: func(name string, args map[string]any) {
			toolsUsed = append(toolsUsed, name)
		},
		OnDone: func(durationMs int64) {},
	}

	_, err := loop.Run(r.Context(), loop.RunConfig{
		Client:    h.chat.llm,
		Request:   agentReq,
		Tools:     tools,
		ToolCtx:   toolCtx,
		Callbacks: cb,
	})
	if err != nil {
		chatLLMCallsTotal.Add(r.Context(), 1, metric.WithAttributes(
			attribute.String("model", model), attribute.String("status", "error"),
		))
		slog.Error("sync chat agent error", "error", err, "model", model)
		axon.WriteError(w, http.StatusInternalServerError, "chat request failed")
		return
	}

	// Metrics
	duration := time.Since(start).Seconds()
	ctx := r.Context()
	modelAttr := metric.WithAttributes(attribute.String("model", model))
	chatLLMCallDuration.Record(ctx, duration, modelAttr)
	chatLLMCallsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("model", model), attribute.String("status", "ok"),
	))
	chatLLMTokensTotal.Add(ctx, float64(content.Len()), modelAttr)

	durationMs := time.Since(start).Milliseconds()

	// Analytics
	if h.chat.analytics != nil && req.AgentSlug != "" {
		runID := axon.RunID(r.Context())
		userEvt := MessageEvent(req.AgentSlug, userID, req.ConversationID, "user", 0)
		userEvt.RunID = runID
		assistantEvt := MessageEvent(req.AgentSlug, userID, req.ConversationID, "assistant", durationMs)
		assistantEvt.RunID = runID

		events := []AnalyticsEvent{userEvt, assistantEvt}
		for _, t := range toolsUsed {
			evt := ToolInvocationEvent(req.AgentSlug, userID, req.ConversationID, t, true, 0)
			evt.RunID = runID
			events = append(events, evt)
		}
		h.chat.analytics.Emit(events...)
	}

	// Persist via events — projectors update the read model
	if req.ConversationID != "" && h.chat.eventStore != nil {
		stream := "conversation-" + req.ConversationID

		if err := h.chat.emit(r.Context(), stream, MessageAppended{Role: "user", Content: userContent}, nil); err != nil {
			slog.Error("failed to emit user message event", "error", err, "conversation_id", req.ConversationID)
		}
		if err := h.chat.emit(r.Context(), stream, MessageAppended{
			Role: "assistant", Content: content.String(),
			Thinking: thinking.String(), DurationMs: &durationMs,
		}, nil); err != nil {
			slog.Error("failed to emit assistant message event", "error", err, "conversation_id", req.ConversationID)
		}

		if h.chat.idleExtractor != nil && req.AgentSlug != "" {
			h.chat.idleExtractor.Touch(req.ConversationID, req.AgentSlug, userID)
		}

		conv, err := h.chat.store.GetConversationByUser(r.Context(), userID, req.ConversationID)
		if err == nil && conv.Title == nil {
			h.chat.bgTasks.Add(1)
			go func() {
				defer h.chat.bgTasks.Done()
				h.chat.generateTitle(h.chat.shutdownCtx, userID, req.ConversationID, userContent, content.String())
			}()
		}
	}

	if toolsUsed == nil {
		toolsUsed = []string{}
	}

	resp := SyncChatResponse{
		Response:   content.String(),
		Thinking:   thinking.String(),
		DurationMs: durationMs,
		ToolsUsed:  toolsUsed,
	}

	axon.WriteJSON(w, http.StatusOK, resp)
}
