package chat

import (
	"context"
	"encoding/json"
	"testing"

	fact "github.com/benaskins/axon-fact"
)

func newTestEvent(stream, typ string, data any) fact.Event {
	raw, _ := json.Marshal(data)
	return fact.Event{
		ID:       "test-evt",
		Stream:   stream,
		Type:     typ,
		Data:     raw,
		Sequence: 1,
	}
}

// --- UserProjector ---

func TestUserProjector_Created(t *testing.T) {
	store := newMemoryStore()
	p := NewUserProjector(store)

	evt := newTestEvent("user-u1", "user.created", UserCreated{UserID: "u1"})
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	exists, _ := store.UserExists("u1")
	if !exists {
		t.Error("user should exist after user.created event")
	}
}

func TestUserProjector_IgnoresUnknownEvents(t *testing.T) {
	store := newMemoryStore()
	p := NewUserProjector(store)

	evt := newTestEvent("user-u1", "agent.created", AgentCreated{})
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("should ignore unknown events, got: %v", err)
	}
}

// --- AgentProjector ---

func TestAgentProjector_Created(t *testing.T) {
	store := newMemoryStore()
	p := NewAgentProjector(store)

	agent := Agent{Slug: "writer", UserID: "u1", Name: "Writer"}
	evt := newTestEvent("agent-u1-writer", "agent.created", AgentCreated{Agent: agent})
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got, err := store.GetAgentByUser("u1", "writer")
	if err != nil {
		t.Fatalf("GetAgentByUser: %v", err)
	}
	if got.Name != "Writer" {
		t.Errorf("Name = %q, want Writer", got.Name)
	}
}

func TestAgentProjector_Updated(t *testing.T) {
	store := newMemoryStore()
	p := NewAgentProjector(store)

	agent := Agent{Slug: "writer", UserID: "u1", Name: "Writer"}
	p.Handle(context.Background(), newTestEvent("agent-u1-writer", "agent.created", AgentCreated{Agent: agent}))

	agent.Name = "Senior Writer"
	p.Handle(context.Background(), newTestEvent("agent-u1-writer", "agent.updated", AgentUpdated{Agent: agent}))

	got, _ := store.GetAgentByUser("u1", "writer")
	if got.Name != "Senior Writer" {
		t.Errorf("Name = %q, want Senior Writer", got.Name)
	}
}

func TestAgentProjector_Deleted(t *testing.T) {
	store := newMemoryStore()
	p := NewAgentProjector(store)

	agent := Agent{Slug: "writer", UserID: "u1", Name: "Writer"}
	p.Handle(context.Background(), newTestEvent("agent-u1-writer", "agent.created", AgentCreated{Agent: agent}))

	// Delete event needs metadata to know which agent
	evt := newTestEvent("agent-u1-writer", "agent.deleted", AgentDeleted{})
	evt.Metadata = map[string]string{"user_id": "u1", "slug": "writer"}
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	_, err := store.GetAgentByUser("u1", "writer")
	if err == nil {
		t.Error("agent should be deleted")
	}
}

// --- ConversationProjector ---

func TestConversationProjector_Created(t *testing.T) {
	store := newMemoryStore()
	p := NewConversationProjector(store)

	evt := newTestEvent("conversation-c1", "conversation.created", ConversationCreated{
		ID:        "c1",
		AgentSlug: "writer",
		UserID:    "u1",
	})
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got, err := store.GetConversationByUser("u1", "c1")
	if err != nil {
		t.Fatalf("GetConversationByUser: %v", err)
	}
	if got.AgentSlug != "writer" {
		t.Errorf("AgentSlug = %q", got.AgentSlug)
	}
}

func TestConversationProjector_MessageAppended(t *testing.T) {
	store := newMemoryStore()
	p := NewConversationProjector(store)

	// Create conversation first
	p.Handle(context.Background(), newTestEvent("conversation-c1", "conversation.created", ConversationCreated{
		ID: "c1", AgentSlug: "writer", UserID: "u1",
	}))

	evt := newTestEvent("conversation-c1", "message.appended", MessageAppended{
		ID: "m1", Role: "user", Content: "hello",
	})
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	msgs, _ := store.GetMessages("c1")
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}
	if msgs[0].Content != "hello" || msgs[0].Role != "user" {
		t.Errorf("message = %+v", msgs[0])
	}
}

func TestConversationProjector_Titled(t *testing.T) {
	store := newMemoryStore()
	p := NewConversationProjector(store)

	p.Handle(context.Background(), newTestEvent("conversation-c1", "conversation.created", ConversationCreated{
		ID: "c1", AgentSlug: "writer", UserID: "u1",
	}))

	evt := newTestEvent("conversation-c1", "conversation.titled", ConversationTitled{Title: "Go Design"})
	evt.Metadata = map[string]string{"user_id": "u1"}
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got, _ := store.GetConversationByUser("u1", "c1")
	if got.Title == nil || *got.Title != "Go Design" {
		t.Errorf("Title = %v", got.Title)
	}
}

func TestConversationProjector_Deleted(t *testing.T) {
	store := newMemoryStore()
	p := NewConversationProjector(store)

	p.Handle(context.Background(), newTestEvent("conversation-c1", "conversation.created", ConversationCreated{
		ID: "c1", AgentSlug: "writer", UserID: "u1",
	}))

	evt := newTestEvent("conversation-c1", "conversation.deleted", ConversationDeleted{})
	evt.Metadata = map[string]string{"user_id": "u1"}
	if err := p.Handle(context.Background(), evt); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	_, err := store.GetConversationByUser("u1", "c1")
	if err == nil {
		t.Error("conversation should be deleted")
	}
}

// --- DefaultProjectors ---

func TestDefaultProjectors(t *testing.T) {
	store := newMemoryStore()
	projectors := DefaultProjectors(store, store)
	if len(projectors) != 3 {
		t.Errorf("got %d projectors, want 3", len(projectors))
	}
}
