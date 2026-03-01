package chat

import (
	"context"
	"testing"

	agent "github.com/benaskins/axon-agent"
	ollamaapi "github.com/ollama/ollama/api"
)

type stubOllamaClient struct {
	responses []ollamaapi.ChatResponse
	lastReq   *ollamaapi.ChatRequest
}

func (s *stubOllamaClient) Chat(ctx context.Context, req *ollamaapi.ChatRequest, fn ollamaapi.ChatResponseFunc) error {
	s.lastReq = req
	for _, resp := range s.responses {
		if err := fn(resp); err != nil {
			return err
		}
	}
	return nil
}

func TestOllamaAdapter_BasicChat(t *testing.T) {
	stub := &stubOllamaClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Content: "Hello!"}},
		},
	}

	adapter := NewOllamaAdapter(stub)

	var content string
	err := adapter.Chat(context.Background(), &agent.ChatRequest{
		Model:    "llama3",
		Messages: []agent.Message{{Role: "user", Content: "Hi"}},
		Stream:   true,
	}, func(resp agent.ChatResponse) error {
		content += resp.Content
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if content != "Hello!" {
		t.Errorf("got %q, want %q", content, "Hello!")
	}

	// Verify the request was translated correctly
	if stub.lastReq.Model != "llama3" {
		t.Errorf("model = %q, want %q", stub.lastReq.Model, "llama3")
	}
	if len(stub.lastReq.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(stub.lastReq.Messages))
	}
	if stub.lastReq.Messages[0].Role != "user" {
		t.Errorf("role = %q, want %q", stub.lastReq.Messages[0].Role, "user")
	}
}

func TestOllamaAdapter_ToolCallsTranslated(t *testing.T) {
	args := ollamaapi.NewToolCallFunctionArguments()
	args.Set("query", "test")

	stub := &stubOllamaClient{
		responses: []ollamaapi.ChatResponse{
			{
				Message: ollamaapi.Message{
					ToolCalls: []ollamaapi.ToolCall{
						{Function: ollamaapi.ToolCallFunction{Name: "web_search", Arguments: args}},
					},
				},
			},
		},
	}

	adapter := NewOllamaAdapter(stub)

	var toolCalls []agent.ToolCall
	err := adapter.Chat(context.Background(), &agent.ChatRequest{
		Model:    "test",
		Messages: []agent.Message{{Role: "user", Content: "search"}},
	}, func(resp agent.ChatResponse) error {
		toolCalls = append(toolCalls, resp.ToolCalls...)
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(toolCalls))
	}
	if toolCalls[0].Name != "web_search" {
		t.Errorf("tool name = %q, want %q", toolCalls[0].Name, "web_search")
	}
	if toolCalls[0].Arguments["query"] != "test" {
		t.Errorf("tool arg query = %v, want %q", toolCalls[0].Arguments["query"], "test")
	}
}

func TestOllamaAdapter_ThinkingTranslated(t *testing.T) {
	stub := &stubOllamaClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Thinking: "Let me think..."}},
			{Message: ollamaapi.Message{Content: "Answer."}},
		},
	}

	adapter := NewOllamaAdapter(stub)

	var thinking, content string
	err := adapter.Chat(context.Background(), &agent.ChatRequest{
		Model:    "test",
		Messages: []agent.Message{{Role: "user", Content: "think"}},
	}, func(resp agent.ChatResponse) error {
		thinking += resp.Thinking
		content += resp.Content
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if thinking != "Let me think..." {
		t.Errorf("thinking = %q, want %q", thinking, "Let me think...")
	}
	if content != "Answer." {
		t.Errorf("content = %q, want %q", content, "Answer.")
	}
}
