package chat

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	ollamaapi "github.com/ollama/ollama/api"
)

// ToolResult is returned by a tool's Execute function.
type ToolResult struct {
	Content string
}

// ToolContext carries handler dependencies and request-scoped state into tool execution.
type ToolContext struct {
	Handler        *chatHandler
	Request        *http.Request
	AgentSlug      string
	ConversationID string
	UserID         string
	Username       string
	SystemPrompt   string
	SendEvent      func(sseEvent)
	ImageRefs      *[]map[string]string
}

// ToolDef bundles a tool's Ollama definition with its execution logic.
type ToolDef struct {
	Name    string
	Build   func() ollamaapi.Tool
	Execute func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult
}

// registerTools populates the tool registry on the chat handler.
func (h *chatHandler) registerTools() {
	h.tools = map[string]*ToolDef{
		"take_photo":         takePhotoDef(h),
		"REMOVED_TOOL": takePrivatePhotoDef(h),
		"web_search":         webSearchDef(),
		"fetch_page":         fetchPageDef(),
		"use_claude":         useClaudeDef(),
		"current_time":       currentTimeDef(),
		"check_weather":      checkWeatherDef(),
	}
}

// buildToolForSkill returns the Ollama tool definition for a skill name, or nil if unknown.
func (h *chatHandler) buildToolForSkill(skill string) *ollamaapi.Tool {
	if def, ok := h.tools[skill]; ok {
		t := def.Build()
		return &t
	}
	return nil
}

// buildToolsForSkills returns Ollama tool definitions for a list of skill names.
func (h *chatHandler) buildToolsForSkills(skills []string) ollamaapi.Tools {
	var tools ollamaapi.Tools
	for _, skill := range skills {
		if t := h.buildToolForSkill(skill); t != nil {
			tools = append(tools, *t)
		}
	}
	return tools
}

// --- Tool definitions ---

func takePhotoDef(h *chatHandler) *ToolDef {
	return &ToolDef{
		Name: "take_photo",
		Build: func() ollamaapi.Tool {
			props := ollamaapi.NewToolPropertiesMap()
			props.Set("prompt", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "A detailed description of the image to generate. Include subject, setting, lighting, mood, and composition details.",
			})
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "take_photo",
					Description: "Take a photo or selfie. Only use when the user asks, you're in a new setting, or a significant visual moment happens. Most conversation turns should NOT include a photo.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{"prompt"},
						Properties: props,
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			prompt, _ := args.Get("prompt")
			promptStr, _ := prompt.(string)

			if promptStr == "" || h.taskRunner == nil {
				return ToolResult{Content: "Error: image generation not available"}
			}

			return h.submitImageTask(ctx, promptStr, false)
		},
	}
}

func takePrivatePhotoDef(h *chatHandler) *ToolDef {
	return &ToolDef{
		Name: "REMOVED_TOOL",
		Build: func() ollamaapi.Tool {
			props := ollamaapi.NewToolPropertiesMap()
			props.Set("prompt", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "A detailed description of the image to generate. Include subject, setting, lighting, mood, and composition details. This pipeline has no content restrictions.",
			})
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "REMOVED_TOOL",
					Description: "Take a photo using the private unrestricted pipeline. Use instead of take_photo when the scene requires content that the standard pipeline would filter out. Only use when the user specifically requests this kind of content.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{"prompt"},
						Properties: props,
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			prompt, _ := args.Get("prompt")
			promptStr, _ := prompt.(string)

			if promptStr == "" || h.taskRunner == nil {
				return ToolResult{Content: "Error: private image generation not available"}
			}

			return h.submitImageTask(ctx, promptStr, true)
		},
	}
}

func webSearchDef() *ToolDef {
	return &ToolDef{
		Name: "web_search",
		Build: func() ollamaapi.Tool {
			props := ollamaapi.NewToolPropertiesMap()
			props.Set("query", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "The search query to look up on the web.",
			})
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "web_search",
					Description: "Search the web for current information, news, facts, or anything not in your training data. Use when the user asks about recent events, specific facts you're unsure about, or when they explicitly ask you to look something up.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{"query"},
						Properties: props,
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			query, _ := args.Get("query")
			queryStr, _ := query.(string)

			h := ctx.Handler
			if h.searcher == nil || queryStr == "" {
				return ToolResult{Content: "Search is not available."}
			}

			// Qualify the search query using agent context
			searchQuery := queryStr
			if h.searchQualifier != nil && ctx.SystemPrompt != "" {
				searchQuery = h.searchQualifier.Qualify(ctx.Request.Context(), queryStr, ctx.SystemPrompt)
			}

			results, err := h.searcher.Search(ctx.Request.Context(), searchQuery, 5)
			if err != nil {
				slog.Error("web search failed", "error", err, "query", searchQuery)
				return ToolResult{Content: fmt.Sprintf("Search failed: %v", err)}
			}
			if len(results) == 0 {
				return ToolResult{Content: fmt.Sprintf("No results found for %q.", queryStr)}
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
			return ToolResult{Content: sb.String()}
		},
	}
}

func fetchPageDef() *ToolDef {
	return &ToolDef{
		Name: "fetch_page",
		Build: func() ollamaapi.Tool {
			props := ollamaapi.NewToolPropertiesMap()
			props.Set("url", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "The URL of the web page to fetch and read.",
			})
			props.Set("question", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "What you are looking for on this page. Guides extraction of relevant content.",
			})
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "fetch_page",
					Description: "Fetch a web page and extract content relevant to a specific question. Use after web_search to read promising results in detail. Returns a focused summary of the page's relevant content.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{"url", "question"},
						Properties: props,
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			rawURL, _ := args.Get("url")
			urlStr, _ := rawURL.(string)
			question, _ := args.Get("question")
			questionStr, _ := question.(string)

			h := ctx.Handler
			if h.pageFetcher == nil {
				return ToolResult{Content: "Page fetching is not available."}
			}
			if urlStr == "" {
				return ToolResult{Content: "Error: url is required."}
			}
			if questionStr == "" {
				return ToolResult{Content: "Error: question is required."}
			}

			result, err := h.pageFetcher.FetchAndExtract(ctx.Request.Context(), urlStr, questionStr)
			if err != nil {
				slog.Error("fetch_page failed", "error", err, "url", urlStr)
				return ToolResult{Content: err.Error()}
			}

			return ToolResult{Content: result}
		},
	}
}

func useClaudeDef() *ToolDef {
	return &ToolDef{
		Name: "use_claude",
		Build: func() ollamaapi.Tool {
			props := ollamaapi.NewToolPropertiesMap()
			props.Set("description", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "A clear, user-facing description of the code change to make. This will be shown to the user, so write it as a transparent explanation of what you're requesting and why. Be specific about what will change and the expected outcome.",
			})
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "use_claude",
					Description: "Request a code change to yourself — your config, chat UI, or tools. Use when the user asks you to change something about how you work, look, or behave that requires modifying code. Always tell the user what change you're requesting before calling this tool. Examples: changing your greeting, adding a UI feature, modifying your system prompt assembly.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{"description"},
						Properties: props,
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			desc, _ := args.Get("description")
			descStr, _ := desc.(string)

			h := ctx.Handler
			if h.taskRunner == nil {
				return ToolResult{Content: "Code change capability is not available."}
			}
			if descStr == "" {
				return ToolResult{Content: "Error: description is required."}
			}

			sub, err := h.taskRunner.SubmitTask(ctx.Request.Context(), NewClaudeSessionRequest(descStr, ctx.AgentSlug, ctx.Username))
			if err != nil {
				slog.Error("task submission failed", "error", err)
				return ToolResult{Content: fmt.Sprintf("Failed to submit code change: %v", err)}
			}

			ctx.SendEvent(sseEvent{Task: &SSETaskEvent{
				TaskID:      sub.TaskID,
				Type:        "claude_session",
				Status:      "queued",
				Description: descStr,
			}})

			// Poll for completion in background
			h.bgTasks.Add(1)
			go func(taskID string) {
				defer h.bgTasks.Done()
				h.pollTaskCompletion(taskID)
			}(sub.TaskID)

			return ToolResult{Content: fmt.Sprintf("Code change submitted (task %s): %s\nIt will be implemented, tested, and deployed automatically. This usually takes a few minutes.", sub.TaskID, descStr)}
		},
	}
}

func currentTimeDef() *ToolDef {
	return &ToolDef{
		Name: "current_time",
		Build: func() ollamaapi.Tool {
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "current_time",
					Description: "Get the current date and time. Use when the user asks what time it is, the current date, or anything requiring knowledge of the current moment.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{},
						Properties: ollamaapi.NewToolPropertiesMap(),
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			now := time.Now()
			return ToolResult{Content: fmt.Sprintf("Current time: %s", now.Format("Monday, 2 January 2006 3:04 PM MST"))}
		},
	}
}

func checkWeatherDef() *ToolDef {
	return &ToolDef{
		Name: "check_weather",
		Build: func() ollamaapi.Tool {
			props := ollamaapi.NewToolPropertiesMap()
			props.Set("location", ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{"string"},
				Description: "The city or location to check weather for, e.g. 'Melbourne', 'Tokyo', 'New York'.",
			})
			return ollamaapi.Tool{
				Type: "function",
				Function: ollamaapi.ToolFunction{
					Name:        "check_weather",
					Description: "Check the current weather conditions for a given location. Use when the user asks about the weather, temperature, or conditions somewhere.",
					Parameters: ollamaapi.ToolFunctionParameters{
						Type:       "object",
						Required:   []string{"location"},
						Properties: props,
					},
				},
			}
		},
		Execute: func(ctx *ToolContext, args ollamaapi.ToolCallFunctionArguments) ToolResult {
			loc, _ := args.Get("location")
			locStr, _ := loc.(string)

			h := ctx.Handler
			if h.weather == nil {
				return ToolResult{Content: "Weather checking is not available."}
			}
			if locStr == "" {
				return ToolResult{Content: "Error: location is required."}
			}

			result, err := h.weather.GetWeather(ctx.Request.Context(), locStr)
			if err != nil {
				slog.Error("weather lookup failed", "error", err, "location", locStr)
				return ToolResult{Content: fmt.Sprintf("Weather lookup failed: %v", err)}
			}

			return ToolResult{Content: fmt.Sprintf("Weather for %s:\nConditions: %s\nTemperature: %.1f°C (feels like %.1f°C)\nHumidity: %d%%\nWind: %.1f km/h\nTime of day: %s",
				result.Location, result.Description, result.Temperature, result.FeelsLike,
				result.Humidity, result.WindSpeed, dayOrNight(result.IsDay))}
		},
	}
}

// SkillInfo describes an available skill for the frontend.
type SkillInfo struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// AvailableSkills is the single source of truth for all skills.
var AvailableSkills = []SkillInfo{
	{ID: "take_photo", Label: "Take Photo", Description: "generate images during conversation"},
	{ID: "REMOVED_TOOL", Label: "Private Photo", Description: "generate unrestricted images during conversation"},
	{ID: "web_search", Label: "Web Search", Description: "search the web for current information"},
	{ID: "fetch_page", Label: "Fetch Page", Description: "read web pages and extract relevant content"},
	{ID: "use_claude", Label: "Use Claude", Description: "request code changes to itself"},
	{ID: "current_time", Label: "Current Time", Description: "tell the current date and time"},
	{ID: "check_weather", Label: "Check Weather", Description: "check current weather conditions for a location"},
}

// BuildToolsForAgent returns the Ollama tool definitions for an agent's enabled skills.
// Used by tests — production code uses chatHandler.buildToolsForSkills.
func BuildToolsForAgent(agent Agent) ollamaapi.Tools {
	builders := map[string]func() ollamaapi.Tool{
		"take_photo":         takePhotoDef(nil).Build,
		"REMOVED_TOOL": takePrivatePhotoDef(nil).Build,
		"web_search":         webSearchDef().Build,
		"fetch_page":         fetchPageDef().Build,
		"use_claude":         useClaudeDef().Build,
		"current_time":       currentTimeDef().Build,
		"check_weather":      checkWeatherDef().Build,
	}
	var tools ollamaapi.Tools
	for _, skill := range agent.Skills {
		if build, ok := builders[skill]; ok {
			tools = append(tools, build())
		}
	}
	return tools
}
