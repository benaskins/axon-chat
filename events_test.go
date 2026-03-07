package chat

import (
	"encoding/json"
	"testing"
)

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		event    EventTyper
		wantType string
	}{
		{"UserCreated", UserCreated{UserID: "u1"}, "user.created"},
		{"AgentCreated", AgentCreated{Agent: Agent{Slug: "writer"}}, "agent.created"},
		{"AgentUpdated", AgentUpdated{Agent: Agent{Slug: "writer"}}, "agent.updated"},
		{"AgentDeleted", AgentDeleted{}, "agent.deleted"},
		{"ConversationCreated", ConversationCreated{AgentSlug: "writer", UserID: "u1"}, "conversation.created"},
		{"MessageAppended", MessageAppended{ID: "m1", Role: "user", Content: "hello"}, "message.appended"},
		{"ConversationTitled", ConversationTitled{Title: "Chat about Go"}, "conversation.titled"},
		{"ConversationDeleted", ConversationDeleted{}, "conversation.deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.event.EventType(); got != tt.wantType {
				t.Errorf("EventType() = %q, want %q", got, tt.wantType)
			}
		})
	}
}

func TestEventMarshalRoundTrip(t *testing.T) {
	original := MessageAppended{
		ID:      "m1",
		Role:    "assistant",
		Content: "hello world",
		Thinking: "let me think",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded MessageAppended
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if decoded.ID != original.ID || decoded.Role != original.Role ||
		decoded.Content != original.Content || decoded.Thinking != original.Thinking {
		t.Errorf("round trip mismatch: got %+v", decoded)
	}
}

func TestAgentCreatedPreservesFullConfig(t *testing.T) {
	temp := 0.7
	agent := Agent{
		Slug:         "writer",
		UserID:       "u1",
		Name:         "Writer",
		SystemPrompt: "You are a writer",
		Temperature:  &temp,
		Tools:        []string{"web_search"},
	}

	event := AgentCreated{Agent: agent}
	data, _ := json.Marshal(event)

	var decoded AgentCreated
	json.Unmarshal(data, &decoded)

	if decoded.Agent.Slug != "writer" || decoded.Agent.Name != "Writer" {
		t.Errorf("agent fields lost: %+v", decoded.Agent)
	}
	if decoded.Agent.Temperature == nil || *decoded.Agent.Temperature != 0.7 {
		t.Error("temperature lost")
	}
	if len(decoded.Agent.Tools) != 1 || decoded.Agent.Tools[0] != "web_search" {
		t.Errorf("tools lost: %v", decoded.Agent.Tools)
	}
}

func TestNewEvent(t *testing.T) {
	evt, err := NewEvent("conversation-abc", ConversationCreated{
		AgentSlug: "writer",
		UserID:    "u1",
	})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}

	if evt.Stream != "conversation-abc" {
		t.Errorf("Stream = %q", evt.Stream)
	}
	if evt.Type != "conversation.created" {
		t.Errorf("Type = %q", evt.Type)
	}
	if evt.ID == "" {
		t.Error("ID should be set")
	}
	if len(evt.Data) == 0 {
		t.Error("Data should be set")
	}
}
