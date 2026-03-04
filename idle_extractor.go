package chat

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// MemoryExtractor triggers memory extraction for conversations.
type MemoryExtractor interface {
	ExtractMemories(ctx context.Context, conversationID, agentSlug, userID string) error
}

// IdleExtractor tracks conversation activity and triggers memory extraction
// when a conversation has been idle for a configured duration.
type IdleExtractor struct {
	extractor   MemoryExtractor
	idleTimeout time.Duration
	ctx         context.Context

	mu      sync.Mutex
	timers  map[string]*time.Timer // keyed by conversationID
	convMeta map[string]conversationMeta
}

type conversationMeta struct {
	agentSlug string
	userID    string
}

// NewIdleExtractor creates an IdleExtractor that triggers extraction after idleTimeout.
func NewIdleExtractor(ctx context.Context, extractor MemoryExtractor, idleTimeout time.Duration) *IdleExtractor {
	return &IdleExtractor{
		extractor:   extractor,
		idleTimeout: idleTimeout,
		ctx:         ctx,
		timers:      make(map[string]*time.Timer),
		convMeta:    make(map[string]conversationMeta),
	}
}

// Touch records activity for a conversation, resetting its idle timer.
func (ie *IdleExtractor) Touch(conversationID, agentSlug, userID string) {
	ie.mu.Lock()
	defer ie.mu.Unlock()

	// Cancel existing timer
	if timer, ok := ie.timers[conversationID]; ok {
		timer.Stop()
	}

	ie.convMeta[conversationID] = conversationMeta{
		agentSlug: agentSlug,
		userID:    userID,
	}

	ie.timers[conversationID] = time.AfterFunc(ie.idleTimeout, func() {
		ie.extract(conversationID)
	})
}

func (ie *IdleExtractor) extract(conversationID string) {
	ie.mu.Lock()
	meta, ok := ie.convMeta[conversationID]
	delete(ie.timers, conversationID)
	delete(ie.convMeta, conversationID)
	ie.mu.Unlock()

	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(ie.ctx, 30*time.Second)
	defer cancel()

	if err := ie.extractor.ExtractMemories(ctx, conversationID, meta.agentSlug, meta.userID); err != nil {
		slog.Error("idle memory extraction failed",
			"conversation_id", conversationID,
			"agent", meta.agentSlug,
			"error", err,
		)
		return
	}

	slog.Info("idle memory extraction triggered",
		"conversation_id", conversationID,
		"agent", meta.agentSlug,
	)
}

// FlushAll triggers extraction for all tracked conversations and clears state.
// Call this during graceful shutdown.
func (ie *IdleExtractor) FlushAll() {
	ie.mu.Lock()
	conversations := make(map[string]conversationMeta, len(ie.convMeta))
	for id, meta := range ie.convMeta {
		conversations[id] = meta
		if timer, ok := ie.timers[id]; ok {
			timer.Stop()
		}
	}
	ie.timers = make(map[string]*time.Timer)
	ie.convMeta = make(map[string]conversationMeta)
	ie.mu.Unlock()

	for id, meta := range conversations {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := ie.extractor.ExtractMemories(ctx, id, meta.agentSlug, meta.userID); err != nil {
			slog.Error("shutdown memory extraction failed",
				"conversation_id", id,
				"error", err,
			)
		} else {
			slog.Info("shutdown memory extraction triggered",
				"conversation_id", id,
				"agent", meta.agentSlug,
			)
		}
		cancel()
	}
}
