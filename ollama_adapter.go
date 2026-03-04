package chat

import (
	"context"

	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
	ollamaapi "github.com/ollama/ollama/api"
)

// OllamaAdapter implements loop.ChatClient by translating to/from
// the Ollama API types. This is where the Ollama dependency concentrates.
type OllamaAdapter struct {
	client ChatClient
}

// NewOllamaAdapter wraps an Ollama ChatClient as a loop.ChatClient.
func NewOllamaAdapter(client ChatClient) *OllamaAdapter {
	return &OllamaAdapter{client: client}
}

// Chat translates a loop.ChatRequest to an Ollama request, calls the
// underlying client, and translates responses back.
func (a *OllamaAdapter) Chat(ctx context.Context, req *loop.ChatRequest, fn func(loop.ChatResponse) error) error {
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

	if req.Think != nil {
		tv := ollamaapi.ThinkValue{Value: *req.Think}
		ollamaReq.Think = &tv
	}

	if len(req.Tools) > 0 {
		ollamaReq.Tools = toOllamaToolDefs(req.Tools)
	}

	// Keep model loaded indefinitely
	keepAlive := ollamaapi.Duration{Duration: -1}
	ollamaReq.KeepAlive = &keepAlive

	return a.client.Chat(ctx, ollamaReq, func(resp ollamaapi.ChatResponse) error {
		return fn(fromOllamaResponse(resp))
	})
}

// toOllamaMessages converts agent messages to Ollama messages.
func toOllamaMessages(msgs []loop.Message) []ollamaapi.Message {
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
func toOllamaToolCalls(calls []loop.ToolCall) []ollamaapi.ToolCall {
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
func fromOllamaResponse(resp ollamaapi.ChatResponse) loop.ChatResponse {
	result := loop.ChatResponse{
		Content:  resp.Message.Content,
		Thinking: resp.Message.Thinking,
		Done:     resp.Done,
	}

	if len(resp.Message.ToolCalls) > 0 {
		result.ToolCalls = fromOllamaToolCalls(resp.Message.ToolCalls)
	}

	return result
}

// toOllamaToolDefs converts axon-tool ToolDefs to Ollama tool definitions.
func toOllamaToolDefs(defs []tool.ToolDef) ollamaapi.Tools {
	tools := make(ollamaapi.Tools, len(defs))
	for i, d := range defs {
		props := ollamaapi.NewToolPropertiesMap()
		for name, prop := range d.Parameters.Properties {
			props.Set(name, ollamaapi.ToolProperty{
				Type:        ollamaapi.PropertyType{prop.Type},
				Description: prop.Description,
			})
		}
		tools[i] = ollamaapi.Tool{
			Type: "function",
			Function: ollamaapi.ToolFunction{
				Name:        d.Name,
				Description: d.Description,
				Parameters: ollamaapi.ToolFunctionParameters{
					Type:       d.Parameters.Type,
					Required:   d.Parameters.Required,
					Properties: props,
				},
			},
		}
	}
	return tools
}

// fromOllamaToolCalls converts Ollama tool calls to agent tool calls.
func fromOllamaToolCalls(calls []ollamaapi.ToolCall) []loop.ToolCall {
	result := make([]loop.ToolCall, len(calls))
	for i, tc := range calls {
		result[i] = loop.ToolCall{
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments.ToMap(),
		}
	}
	return result
}
