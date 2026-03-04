package chat

import "context"

// MemoryRecaller retrieves memories relevant to a query for an agent/user pair.
type MemoryRecaller interface {
	RecallMemories(ctx context.Context, agentSlug, userID, query string, limit int) (*MemoryRecallResponse, error)
}

// MemoryRecallResponse contains recalled memories and relationship context.
type MemoryRecallResponse struct {
	Memories            []RecalledMemory     `json:"memories"`
	RelationshipContext *RelationshipContext `json:"relationship_context"`
}

// RecalledMemory is a memory returned from recall with relevance scoring.
type RecalledMemory struct {
	Type             string  `json:"type"`
	Content          string  `json:"content"`
	EmotionalContext string  `json:"emotional_context"`
	Importance       float64 `json:"importance"`
	RelevanceScore   float64 `json:"relevance_score"`
}

// RelationshipContext provides relationship state for recall responses.
type RelationshipContext struct {
	Trust              float64 `json:"trust"`
	Intimacy           float64 `json:"intimacy"`
	Autonomy           float64 `json:"autonomy"`
	Reciprocity        float64 `json:"reciprocity"`
	Playfulness        float64 `json:"playfulness"`
	Conflict           float64 `json:"conflict"`
	PersonalityContext string  `json:"personality_context"`
	TotalConversations int     `json:"total_conversations"`
	TotalMemories      int     `json:"total_memories"`
}
