package chat

import "time"

// Store abstracts all persistence operations (read + write).
// Kept for backward compatibility — new code should use ReadStore and ReadModelWriter.
type Store interface {
	ReadStore
	ReadModelWriter
}

// ReadStore provides read-only access to the data model.
// Handlers use this to query current state.
type ReadStore interface {
	// Users
	UserExists(id string) (bool, error)

	// Agents
	ListAgentsByUser(userID string) ([]AgentSummary, error)
	GetAgentByUser(userID, slug string) (*Agent, error)
	GetAgentBySlug(slug string) (*Agent, error)

	// Conversations
	ListConversationsByUser(userID string, agentSlug string) ([]ConversationSummary, error)
	GetConversationByUser(userID string, id string) (*Conversation, error)

	// Messages
	GetMessages(conversationID string) ([]Message, error)
	GetRecentMessages(conversationID string, limit int) ([]Message, error)
}

// ReadModelWriter provides write operations for projectors to build read models.
// Not used by handlers directly — handlers emit events, projectors call these.
type ReadModelWriter interface {
	CreateUser(id string) error
	SaveAgent(agent Agent) error
	DeleteAgent(userID, slug string) error
	CreateConversationForUser(userID string, agentSlug string) (*Conversation, error)
	CreateConversationWithID(id, userID, agentSlug string) (*Conversation, error)
	UpdateConversationTitle(userID string, id string, title string) error
	DeleteConversation(userID string, id string) error
	AppendMessage(conversationID string, msg Message) error
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
