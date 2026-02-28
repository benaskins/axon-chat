// Package chattest provides in-memory mock implementations of chat.Store
// for testing without a database.
package chattest

import (
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
	conversations map[string]chat.Conversation  // key: id
	messages      map[string][]chat.Message      // key: conversationID
	galleryImages map[string]chat.GalleryImage   // key: id
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:         make(map[string]bool),
		agents:        make(map[string]chat.Agent),
		conversations: make(map[string]chat.Conversation),
		messages:      make(map[string][]chat.Message),
		galleryImages: make(map[string]chat.GalleryImage),
	}
}

func agentKey(userID, slug string) string {
	return userID + "/" + slug
}

// --- User operations ---

func (s *MemoryStore) CreateUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users[id] = true
	return nil
}

func (s *MemoryStore) UserExists(id string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.users[id], nil
}

// --- Agent operations ---

func (s *MemoryStore) ListAgentsByUser(userID string) ([]chat.AgentSummary, error) {
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

func (s *MemoryStore) GetAgentByUser(userID, slug string) (*chat.Agent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.agents[agentKey(userID, slug)]
	if !ok {
		return nil, fmt.Errorf("agent not found")
	}
	cp := a
	return &cp, nil
}

func (s *MemoryStore) SaveAgent(agent chat.Agent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.agents[agentKey(agent.UserID, agent.Slug)] = agent
	return nil
}

func (s *MemoryStore) DeleteAgent(userID, slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.agents, agentKey(userID, slug))
	return nil
}

// --- Conversation operations ---

func (s *MemoryStore) ListConversationsByUser(userID string, agentSlug string) ([]chat.ConversationSummary, error) {
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

func (s *MemoryStore) GetConversationByUser(userID string, id string) (*chat.Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return nil, fmt.Errorf("conversation not found: %s", id)
	}
	return &c, nil
}

func (s *MemoryStore) CreateConversationForUser(userID string, agentSlug string) (*chat.Conversation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	c := chat.Conversation{
		ID:        uuid.New().String(),
		AgentSlug: agentSlug,
		UserID:    userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.conversations[c.ID] = c
	return &c, nil
}

func (s *MemoryStore) UpdateConversationTitle(userID string, id string, title string) error {
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

func (s *MemoryStore) DeleteConversation(userID string, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.conversations[id]
	if !ok || c.UserID != userID {
		return fmt.Errorf("conversation not found")
	}
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

// --- Message operations ---

func (s *MemoryStore) GetMessages(conversationID string) ([]chat.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	msgs := s.messages[conversationID]
	if msgs == nil {
		msgs = []chat.Message{}
	}
	return msgs, nil
}

func (s *MemoryStore) GetRecentMessages(conversationID string, limit int) ([]chat.Message, error) {
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

func (s *MemoryStore) AppendMessage(conversationID string, msg chat.Message) error {
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

// --- Gallery Image operations ---

func (s *MemoryStore) SaveGalleryImage(img chat.GalleryImage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.galleryImages[img.ID] = img
	return nil
}

func (s *MemoryStore) GetGalleryImage(id string) (*chat.GalleryImage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	img, ok := s.galleryImages[id]
	if !ok {
		return nil, fmt.Errorf("gallery image not found")
	}
	return &img, nil
}

func (s *MemoryStore) ListGalleryImagesByUser(userID string, agentSlug string) ([]chat.GalleryImage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []chat.GalleryImage
	for _, img := range s.galleryImages {
		if img.UserID == userID && img.AgentSlug == agentSlug {
			result = append(result, img)
		}
	}
	return result, nil
}

func (s *MemoryStore) GetBaseImageByUser(userID string, agentSlug string) (*chat.GalleryImage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, img := range s.galleryImages {
		if img.UserID == userID && img.AgentSlug == agentSlug && img.IsBase {
			return &img, nil
		}
	}
	return nil, nil
}

func (s *MemoryStore) SetBaseImage(userID string, agentSlug string, imageID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Unset current base
	for id, img := range s.galleryImages {
		if img.UserID == userID && img.AgentSlug == agentSlug && img.IsBase {
			img.IsBase = false
			s.galleryImages[id] = img
		}
	}
	// Set new base
	if img, ok := s.galleryImages[imageID]; ok {
		img.IsBase = true
		s.galleryImages[imageID] = img
	}
	return nil
}
