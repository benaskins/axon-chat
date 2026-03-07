package chat

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// memoryStore implements Store backed by in-memory maps for testing.
type memoryStore struct {
	mu            sync.RWMutex
	users         map[string]bool
	agents        map[string]Agent
	conversations map[string]Conversation
	messages      map[string][]Message
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		users:         make(map[string]bool),
		agents:        make(map[string]Agent),
		conversations: make(map[string]Conversation),
		messages:      make(map[string][]Message),
	}
}

func memAgentKey(userID, slug string) string { return userID + "/" + slug }

func (s *memoryStore) CreateUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[id] = true
	return nil
}

func (s *memoryStore) UserExists(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id], nil
}

func (s *memoryStore) ListAgentsByUser(userID string) ([]AgentSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []AgentSummary
	for _, a := range s.agents {
		if a.UserID == userID {
			result = append(result, AgentSummary{
				Slug: a.Slug, Name: a.Name, Tagline: a.Tagline,
				AvatarEmoji: a.AvatarEmoji, DefaultModel: a.DefaultModel,
			})
		}
	}
	return result, nil
}

func (s *memoryStore) GetAgentByUser(userID, slug string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[memAgentKey(userID, slug)]
	if !ok {
		return nil, fmt.Errorf("agent not found")
	}
	cp := a
	return &cp, nil
}

func (s *memoryStore) GetAgentBySlug(slug string) (*Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.agents {
		if a.Slug == slug {
			cp := a
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("agent not found")
}

func (s *memoryStore) SaveAgent(agent Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[memAgentKey(agent.UserID, agent.Slug)] = agent
	return nil
}

func (s *memoryStore) DeleteAgent(userID, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.agents, memAgentKey(userID, slug))
	return nil
}

func (s *memoryStore) ListConversationsByUser(userID string, agentSlug string) ([]ConversationSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []ConversationSummary
	for _, c := range s.conversations {
		if c.UserID == userID && c.AgentSlug == agentSlug {
			result = append(result, ConversationSummary{
				ID: c.ID, Title: c.Title, CreatedAt: c.CreatedAt,
				UpdatedAt: c.UpdatedAt, MessageCount: len(s.messages[c.ID]),
			})
		}
	}
	if result == nil {
		result = []ConversationSummary{}
	}
	return result, nil
}

func (s *memoryStore) GetConversationByUser(userID string, id string) (*Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	return &c, nil
}

func (s *memoryStore) CreateConversationForUser(userID string, agentSlug string) (*Conversation, error) {
	return s.CreateConversationWithID(uuid.New().String(), userID, agentSlug)
}

func (s *memoryStore) CreateConversationWithID(id, userID, agentSlug string) (*Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	c := Conversation{
		ID: id, AgentSlug: agentSlug, UserID: userID,
		CreatedAt: now, UpdatedAt: now,
	}
	s.conversations[c.ID] = c
	return &c, nil
}

func (s *memoryStore) UpdateConversationTitle(userID string, id string, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return fmt.Errorf("conversation not found")
	}
	c.Title = &title
	c.UpdatedAt = time.Now()
	s.conversations[id] = c
	return nil
}

func (s *memoryStore) DeleteConversation(userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

func (s *memoryStore) GetMessages(conversationID string) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[conversationID]
	if msgs == nil {
		msgs = []Message{}
	}
	return msgs, nil
}

func (s *memoryStore) GetRecentMessages(conversationID string, limit int) ([]Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[conversationID]
	if msgs == nil {
		msgs = []Message{}
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (s *memoryStore) AppendMessage(conversationID string, msg Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	msg.CreatedAt = time.Now()
	s.messages[conversationID] = append(s.messages[conversationID], msg)
	if c, ok := s.conversations[conversationID]; ok {
		c.UpdatedAt = time.Now()
		s.conversations[conversationID] = c
	}
	return nil
}
