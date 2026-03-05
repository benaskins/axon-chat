package chat

import (
	"context"
	"strings"
	"testing"

	tool "github.com/benaskins/axon-tool"
)

type mockMemoryRecaller struct {
	response *MemoryRecallResponse
	err      error
	called   bool
	query    string
}

func (m *mockMemoryRecaller) RecallMemories(ctx context.Context, agentSlug, userID, query string, limit int) (*MemoryRecallResponse, error) {
	m.called = true
	m.query = query
	return m.response, m.err
}

func TestRecallMemoryTool_ReturnsFormattedMemories(t *testing.T) {
	recaller := &mockMemoryRecaller{
		response: &MemoryRecallResponse{
			Memories: []RecalledMemory{
				{Type: "episodic", Content: "We talked about hiking in Tasmania", Importance: 0.8, RelevanceScore: 0.9},
				{Type: "emotional", Content: "User was excited about the trip", EmotionalContext: "excitement, joy", Importance: 0.7, RelevanceScore: 0.85},
				{Type: "semantic", Content: "User's favourite trail is the Overland Track", Importance: 0.6, RelevanceScore: 0.7},
			},
			RelationshipContext: &RelationshipContext{
				Trust:              0.7,
				Intimacy:           0.5,
				Autonomy:           0.6,
				Reciprocity:        0.5,
				Playfulness:        0.4,
				Conflict:           0.1,
				TotalConversations: 12,
				TotalMemories:      45,
			},
		},
	}

	h := newChatHandler(testModel, nil, nil, context.Background(), nil)
	h.memoryRecaller = recaller

	toolDef := h.recallMemoryTool()

	ctx := &tool.ToolContext{
		Ctx:       context.Background(),
		AgentSlug: "helper",
		UserID:    "user1",
	}

	result := toolDef.Execute(ctx, map[string]any{"query": "hiking"})

	if !recaller.called {
		t.Fatal("expected recaller to be called")
	}
	if recaller.query != "hiking" {
		t.Errorf("expected query 'hiking', got %q", recaller.query)
	}

	// Should contain memory content
	if !strings.Contains(result.Content, "hiking in Tasmania") {
		t.Error("expected result to contain memory content")
	}
	if !strings.Contains(result.Content, "Overland Track") {
		t.Error("expected result to contain semantic memory")
	}

	// Should contain relationship metrics
	if !strings.Contains(result.Content, "Trust") {
		t.Error("expected result to contain relationship metrics")
	}
}

func TestRecallMemoryTool_NotConfigured(t *testing.T) {
	h := newChatHandler(testModel, nil, nil, context.Background(), nil)
	// memoryRecaller is nil

	toolDef := h.recallMemoryTool()

	ctx := &tool.ToolContext{Ctx: context.Background()}
	result := toolDef.Execute(ctx, map[string]any{"query": "hiking"})

	if !strings.Contains(result.Content, "not available") {
		t.Errorf("expected 'not available' message, got: %s", result.Content)
	}
}

func TestRecallMemoryTool_EmptyQuery(t *testing.T) {
	recaller := &mockMemoryRecaller{}
	h := newChatHandler(testModel, nil, nil, context.Background(), nil)
	h.memoryRecaller = recaller

	toolDef := h.recallMemoryTool()

	ctx := &tool.ToolContext{Ctx: context.Background()}
	result := toolDef.Execute(ctx, map[string]any{"query": ""})

	if !strings.Contains(result.Content, "query is required") {
		t.Errorf("expected error about query, got: %s", result.Content)
	}
}

func TestRecallMemoryTool_NoMemoriesFound(t *testing.T) {
	recaller := &mockMemoryRecaller{
		response: &MemoryRecallResponse{
			Memories:            []RecalledMemory{},
			RelationshipContext: &RelationshipContext{Trust: 0.5},
		},
	}

	h := newChatHandler(testModel, nil, nil, context.Background(), nil)
	h.memoryRecaller = recaller

	toolDef := h.recallMemoryTool()

	ctx := &tool.ToolContext{
		Ctx:       context.Background(),
		AgentSlug: "helper",
		UserID:    "user1",
	}
	result := toolDef.Execute(ctx, map[string]any{"query": "something unknown"})

	if !strings.Contains(result.Content, "No memories found") {
		t.Errorf("expected 'No memories found', got: %s", result.Content)
	}
}

func TestBuildSystemPrompt_RecallMemory(t *testing.T) {
	a := Agent{
		SystemPrompt: "You are Hal.",
		Tools:        []string{"recall_memory"},
	}
	result := BuildSystemPrompt(a)

	if !strings.Contains(result, "## Memory") {
		t.Error("expected Memory section in system prompt")
	}
	if !strings.Contains(result, "recall_memory") {
		t.Error("expected recall_memory tool reference in system prompt")
	}
}
