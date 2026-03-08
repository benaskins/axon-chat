package chat

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	loop "github.com/benaskins/axon-loop"
)

const (
	defaultQualifierModel = "qwen2.5:3b"
	qualifierTimeout = 10 * time.Second
)

// SearchQualifier uses a small LLM to refine search queries based on the
// agent's system prompt, producing more targeted and contextually relevant
// search terms.
type SearchQualifier struct {
	llm   loop.LLMClient
	model string
}

// NewSearchQualifier creates a new search qualifier.
// If model is empty, defaults to the built-in qualifier model.
func NewSearchQualifier(llm loop.LLMClient, model string) *SearchQualifier {
	if model == "" {
		model = defaultQualifierModel
	}
	return &SearchQualifier{llm: llm, model: model}
}

// Qualify takes a raw search query and the agent's system prompt, and returns
// a refined query that incorporates the agent's domain expertise and context.
// If qualification fails, returns the original query unchanged.
func (sq *SearchQualifier) Qualify(ctx context.Context, query, systemPrompt string) string {
	if sq.llm == nil || strings.TrimSpace(systemPrompt) == "" {
		return query
	}

	qualCtx, cancel := context.WithTimeout(ctx, qualifierTimeout)
	defer cancel()

	prompt := fmt.Sprintf(`You are a search query optimizer. Given an agent's identity/expertise and a search query, refine the query to be more precise and targeted.

Rules:
- Output ONLY the refined search query, nothing else
- Keep it concise (under 10 words if possible)
- Add relevant domain-specific terms from the agent's expertise
- Don't change the intent of the original query
- If the query is already specific enough, return it unchanged
- Never add quotes or explanations

Agent context:
%s

Original query: %s

Refined query:`, systemPrompt, query)

	var result strings.Builder

	start := time.Now()
	err := sq.llm.Chat(qualCtx, &loop.Request{
		Model:    sq.model,
		Messages: []loop.Message{{Role: "user", Content: prompt}},
	}, func(resp loop.Response) error {
		result.WriteString(resp.Content)
		return nil
	})

	latency := time.Since(start).Milliseconds()

	if err != nil {
		slog.Warn("search qualifier failed, using original query", "error", err, "latency_ms", latency)
		return query
	}

	refined := strings.TrimSpace(result.String())
	if refined == "" {
		return query
	}

	slog.Info("search query qualified",
		"original", query,
		"refined", refined,
		"latency_ms", latency,
	)

	return refined
}
