package chat

import (
	"context"
	"time"
)

// Store combines ReadStore and ReadModelWriter. Composition roots provide a
// concrete implementation that satisfies both interfaces.
type Store interface {
	ReadStore
	ReadModelWriter
}

// ReadStore provides read-only access to the data model.
// Handlers use this to query current state.
type ReadStore interface {
	// Users
	UserExists(ctx context.Context, id string) (bool, error)

	// Agents
	ListAgentsByUser(ctx context.Context, userID string) ([]AgentSummary, error)
	GetAgentByUser(ctx context.Context, userID, slug string) (*Agent, error)
	GetAgentBySlug(ctx context.Context, slug string) (*Agent, error)

	// Conversations
	ListConversationsByUser(ctx context.Context, userID string, agentSlug string) ([]ConversationSummary, error)
	GetConversationByUser(ctx context.Context, userID string, id string) (*Conversation, error)

	// Messages
	GetMessages(ctx context.Context, conversationID string) ([]Message, error)
	GetRecentMessages(ctx context.Context, conversationID string, limit int) ([]Message, error)
}

// ReadModelWriter provides write operations for projectors to build read models.
// Not used by handlers directly — handlers emit events, projectors call these.
type ReadModelWriter interface {
	CreateUser(ctx context.Context, id string) error
	SaveAgent(ctx context.Context, agent Agent) error
	DeleteAgent(ctx context.Context, userID, slug string) error
	CreateConversationForUser(ctx context.Context, userID string, agentSlug string) (*Conversation, error)
	CreateConversationWithID(ctx context.Context, id, userID, agentSlug string) (*Conversation, error)
	UpdateConversationTitle(ctx context.Context, userID string, id string, title string) error
	DeleteConversation(ctx context.Context, userID string, id string) error
	AppendMessage(ctx context.Context, conversationID string, msg Message) error
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
