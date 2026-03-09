package chat

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	fact "github.com/benaskins/axon-fact"
)

// EventTyper is implemented by all domain event structs.
type EventTyper interface {
	EventType() string
}

// NewEvent creates a fact.Event from a domain event struct.
func NewEvent(stream string, data EventTyper) (fact.Event, error) {
	return NewEventWithMeta(stream, data, nil)
}

// NewEventWithMeta creates a fact.Event with metadata from a domain event struct.
func NewEventWithMeta(stream string, data EventTyper, meta map[string]string) (fact.Event, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return fact.Event{}, err
	}
	return fact.Event{
		ID:       generateEventID(),
		Stream:   stream,
		Type:     data.EventType(),
		Data:     raw,
		Metadata: meta,
	}, nil
}

func generateEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// emitEvent appends a domain event to the event store. Returns nil if es is nil (no-op).
func emitEvent(ctx context.Context, es fact.EventStore, stream string, data EventTyper, meta map[string]string) error {
	if es == nil {
		return nil
	}
	evt, err := NewEventWithMeta(stream, data, meta)
	if err != nil {
		return err
	}
	return es.Append(ctx, stream, []fact.Event{evt})
}

// User events

type UserCreated struct {
	UserID string `json:"user_id"`
}

func (e UserCreated) EventType() string { return "user.created" }

// Agent events

type AgentCreated struct {
	Agent Agent `json:"agent"`
}

func (e AgentCreated) EventType() string { return "agent.created" }

type AgentUpdated struct {
	Agent Agent `json:"agent"`
}

func (e AgentUpdated) EventType() string { return "agent.updated" }

type AgentDeleted struct{}

func (e AgentDeleted) EventType() string { return "agent.deleted" }

// Conversation events

type ConversationCreated struct {
	ID        string `json:"id"`
	AgentSlug string `json:"agent_slug"`
	UserID    string `json:"user_id"`
}

func (e ConversationCreated) EventType() string { return "conversation.created" }

type MessageAppended struct {
	ID         string `json:"id"`
	Role       string `json:"role"`
	Content    string `json:"content"`
	Thinking   string `json:"thinking,omitempty"`
	ToolCalls  string `json:"tool_calls,omitempty"`
	Images     string `json:"images,omitempty"`
	DurationMs *int64 `json:"duration_ms,omitempty"`
}

func (e MessageAppended) EventType() string { return "message.appended" }

type ConversationTitled struct {
	Title string `json:"title"`
}

func (e ConversationTitled) EventType() string { return "conversation.titled" }

type ConversationDeleted struct{}

func (e ConversationDeleted) EventType() string { return "conversation.deleted" }
