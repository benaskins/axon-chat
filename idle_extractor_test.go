package chat

import (
	"context"
	"sync"
	"testing"
	"time"
)

type mockExtractor struct {
	mu          sync.Mutex
	extractions []extractionCall
}

type extractionCall struct {
	conversationID string
	agentSlug      string
	userID         string
}

func (m *mockExtractor) ExtractMemories(ctx context.Context, conversationID, agentSlug, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.extractions = append(m.extractions, extractionCall{
		conversationID: conversationID,
		agentSlug:      agentSlug,
		userID:         userID,
	})
	return nil
}

func (m *mockExtractor) calls() []extractionCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]extractionCall, len(m.extractions))
	copy(cp, m.extractions)
	return cp
}

func TestIdleExtractor_TriggersAfterIdle(t *testing.T) {
	ext := &mockExtractor{}
	ie := NewIdleExtractor(context.Background(), ext, 50*time.Millisecond)

	ie.Touch("conv-1", "helper", "user1")

	// Should not have triggered yet
	if len(ext.calls()) != 0 {
		t.Fatal("expected no extractions immediately after touch")
	}

	// Wait for idle timeout
	time.Sleep(100 * time.Millisecond)

	calls := ext.calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 extraction, got %d", len(calls))
	}
	if calls[0].conversationID != "conv-1" {
		t.Errorf("expected conv-1, got %s", calls[0].conversationID)
	}
	if calls[0].agentSlug != "helper" {
		t.Errorf("expected helper, got %s", calls[0].agentSlug)
	}
}

func TestIdleExtractor_ResetsTimerOnTouch(t *testing.T) {
	ext := &mockExtractor{}
	ie := NewIdleExtractor(context.Background(), ext, 80*time.Millisecond)

	ie.Touch("conv-1", "helper", "user1")
	time.Sleep(50 * time.Millisecond)

	// Touch again — should reset the timer
	ie.Touch("conv-1", "helper", "user1")
	time.Sleep(50 * time.Millisecond)

	// Should not have triggered yet (only 50ms since last touch, need 80ms)
	if len(ext.calls()) != 0 {
		t.Fatal("expected no extractions — timer should have been reset")
	}

	// Wait for remaining timeout
	time.Sleep(50 * time.Millisecond)

	calls := ext.calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 extraction after reset, got %d", len(calls))
	}
}

func TestIdleExtractor_TracksMultipleConversations(t *testing.T) {
	ext := &mockExtractor{}
	ie := NewIdleExtractor(context.Background(), ext, 50*time.Millisecond)

	ie.Touch("conv-1", "helper", "user1")
	ie.Touch("conv-2", "bot", "user2")

	time.Sleep(100 * time.Millisecond)

	calls := ext.calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 extractions, got %d", len(calls))
	}
}

func TestIdleExtractor_FlushAll(t *testing.T) {
	ext := &mockExtractor{}
	ie := NewIdleExtractor(context.Background(), ext, 1*time.Hour) // Long timeout — won't fire naturally

	ie.Touch("conv-1", "helper", "user1")
	ie.Touch("conv-2", "bot", "user2")

	ie.FlushAll()

	calls := ext.calls()
	if len(calls) != 2 {
		t.Fatalf("expected 2 extractions from flush, got %d", len(calls))
	}

	// State should be cleared — no double-extraction
	time.Sleep(50 * time.Millisecond)
	if len(ext.calls()) != 2 {
		t.Error("expected no additional extractions after flush")
	}
}

func TestIdleExtractor_FlushAll_NoActiveConversations(t *testing.T) {
	ext := &mockExtractor{}
	ie := NewIdleExtractor(context.Background(), ext, 1*time.Hour)

	// Flush with nothing tracked — should not panic
	ie.FlushAll()

	if len(ext.calls()) != 0 {
		t.Error("expected no extractions when nothing is tracked")
	}
}
