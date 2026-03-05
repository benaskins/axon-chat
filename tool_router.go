package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
)

// ToolRouter uses a small LLM to decide which tools (if any) are appropriate
// for a given user message, preventing the conversational model from being
// biased toward calling tools on every turn.
type ToolRouter struct {
	llm   loop.LLMClient
	model string
}

// toolRouterResponse is the structured output schema for the tool router.
type toolRouterResponse struct {
	Tools []string `json:"tools"`
}

// NewToolRouter creates a new tool router.
func NewToolRouter(llm loop.LLMClient, model string) *ToolRouter {
	return &ToolRouter{
		llm:   llm,
		model: model,
	}
}

// Route evaluates the user's latest message against available tools and returns
// only the tool names that should be offered to the conversational model.
// On error or timeout, returns nil (safe default — no tools).
func (tr *ToolRouter) Route(ctx context.Context, userMessage string, tools []tool.ToolDef) ([]string, error) {
	if len(tools) == 0 {
		return nil, nil
	}

	// Build tool descriptions
	var toolList strings.Builder
	var names []string
	for _, t := range tools {
		names = append(names, t.Name)
		fmt.Fprintf(&toolList, "- %s: %s\n", t.Name, t.Description)
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
	var response strings.Builder

	err := tr.llm.Chat(routeCtx, &loop.Request{
		Model:    tr.model,
		Messages: []loop.Message{{Role: "user", Content: prompt}},
	}, func(resp loop.Response) error {
		response.WriteString(resp.Content)
		return nil
	})

	latency := time.Since(start).Milliseconds()

	if err != nil {
		slog.Warn("tool router failed, defaulting to no tools", "error", err, "latency_ms", latency)
		return nil, err
	}

	nameSet := make(map[string]bool)
	for _, n := range names {
		nameSet[n] = true
	}

	selected := parseToolResponse(response.String(), nameSet)

	slog.Info("tool router decision",
		"user_message", userMessage,
		"raw_response", response.String(),
		"selected_tools", selected,
		"latency_ms", latency,
	)

	return selected, nil
}

// parseToolResponse parses the structured JSON response and returns matching tool names.
func parseToolResponse(response string, available map[string]bool) []string {
	var result toolRouterResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		slog.Warn("tool router response not valid JSON, falling back to scan", "response", response, "error", err)
		return scanForToolNames(response, available)
	}

	var selected []string
	seen := make(map[string]bool)
	for _, name := range result.Tools {
		if available[name] && !seen[name] {
			selected = append(selected, name)
			seen[name] = true
		}
	}
	return selected
}

// scanForToolNames is a fallback that scans the response for known tool names.
func scanForToolNames(response string, available map[string]bool) []string {
	cleaned := strings.ToLower(response)
	var selected []string
	for name := range available {
		if strings.Contains(cleaned, name) {
			selected = append(selected, name)
		}
	}
	return selected
}
