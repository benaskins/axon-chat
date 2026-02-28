package chat

import (
	"context"

	ollamaapi "github.com/ollama/ollama/api"
)

// ChatClient abstracts the Ollama chat call for testing.
type ChatClient interface {
	Chat(ctx context.Context, req *ollamaapi.ChatRequest, fn ollamaapi.ChatResponseFunc) error
}

// ModelLister abstracts listing available models for testing.
type ModelLister interface {
	List(ctx context.Context) (*ollamaapi.ListResponse, error)
}
