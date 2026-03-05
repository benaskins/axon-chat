package chat

// Agent represents a full agent definition.
type Agent struct {
	Slug            string   `json:"slug"`
	UserID          string   `json:"user_id"`
	Name            string   `json:"name"`
	Tagline         string   `json:"tagline"`
	AvatarEmoji     string   `json:"avatar_emoji"`
	SystemPrompt    string   `json:"system_prompt"`
	Constraints     string   `json:"constraints,omitempty"`
	Greeting        string   `json:"greeting"`
	DefaultModel    string   `json:"default_model,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	Think           *bool    `json:"think,omitempty"`
	TopP            *float64 `json:"top_p,omitempty"`
	TopK            *int     `json:"top_k,omitempty"`
	MinP            *float64 `json:"min_p,omitempty"`
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`
	MaxTokens       *int     `json:"max_tokens,omitempty"`
	Tools           []string `json:"tools,omitempty"`
}

// AgentSummary is the lightweight representation for list responses.
type AgentSummary struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Tagline      string `json:"tagline"`
	AvatarEmoji  string `json:"avatar_emoji"`
	DefaultModel string `json:"default_model,omitempty"`
}

// AgentDetailResponse is the full agent response including the assembled full prompt.
type AgentDetailResponse struct {
	Agent
	FullPrompt string `json:"full_prompt"`
}
