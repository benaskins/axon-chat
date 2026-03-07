package chat

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

func (s *readStoreCheck) UserExists(id string) (bool, error)                       { return false, nil }
func (s *readStoreCheck) ListAgentsByUser(userID string) ([]AgentSummary, error)    { return nil, nil }
func (s *readStoreCheck) GetAgentByUser(userID, slug string) (*Agent, error)        { return nil, nil }
func (s *readStoreCheck) GetAgentBySlug(slug string) (*Agent, error)                { return nil, nil }
func (s *readStoreCheck) ListConversationsByUser(userID string, agentSlug string) ([]ConversationSummary, error) {
	return nil, nil
}
func (s *readStoreCheck) GetConversationByUser(userID string, id string) (*Conversation, error) {
	return nil, nil
}
func (s *readStoreCheck) GetMessages(conversationID string) ([]Message, error)             { return nil, nil }
func (s *readStoreCheck) GetRecentMessages(conversationID string, limit int) ([]Message, error) {
	return nil, nil
}

// Minimal compile-time check: ReadModelWriter has write methods for projectors
type writeStoreCheck struct{}

func (s *writeStoreCheck) CreateUser(id string) error                                          { return nil }
func (s *writeStoreCheck) SaveAgent(agent Agent) error                                         { return nil }
func (s *writeStoreCheck) DeleteAgent(userID, slug string) error                               { return nil }
func (s *writeStoreCheck) CreateConversationForUser(userID string, agentSlug string) (*Conversation, error) {
	return nil, nil
}
func (s *writeStoreCheck) UpdateConversationTitle(userID string, id string, title string) error { return nil }
func (s *writeStoreCheck) DeleteConversation(userID string, id string) error                    { return nil }
func (s *writeStoreCheck) AppendMessage(conversationID string, msg Message) error               { return nil }
