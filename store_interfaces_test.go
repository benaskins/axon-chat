package chat

import "context"

// Verify that the ReadStore and ReadModelWriter interfaces
// are defined and can be composed into a full store.
func _assertInterfaces() {
	// A type implementing both should satisfy Store (for backward compat)
	type fullStore interface {
		ReadStore
		ReadModelWriter
	}

	// ReadStore should have only read methods
	var _ ReadStore = (*readStoreCheck)(nil)

	// ReadModelWriter should have only write methods for projectors
	var _ ReadModelWriter = (*writeStoreCheck)(nil)
}

// Minimal compile-time check: ReadStore has read-only methods
type readStoreCheck struct{}

func (s *readStoreCheck) UserExists(_ context.Context, id string) (bool, error)                       { return false, nil }
func (s *readStoreCheck) ListAgentsByUser(_ context.Context, userID string) ([]AgentSummary, error)    { return nil, nil }
func (s *readStoreCheck) GetAgentByUser(_ context.Context, userID, slug string) (*Agent, error)        { return nil, nil }
func (s *readStoreCheck) GetAgentBySlug(_ context.Context, slug string) (*Agent, error)                { return nil, nil }
func (s *readStoreCheck) ListConversationsByUser(_ context.Context, userID string, agentSlug string) ([]ConversationSummary, error) {
	return nil, nil
}
func (s *readStoreCheck) GetConversationByUser(_ context.Context, userID string, id string) (*Conversation, error) {
	return nil, nil
}
func (s *readStoreCheck) GetMessages(_ context.Context, conversationID string) ([]Message, error)             { return nil, nil }
func (s *readStoreCheck) GetRecentMessages(_ context.Context, conversationID string, limit int) ([]Message, error) {
	return nil, nil
}

// Minimal compile-time check: ReadModelWriter has write methods for projectors
type writeStoreCheck struct{}

func (s *writeStoreCheck) CreateUser(_ context.Context, id string) error                                          { return nil }
func (s *writeStoreCheck) SaveAgent(_ context.Context, agent Agent) error                                         { return nil }
func (s *writeStoreCheck) DeleteAgent(_ context.Context, userID, slug string) error                               { return nil }
func (s *writeStoreCheck) CreateConversationForUser(_ context.Context, userID string, agentSlug string) (*Conversation, error) {
	return nil, nil
}
func (s *writeStoreCheck) CreateConversationWithID(_ context.Context, id, userID, agentSlug string) (*Conversation, error) {
	return nil, nil
}
func (s *writeStoreCheck) UpdateConversationTitle(_ context.Context, userID string, id string, title string) error { return nil }
func (s *writeStoreCheck) DeleteConversation(_ context.Context, userID string, id string) error                    { return nil }
func (s *writeStoreCheck) AppendMessage(_ context.Context, conversationID string, msg Message) error               { return nil }
