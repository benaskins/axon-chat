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
- `handler_agents.go` — agent CRUD endpoints
- `handler_conversations.go` — conversation and message endpoints
- `handler_events.go` — SSE streaming endpoint
- `handler_internal.go` — internal service-to-service endpoints
- `handler_models.go` — model listing endpoint
- `handler_sync.go` — synchronous chat endpoint (non-streaming)
- `handler_webhook.go` — webhook endpoints for external events
- `config.go` — service configuration
- `types.go` — domain types (Agent, Conversation, Message)
- `store.go` — persistence interfaces (ReadStore, ReadModelWriter)
- `events.go` — domain event helpers and types
- `domain_events.go` — event definitions for chat domain
- `projectors.go` — event projectors for read model updates
- `tool_router.go` — routes tool calls to agent-enabled subsets
- `tools.go` — tool registry and definitions
- `fetch_page.go` — rate-limited web page fetcher with LLM extraction
- `memory.go` — memory service client and types
- `idle_extractor.go` — idle conversation tracking and memory extraction
- `weather.go` — weather service client
- `taskrunner.go` — async task runner client
- `analytics.go` — analytics event tracking