package chat

import "time"

// Store abstracts all persistence operations.
type Store interface {
	// Users
	CreateUser(id string) error
	UserExists(id string) (bool, error)

	// Agents
	ListAgentsByUser(userID string) ([]AgentSummary, error)
	GetAgentByUser(userID, slug string) (*Agent, error)
	SaveAgent(agent Agent) error
	DeleteAgent(userID, slug string) error

	// Conversations
	ListConversationsByUser(userID string, agentSlug string) ([]ConversationSummary, error)
	GetConversationByUser(userID string, id string) (*Conversation, error)
	CreateConversationForUser(userID string, agentSlug string) (*Conversation, error)
	UpdateConversationTitle(userID string, id string, title string) error
	DeleteConversation(userID string, id string) error

	// Messages
	GetMessages(conversationID string) ([]Message, error)
	GetRecentMessages(conversationID string, limit int) ([]Message, error)
	AppendMessage(conversationID string, msg Message) error

	// Gallery Images
	SaveGalleryImage(img GalleryImage) error
	GetGalleryImage(id string) (*GalleryImage, error)
	ListGalleryImagesByUser(userID string, agentSlug string) ([]GalleryImage, error)
	GetBaseImageByUser(userID string, agentSlug string) (*GalleryImage, error)
	SetBaseImage(userID string, agentSlug string, imageID string) error
}

// Conversation represents a chat conversation with an agent.
type Conversation struct {
	ID        string    `json:"id"`
	AgentSlug string    `json:"agent_slug"`
	UserID    string    `json:"user_id"`
	Title     *string   `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ConversationSummary is the lightweight representation for list responses.
type ConversationSummary struct {
	ID           string    `json:"id"`
	Title        *string   `json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// Message represents a single message in a conversation.
type Message struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	Thinking       string    `json:"thinking,omitempty"`
	ToolCalls      string    `json:"tool_calls,omitempty"`
	Images         string    `json:"images,omitempty"`
	DurationMs     *int64    `json:"duration_ms,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// GalleryImage represents a generated image with metadata.
type GalleryImage struct {
	ID             string    `json:"id"`
	AgentSlug      string    `json:"agent_slug"`
	UserID         string    `json:"user_id"`
	ConversationID *string   `json:"conversation_id"`
	Prompt         string    `json:"prompt"`
	Model          string    `json:"model"`
	IsBase         bool      `json:"is_base"`
	NSFWDetected   bool      `json:"nsfw_detected"`
	CreatedAt      time.Time `json:"created_at"`
}
