package chat

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/benaskins/axon"
	agent "github.com/benaskins/axon-agent"
	tool "github.com/benaskins/axon-tool"
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
		if _, err := h.chat.store.GetAgentByUser(userID, req.AgentSlug); err != nil {
			axon.WriteError(w, http.StatusNotFound, "agent not found")
			return
		}
	}

	model := req.Model
	if model == "" {
		model = h.chat.defaultModel
	}

	if h.chat.client == nil {
		axon.WriteError(w, http.StatusInternalServerError, "ollama client not configured")
		return
	}

	start := time.Now()

	// Convert messages
	agentMessages := ollamaMessagesToAgent(req.Messages)

	var systemPrompt string
	if len(agentMessages) > 0 && agentMessages[0].Role == "system" {
		systemPrompt = agentMessages[0].Content
	}

	// Extract last user message content
	var userContent string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userContent = req.Messages[i].Content
			break
		}
	}

	// Build tool map from skills
	tools := h.chat.buildToolMap(req.Skills, func(sseEvent) {}, r, req.AgentSlug, req.ConversationID, systemPrompt)

	// Use tool router if available
	if h.chat.toolRouter != nil && len(tools) > 0 && userContent != "" {
		toolDefs := toolDefsFromMap(tools)
		routed, err := h.chat.toolRouter.Route(r.Context(), userContent, toolDefsToOllama(toolDefs))
		if err == nil {
			tools = filterToolMap(tools, routed)
		}
	}

	// Build agent request
	agentReq := &agent.ChatRequest{
		Model:    model,
		Messages: agentMessages,
		Stream:   true,
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

	cb := agent.Callbacks{
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

	_, err := agent.Run(r.Context(), h.chat.adapter, agentReq, tools, toolCtx, cb)
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "agent error: "+err.Error())
		return
	}

	durationMs := time.Since(start).Milliseconds()

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
