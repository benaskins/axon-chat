package chat

import (
	"context"
	"encoding/json"
	"strings"

	fact "github.com/benaskins/axon-fact"
)

// DefaultProjectors returns the standard set of projectors for axon-chat.
// readStore provides query access, writer provides mutation access for projections.
func DefaultProjectors(readStore ReadStore, writer ReadModelWriter) []fact.Projector {
	return []fact.Projector{
		NewUserProjector(writer),
		NewAgentProjector(writer),
		NewConversationProjector(writer),
	}
}

// UserProjector projects user events into the read model.
type UserProjector struct {
	store ReadModelWriter
}

func NewUserProjector(store ReadModelWriter) *UserProjector {
	return &UserProjector{store: store}
}

func (p *UserProjector) Handle(ctx context.Context, e fact.Event) error {
	switch e.Type {
	case "user.created":
		var data UserCreated
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return err
		}
		return p.store.CreateUser(data.UserID)
	}
	return nil
}

// AgentProjector projects agent events into the read model.
type AgentProjector struct {
	store ReadModelWriter
}

func NewAgentProjector(store ReadModelWriter) *AgentProjector {
	return &AgentProjector{store: store}
}

func (p *AgentProjector) Handle(ctx context.Context, e fact.Event) error {
	switch e.Type {
	case "agent.created":
		var data AgentCreated
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return err
		}
		return p.store.SaveAgent(data.Agent)

	case "agent.updated":
		var data AgentUpdated
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return err
		}
		return p.store.SaveAgent(data.Agent)

	case "agent.deleted":
		userID := e.Metadata["user_id"]
		slug := e.Metadata["slug"]
		return p.store.DeleteAgent(userID, slug)
	}
	return nil
}

// ConversationProjector projects conversation and message events into the read model.
type ConversationProjector struct {
	store ReadModelWriter
}

func NewConversationProjector(store ReadModelWriter) *ConversationProjector {
	return &ConversationProjector{store: store}
}

func (p *ConversationProjector) Handle(ctx context.Context, e fact.Event) error {
	convID := streamToConversationID(e.Stream)

	switch e.Type {
	case "conversation.created":
		var data ConversationCreated
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return err
		}
		_, err := p.store.CreateConversationWithID(data.ID, data.UserID, data.AgentSlug)
		return err

	case "message.appended":
		var data MessageAppended
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return err
		}
		return p.store.AppendMessage(convID, Message{
			ID:         data.ID,
			Role:       data.Role,
			Content:    data.Content,
			Thinking:   data.Thinking,
			ToolCalls:  data.ToolCalls,
			Images:     data.Images,
			DurationMs: data.DurationMs,
		})

	case "conversation.titled":
		var data ConversationTitled
		if err := json.Unmarshal(e.Data, &data); err != nil {
			return err
		}
		userID := e.Metadata["user_id"]
		return p.store.UpdateConversationTitle(userID, convID, data.Title)

	case "conversation.deleted":
		userID := e.Metadata["user_id"]
		return p.store.DeleteConversation(userID, convID)
	}
	return nil
}

// streamToConversationID extracts the conversation ID from a stream name.
// "conversation-abc123" → "abc123"
func streamToConversationID(stream string) string {
	return strings.TrimPrefix(stream, "conversation-")
}
