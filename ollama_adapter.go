package chat

import (
	"context"

	agent "github.com/benaskins/axon-agent"
	ollamaapi "github.com/ollama/ollama/api"
)

// OllamaAdapter implements agent.ChatClient by translating to/from
// the Ollama API types. This is where the Ollama dependency concentrates.
type OllamaAdapter struct {
	client ChatClient
}

// NewOllamaAdapter wraps an Ollama ChatClient as an agent.ChatClient.
func NewOllamaAdapter(client ChatClient) *OllamaAdapter {
	return &OllamaAdapter{client: client}
}

// Chat translates an agent.ChatRequest to an Ollama request, calls the
// underlying client, and translates responses back.
func (a *OllamaAdapter) Chat(ctx context.Context, req *agent.ChatRequest, fn func(agent.ChatResponse) error) error {
	ollamaReq := &ollamaapi.ChatRequest{
		Model:    req.Model,
		Messages: toOllamaMessages(req.Messages),
	}

	if req.Stream {
		stream := true
		ollamaReq.Stream = &stream
	}

	if req.Options != nil {
		ollamaReq.Options = req.Options
	}

	return a.client.Chat(ctx, ollamaReq, func(resp ollamaapi.ChatResponse) error {
		return fn(fromOllamaResponse(resp))
	})
}

// toOllamaMessages converts agent messages to Ollama messages.
func toOllamaMessages(msgs []agent.Message) []ollamaapi.Message {
	result := make([]ollamaapi.Message, len(msgs))
	for i, m := range msgs {
		result[i] = ollamaapi.Message{
			Role:     m.Role,
			Content:  m.Content,
			Thinking: m.Thinking,
		}
		if len(m.ToolCalls) > 0 {
			result[i].ToolCalls = toOllamaToolCalls(m.ToolCalls)
		}
	}
	return result
}

// toOllamaToolCalls converts agent tool calls to Ollama tool calls.
func toOllamaToolCalls(calls []agent.ToolCall) []ollamaapi.ToolCall {
	result := make([]ollamaapi.ToolCall, len(calls))
	for i, tc := range calls {
		args := ollamaapi.NewToolCallFunctionArguments()
		for k, v := range tc.Arguments {
			args.Set(k, v)
		}
		result[i] = ollamaapi.ToolCall{
			Function: ollamaapi.ToolCallFunction{
				Name:      tc.Name,
				Arguments: args,
			},
		}
	}
	return result
}

// fromOllamaResponse converts an Ollama response to an agent response.
func fromOllamaResponse(resp ollamaapi.ChatResponse) agent.ChatResponse {
	result := agent.ChatResponse{
		Content:  resp.Message.Content,
		Thinking: resp.Message.Thinking,
		Done:     resp.Done,
	}

	if len(resp.Message.ToolCalls) > 0 {
		result.ToolCalls = fromOllamaToolCalls(resp.Message.ToolCalls)
	}

	return result
}

// fromOllamaToolCalls converts Ollama tool calls to agent tool calls.
func fromOllamaToolCalls(calls []ollamaapi.ToolCall) []agent.ToolCall {
	result := make([]agent.ToolCall, len(calls))
	for i, tc := range calls {
		result[i] = agent.ToolCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments.ToMap(),
		}
	}
	return result
}
