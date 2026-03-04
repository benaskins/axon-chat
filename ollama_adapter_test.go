package chat

import (
	"context"
	"testing"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
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
	err := adapter.Chat(context.Background(), &loop.ChatRequest{
		Model:    "llama3",
		Messages: []loop.Message{{Role: "user", Content: "Hi"}},
		Stream:   true,
	}, func(resp loop.ChatResponse) error {
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

	var toolCalls []loop.ToolCall
	err := adapter.Chat(context.Background(), &loop.ChatRequest{
		Model:    "test",
		Messages: []loop.Message{{Role: "user", Content: "search"}},
	}, func(resp loop.ChatResponse) error {
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

func TestOllamaAdapter_ToolDefsConverted(t *testing.T) {
	stub := &stubOllamaClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Content: "ok"}, Done: true},
		},
	}

	adapter := NewOllamaAdapter(stub)
	err := adapter.Chat(context.Background(), &loop.ChatRequest{
		Model:    "test",
		Messages: []loop.Message{{Role: "user", Content: "time?"}},
		Tools: []tool.ToolDef{
			{
				Name:        "current_time",
				Description: "Get the current time",
				Parameters: tool.ParameterSchema{
					Type:       "object",
					Required:   []string{},
					Properties: map[string]tool.PropertySchema{},
				},
			},
		},
	}, func(resp loop.ChatResponse) error {
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(stub.lastReq.Tools) != 1 {
		t.Fatalf("expected 1 tool in Ollama request, got %d", len(stub.lastReq.Tools))
	}
	if stub.lastReq.Tools[0].Function.Name != "current_time" {
		t.Errorf("tool name = %q, want %q", stub.lastReq.Tools[0].Function.Name, "current_time")
	}
	if stub.lastReq.Tools[0].Function.Description != "Get the current time" {
		t.Errorf("tool description = %q, want %q", stub.lastReq.Tools[0].Function.Description, "Get the current time")
	}
}

func TestOllamaAdapter_ToolDefsWithParameters(t *testing.T) {
	stub := &stubOllamaClient{
		responses: []ollamaapi.ChatResponse{
			{Message: ollamaapi.Message{Content: "ok"}, Done: true},
		},
	}

	adapter := NewOllamaAdapter(stub)
	err := adapter.Chat(context.Background(), &loop.ChatRequest{
		Model:    "test",
		Messages: []loop.Message{{Role: "user", Content: "search"}},
		Tools: []tool.ToolDef{
			{
				Name:        "web_search",
				Description: "Search the web",
				Parameters: tool.ParameterSchema{
					Type:     "object",
					Required: []string{"query"},
					Properties: map[string]tool.PropertySchema{
						"query": {Type: "string", Description: "The search query"},
					},
				},
			},
		},
	}, func(resp loop.ChatResponse) error {
		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
	if len(stub.lastReq.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(stub.lastReq.Tools))
	}

	params := stub.lastReq.Tools[0].Function.Parameters
	if params.Type != "object" {
		t.Errorf("params type = %q, want %q", params.Type, "object")
	}
	if len(params.Required) != 1 || params.Required[0] != "query" {
		t.Errorf("required = %v, want [query]", params.Required)
	}

	queryProp, ok := params.Properties.Get("query")
	if !ok {
		t.Fatal("expected 'query' property")
	}
	if queryProp.Description != "The search query" {
		t.Errorf("query description = %q, want %q", queryProp.Description, "The search query")
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
	err := adapter.Chat(context.Background(), &loop.ChatRequest{
		Model:    "test",
		Messages: []loop.Message{{Role: "user", Content: "think"}},
	}, func(resp loop.ChatResponse) error {
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
