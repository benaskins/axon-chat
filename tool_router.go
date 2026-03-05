package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	ollamaapi "github.com/ollama/ollama/api"
)

// ToolRouter uses a small LLM to decide which tools (if any) are appropriate
// for a given user message, preventing the conversational model from being
// biased toward calling tools on every turn.
type ToolRouter struct {
	client ChatClient
	model  string
}

// toolRouterResponse is the structured output schema for the tool router.
type toolRouterResponse struct {
	Tools []string `json:"tools"`
}

// NewToolRouter creates a new tool router.
func NewToolRouter(client ChatClient, model string) *ToolRouter {
	return &ToolRouter{
		client: client,
		model:  model,
	}
}

// buildFormatSchema builds a JSON schema that constrains the LLM output
// to {"tools": ["tool_name", ...]} where tool names are from the enum.
func buildFormatSchema(toolNames []string) json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tools": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
					"enum": toolNames,
				},
			},
		},
		"required": []string{"tools"},
	}
	b, _ := json.Marshal(schema)
	return b
}

// Route evaluates the user's latest message against available tools and returns
// only the tools that should be offered to the conversational model.
// On error or timeout, returns empty tools (safe default).
func (tr *ToolRouter) Route(ctx context.Context, userMessage string, tools ollamaapi.Tools) (ollamaapi.Tools, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Build tool descriptions and index
	var toolList strings.Builder
	toolIndex := make(map[string]ollamaapi.Tool)
	var names []string
	for _, t := range tools {
		name := t.Function.Name
		toolIndex[name] = t
		names = append(names, name)
		fmt.Fprintf(&toolList, "- %s: %s\n", name, t.Function.Description)
	}

	prompt := fmt.Sprintf(`You decide which tools (if any) should be used for this conversation turn.
Return a JSON object with a "tools" array containing the tool names to use. Use an empty array if no tools are needed.

Most messages need no tools. Only select a tool when the user's message clearly calls for it.
- use_claude: ONLY when the user explicitly asks to change, modify, or fix something about the agent's code, UI, config, or behavior. Never for general conversation, topic changes, or compliments.
- take_photo: ONLY when the user asks for an image or photo.
- web_search: ONLY when the user asks to look something up or needs current information.
- current_time: ONLY when the user asks what time or date it is.
- check_weather: ONLY when the user asks about weather, temperature, or conditions for a location.

Examples:
User: "Hey, how's your day going?" → {"tools": []}
User: "Can you take a photo of yourself?" → {"tools": ["take_photo"]}
User: "What's the latest news on AI?" → {"tools": ["web_search"]}
User: "Could you add a clock to your interface?" → {"tools": ["use_claude"]}
User: "Take a photo and search for similar images" → {"tools": ["take_photo", "web_search"]}
User: "That's really cool, thanks!" → {"tools": []}
User: "What time is it?" → {"tools": ["current_time"]}
User: "What's the weather like in Tokyo?" → {"tools": ["check_weather"]}

Available tools:
%s
User message: %s`, toolList.String(), userMessage)

	routeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	stream := false
	var response strings.Builder

	err := tr.client.Chat(routeCtx, &ollamaapi.ChatRequest{
		Model:    tr.model,
		Messages: []ollamaapi.Message{{Role: "user", Content: prompt}},
		Stream:   &stream,
		Format:   buildFormatSchema(names),
	}, func(resp ollamaapi.ChatResponse) error {
		response.WriteString(resp.Message.Content)
		return nil
	})

	latency := time.Since(start).Milliseconds()

	if err != nil {
		slog.Warn("tool router failed, defaulting to no tools", "error", err, "latency_ms", latency)
		return nil, err
	}

	selected := parseToolResponse(response.String(), toolIndex)

	slog.Info("tool router decision",
		"user_message", userMessage,
		"raw_response", response.String(),
		"selected_tools", toolNamesFromTools(selected),
		"latency_ms", latency,
	)

	return selected, nil
}

// parseToolResponse parses the structured JSON response and returns matching tools.
func parseToolResponse(response string, available map[string]ollamaapi.Tool) ollamaapi.Tools {
	var result toolRouterResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		slog.Warn("tool router response not valid JSON, falling back to scan", "response", response, "error", err)
		return scanForToolNames(response, available)
	}

	var selected ollamaapi.Tools
	seen := make(map[string]bool)
	for _, name := range result.Tools {
		if tool, ok := available[name]; ok && !seen[name] {
			selected = append(selected, tool)
			seen[name] = true
		}
	}
	return selected
}

// scanForToolNames is a fallback that scans the response for known tool names.
func scanForToolNames(response string, available map[string]ollamaapi.Tool) ollamaapi.Tools {
	cleaned := strings.ToLower(response)
	var selected ollamaapi.Tools
	for name, tool := range available {
		if strings.Contains(cleaned, name) {
			selected = append(selected, tool)
		}
	}
	return selected
}

// toolNamesFromTools returns a slice of tool names for logging.
func toolNamesFromTools(tools ollamaapi.Tools) []string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Function.Name
	}
	return names
}
