# axon-chat

Chat service with LLM integration, tool calling, SSE streaming, and agent management.

## Build & Test

```bash
go test ./...
go vet ./...
```

## Key Files

- `chat.go` — core chat service logic
- `server.go` — Server struct, constructor, and option wiring
- `handler.go` — HTTP handler and route registration
- `handler_sync.go` — synchronous chat endpoint (non-streaming)
- `config.go` — service configuration
- `domain_events.go` — event definitions for chat domain
- `tool_router.go` — routes tool calls to agent-enabled subsets
- `fetch_page.go` — rate-limited web page fetcher with LLM extraction
- `analytics.go` — analytics event tracking
