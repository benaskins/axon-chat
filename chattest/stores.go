// Package chattest provides in-memory mock implementations of chat.Store
// for testing without a database.
package chattest

import (
	"context"
	"fmt"
	"sync"
	"time"

	chat "github.com/benaskins/axon-chat"
	"github.com/google/uuid"
)

// MemoryStore implements chat.Store backed by in-memory maps.
type MemoryStore struct {
	mu            sync.RWMutex
	users         map[string]bool
	agents        map[string]chat.Agent        // key: userID+"/"+slug
	conversations map[string]chat.Conversation // key: id
	messages      map[string][]chat.Message    // key: conversationID
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:         make(map[string]bool),
		agents:        make(map[string]chat.Agent),
		conversations: make(map[string]chat.Conversation),
		messages:      make(map[string][]chat.Message),
	}
}

func agentKey(userID, slug string) string {
	return userID + "/" + slug
}

// --- User operations ---

func (s *MemoryStore) CreateUser(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[id] = true
	return nil
}

func (s *MemoryStore) UserExists(_ context.Context, id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id], nil
}

// --- Agent operations ---

func (s *MemoryStore) ListAgentsByUser(_ context.Context, userID string) ([]chat.AgentSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []chat.AgentSummary
	for _, a := range s.agents {
		if a.UserID == userID {
			result = append(result, chat.AgentSummary{
				Slug:         a.Slug,
				Name:         a.Name,
				Tagline:      a.Tagline,
				AvatarEmoji:  a.AvatarEmoji,
				DefaultModel: a.DefaultModel,
			})
		}
	}
	return result, nil
}

func (s *MemoryStore) GetAgentByUser(_ context.Context, userID, slug string) (*chat.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[agentKey(userID, slug)]
	if !ok {
		return nil, fmt.Errorf("%s: %w", slug, chat.ErrAgentNotFound)
	}
	cp := a
	return &cp, nil
}

func (s *MemoryStore) GetAgentBySlug(_ context.Context, slug string) (*chat.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, a := range s.agents {
		if a.Slug == slug {
			cp := a
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("%s: %w", slug, chat.ErrAgentNotFound)
}

func (s *MemoryStore) SaveAgent(_ context.Context, agent chat.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[agentKey(agent.UserID, agent.Slug)] = agent
	return nil
}

func (s *MemoryStore) DeleteAgent(_ context.Context, userID, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.agents, agentKey(userID, slug))
	return nil
}

// --- Conversation operations ---

func (s *MemoryStore) ListConversationsByUser(_ context.Context, userID string, agentSlug string) ([]chat.ConversationSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []chat.ConversationSummary
	for _, c := range s.conversations {
		if c.UserID == userID && c.AgentSlug == agentSlug {
			msgs := s.messages[c.ID]
			result = append(result, chat.ConversationSummary{
				ID:           c.ID,
				Title:        c.Title,
				CreatedAt:    c.CreatedAt,
				UpdatedAt:    c.UpdatedAt,
				MessageCount: len(msgs),
			})
		}
	}
	if result == nil {
		result = []chat.ConversationSummary{}
	}
	return result, nil
}

func (s *MemoryStore) GetConversationByUser(_ context.Context, userID string, id string) (*chat.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return nil, fmt.Errorf("%s: %w", id, chat.ErrConversationNotFound)
	}
	return &c, nil
}

func (s *MemoryStore) CreateConversationForUser(ctx context.Context, userID string, agentSlug string) (*chat.Conversation, error) {
	return s.CreateConversationWithID(ctx, uuid.New().String(), userID, agentSlug)
}

func (s *MemoryStore) CreateConversationWithID(_ context.Context, id, userID, agentSlug string) (*chat.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	c := chat.Conversation{
		ID:        id,
		AgentSlug: agentSlug,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.conversations[c.ID] = c
	return &c, nil
}

func (s *MemoryStore) UpdateConversationTitle(_ context.Context, userID string, id string, title string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return fmt.Errorf("%s: %w", id, chat.ErrConversationNotFound)
	}
	c.Title = &title
	c.UpdatedAt = time.Now()
	s.conversations[id] = c
	return nil
}

func (s *MemoryStore) DeleteConversation(_ context.Context, userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return fmt.Errorf("%s: %w", id, chat.ErrConversationNotFound)
	}
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

// --- Message operations ---

func (s *MemoryStore) GetMessages(_ context.Context, conversationID string) ([]chat.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[conversationID]
	if msgs == nil {
		msgs = []chat.Message{}
	}
	return msgs, nil
}

func (s *MemoryStore) GetRecentMessages(_ context.Context, conversationID string, limit int) ([]chat.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[conversationID]
	if msgs == nil {
		msgs = []chat.Message{}
	}
	if len(msgs) > limit {
		msgs = msgs[len(msgs)-limit:]
	}
	return msgs, nil
}

func (s *MemoryStore) AppendMessage(_ context.Context, conversationID string, msg chat.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	msg.CreatedAt = time.Now()
	s.messages[conversationID] = append(s.messages[conversationID], msg)
	// Touch conversation updated_at
	if c, ok := s.conversations[conversationID]; ok {
		c.UpdatedAt = time.Now()
		s.conversations[conversationID] = c
	}
	return nil
}
