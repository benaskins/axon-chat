package chat

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	tool "github.com/benaskins/axon-tool"
	ollamaapi "github.com/ollama/ollama/api"
)

// buildToolMap builds a map of tool.ToolDef for the given skills.
// Per-request state (sendEvent, imageRefs) is closed over by chat-specific tools.
func (h *chatHandler) buildToolMap(skills []string, sendEvent func(sseEvent), r *http.Request, agentSlug string, conversationID string, systemPrompt string) map[string]tool.ToolDef {
	result := make(map[string]tool.ToolDef)
	for _, skill := range skills {
		if def, ok := h.buildTool(skill, sendEvent, r, agentSlug, conversationID, systemPrompt); ok {
			result[skill] = def
		}
	}
	return result
}

func (h *chatHandler) buildTool(skill string, sendEvent func(sseEvent), r *http.Request, agentSlug string, conversationID string, systemPrompt string) (tool.ToolDef, bool) {
	switch skill {
	case "current_time":
		return tool.CurrentTimeTool(), true
	case "web_search":
		return h.webSearchTool(r, systemPrompt), true
	case "fetch_page":
		return h.fetchPageTool(r), true
	case "check_weather":
		return h.checkWeatherTool(r), true
	case "use_claude":
		return h.useClaudeTool(sendEvent, r, agentSlug, conversationID), true
	case "recall_memory":
		return h.recallMemoryTool(), true
	default:
		if def, ok := h.extraTools[skill]; ok {
			return def, true
		}
		return tool.ToolDef{}, false
	}
}

func (h *chatHandler) webSearchTool(r *http.Request, systemPrompt string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "web_search",
		Description: "Search the web for current information, news, facts, or anything not in your training data. Use when the user asks about recent events, specific facts you're unsure about, or when they explicitly ask you to look something up.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]tool.PropertySchema{
				"query": {Type: "string", Description: "The search query to look up on the web."},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			queryStr, _ := args["query"].(string)
			if h.searcher == nil || queryStr == "" {
				return tool.ToolResult{Content: "Search is not available."}
			}

			searchQuery := queryStr
			if h.searchQualifier != nil && ctx.SystemPrompt != "" {
				searchQuery = h.searchQualifier.Qualify(ctx.Ctx, queryStr, ctx.SystemPrompt)
			}

			results, err := h.searcher.Search(ctx.Ctx, searchQuery, 5)
			if err != nil {
				slog.Error("web search failed", "error", err, "query", searchQuery)
				return tool.ToolResult{Content: fmt.Sprintf("Search failed: %v", err)}
			}
			if len(results) == 0 {
				return tool.ToolResult{Content: fmt.Sprintf("No results found for %q.", queryStr)}
			}

			var sb strings.Builder
			if searchQuery != queryStr {
				sb.WriteString(fmt.Sprintf("Search results for %q (refined from %q):\n\n", searchQuery, queryStr))
			} else {
				sb.WriteString(fmt.Sprintf("Search results for %q:\n\n", queryStr))
			}
			for i, r := range results {
				sb.WriteString(fmt.Sprintf("%d. %s\n   URL: %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet))
			}
			return tool.ToolResult{Content: sb.String()}
		},
	}
}

func (h *chatHandler) fetchPageTool(r *http.Request) tool.ToolDef {
	return tool.ToolDef{
		Name:        "fetch_page",
		Description: "Fetch a web page and extract content relevant to a specific question. Use after web_search to read promising results in detail. Returns a focused summary of the page's relevant content.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"url", "question"},
			Properties: map[string]tool.PropertySchema{
				"url":      {Type: "string", Description: "The URL of the web page to fetch and read."},
				"question": {Type: "string", Description: "What you are looking for on this page. Guides extraction of relevant content."},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			urlStr, _ := args["url"].(string)
			questionStr, _ := args["question"].(string)

			if h.pageFetcher == nil {
				return tool.ToolResult{Content: "Page fetching is not available."}
			}
			if urlStr == "" {
				return tool.ToolResult{Content: "Error: url is required."}
			}
			if questionStr == "" {
				return tool.ToolResult{Content: "Error: question is required."}
			}

			result, err := h.pageFetcher.FetchAndExtract(ctx.Ctx, urlStr, questionStr)
			if err != nil {
				slog.Error("fetch_page failed", "error", err, "url", urlStr)
				return tool.ToolResult{Content: err.Error()}
			}

			return tool.ToolResult{Content: result}
		},
	}
}

func (h *chatHandler) checkWeatherTool(r *http.Request) tool.ToolDef {
	return tool.ToolDef{
		Name:        "check_weather",
		Description: "Check the current weather conditions for a given location. Use when the user asks about the weather, temperature, or conditions somewhere.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"location"},
			Properties: map[string]tool.PropertySchema{
				"location": {Type: "string", Description: "The city or location to check weather for, e.g. 'Melbourne', 'Tokyo', 'New York'."},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			locStr, _ := args["location"].(string)
			if h.weather == nil {
				return tool.ToolResult{Content: "Weather checking is not available."}
			}
			if locStr == "" {
				return tool.ToolResult{Content: "Error: location is required."}
			}

			result, err := h.weather.GetWeather(ctx.Ctx, locStr)
			if err != nil {
				slog.Error("weather lookup failed", "error", err, "location", locStr)
				return tool.ToolResult{Content: fmt.Sprintf("Weather lookup failed: %v", err)}
			}

			return tool.ToolResult{Content: fmt.Sprintf("Weather for %s:\nConditions: %s\nTemperature: %.1f°C (feels like %.1f°C)\nHumidity: %d%%\nWind: %.1f km/h\nTime of day: %s",
				result.Location, result.Description, result.Temperature, result.FeelsLike,
				result.Humidity, result.WindSpeed, dayOrNight(result.IsDay))}
		},
	}
}

func (h *chatHandler) useClaudeTool(sendEvent func(sseEvent), r *http.Request, agentSlug string, conversationID string) tool.ToolDef {
	return tool.ToolDef{
		Name:        "use_claude",
		Description: "Request a code change to yourself — your config, chat UI, or tools. Use when the user asks you to change something about how you work, look, or behave that requires modifying code. Always tell the user what change you're requesting before calling this tool. Examples: changing your greeting, adding a UI feature, modifying your system prompt assembly.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"description"},
			Properties: map[string]tool.PropertySchema{
				"description": {Type: "string", Description: "A clear, user-facing description of the code change to make. This will be shown to the user, so write it as a transparent explanation of what you're requesting and why. Be specific about what will change and the expected outcome."},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			descStr, _ := args["description"].(string)
			if h.taskRunner == nil {
				return tool.ToolResult{Content: "Code change capability is not available."}
			}
			if descStr == "" {
				return tool.ToolResult{Content: "Error: description is required."}
			}

			sub, err := h.taskRunner.SubmitTask(ctx.Ctx, NewClaudeSessionRequest(descStr, ctx.AgentSlug, ctx.Username))
			if err != nil {
				slog.Error("task submission failed", "error", err)
				return tool.ToolResult{Content: fmt.Sprintf("Failed to submit code change: %v", err)}
			}

			sendEvent(sseEvent{Task: &SSETaskEvent{
				TaskID:      sub.TaskID,
				Type:        "claude_session",
				Status:      "queued",
				Description: descStr,
			}})

			h.bgTasks.Add(1)
			go func(taskID string) {
				defer h.bgTasks.Done()
				h.pollTaskCompletion(taskID)
			}(sub.TaskID)

			return tool.ToolResult{Content: fmt.Sprintf("Code change submitted (task %s): %s\nIt will be implemented, tested, and deployed automatically. This usually takes a few minutes.", sub.TaskID, descStr)}
		},
	}
}

func (h *chatHandler) recallMemoryTool() tool.ToolDef {
	return tool.ToolDef{
		Name:        "recall_memory",
		Description: "Search your memory for past conversations, experiences, and knowledge about the user. Use when the user mentions something you don't recall, references a previous conversation, or when context from past interactions would help you respond better.",
		Parameters: tool.ParameterSchema{
			Type:     "object",
			Required: []string{"query"},
			Properties: map[string]tool.PropertySchema{
				"query": {Type: "string", Description: "What to search your memory for — a topic, event, feeling, or fact from past interactions."},
			},
		},
		Execute: func(ctx *tool.ToolContext, args map[string]any) tool.ToolResult {
			queryStr, _ := args["query"].(string)
			if h.memoryRecaller == nil {
				return tool.ToolResult{Content: "Memory recall is not available."}
			}
			if queryStr == "" {
				return tool.ToolResult{Content: "Error: query is required."}
			}

			resp, err := h.memoryRecaller.RecallMemories(ctx.Ctx, ctx.AgentSlug, ctx.UserID, queryStr, 5)
			if err != nil {
				slog.Error("memory recall failed", "error", err, "query", queryStr)
				return tool.ToolResult{Content: fmt.Sprintf("Memory recall failed: %v", err)}
			}

			return tool.ToolResult{Content: formatMemoryResponse(resp)}
		},
	}
}

func formatMemoryResponse(resp *MemoryRecallResponse) string {
	if len(resp.Memories) == 0 {
		return "No memories found for this query."
	}

	var sb strings.Builder
	sb.WriteString("## Recalled Memories\n\n")

	for i, m := range resp.Memories {
		sb.WriteString(fmt.Sprintf("%d. [%s] %s", i+1, m.Type, m.Content))
		if m.EmotionalContext != "" {
			sb.WriteString(fmt.Sprintf(" (emotions: %s)", m.EmotionalContext))
		}
		sb.WriteString(fmt.Sprintf(" [importance: %.0f%%]\n", m.Importance*100))
	}

	if rc := resp.RelationshipContext; rc != nil {
		sb.WriteString("\n## Relationship State\n")
		sb.WriteString(fmt.Sprintf("Trust: %.0f%% | Intimacy: %.0f%% | Autonomy: %.0f%%\n", rc.Trust*100, rc.Intimacy*100, rc.Autonomy*100))
		sb.WriteString(fmt.Sprintf("Reciprocity: %.0f%% | Playfulness: %.0f%% | Conflict: %.0f%%\n", rc.Reciprocity*100, rc.Playfulness*100, rc.Conflict*100))
		sb.WriteString(fmt.Sprintf("Total conversations: %d | Total memories: %d\n", rc.TotalConversations, rc.TotalMemories))
		if rc.PersonalityContext != "" {
			sb.WriteString(fmt.Sprintf("\n## Personality Context\n%s\n", rc.PersonalityContext))
		}
	}

	return sb.String()
}

// SkillInfo describes an available skill for the frontend.
type SkillInfo struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailableSkills is the single source of truth for all skills.
var AvailableSkills = []SkillInfo{
	{ID: "web_search", Label: "Web Search", Description: "search the web for current information"},
	{ID: "fetch_page", Label: "Fetch Page", Description: "read web pages and extract relevant content"},
	{ID: "use_claude", Label: "Use Claude", Description: "request code changes to itself"},
	{ID: "current_time", Label: "Current Time", Description: "tell the current date and time"},
	{ID: "check_weather", Label: "Check Weather", Description: "check current weather conditions for a location"},
	{ID: "recall_memory", Label: "Recall Memory", Description: "search past conversations and memories"},
}

// BuildToolsForAgent returns the Ollama tool definitions for an agent's enabled skills.
// Used by non-chat callers that need Ollama-formatted tool schemas.
func BuildToolsForAgent(a Agent) ollamaapi.Tools {
	defs := make(map[string]tool.ToolDef)
	for _, skill := range a.Skills {
		switch skill {
		case "current_time":
			defs[skill] = tool.CurrentTimeTool()
		case "web_search":
			defs[skill] = tool.ToolDef{
				Name:        "web_search",
				Description: "Search the web for current information.",
				Parameters: tool.ParameterSchema{
					Type:     "object",
					Required: []string{"query"},
					Properties: map[string]tool.PropertySchema{
						"query": {Type: "string", Description: "The search query."},
					},
				},
			}
		case "fetch_page":
			defs[skill] = tool.ToolDef{
				Name:        "fetch_page",
				Description: "Fetch a web page and extract relevant content.",
				Parameters: tool.ParameterSchema{
					Type:     "object",
					Required: []string{"url", "question"},
					Properties: map[string]tool.PropertySchema{
						"url":      {Type: "string", Description: "The URL to fetch."},
						"question": {Type: "string", Description: "What to look for."},
					},
				},
			}
		case "use_claude":
			defs[skill] = tool.ToolDef{
				Name:        "use_claude",
				Description: "Request a code change.",
				Parameters: tool.ParameterSchema{
					Type:     "object",
					Required: []string{"description"},
					Properties: map[string]tool.PropertySchema{
						"description": {Type: "string", Description: "Description of the change."},
					},
				},
			}
		case "check_weather":
			defs[skill] = tool.ToolDef{
				Name:        "check_weather",
				Description: "Check the current weather.",
				Parameters: tool.ParameterSchema{
					Type:     "object",
					Required: []string{"location"},
					Properties: map[string]tool.PropertySchema{
						"location": {Type: "string", Description: "The location."},
					},
				},
			}
		}
	}
	var toolDefs []tool.ToolDef
	for _, d := range defs {
		toolDefs = append(toolDefs, d)
	}
	return toOllamaToolDefs(toolDefs)
}
