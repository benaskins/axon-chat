# axon-chat

Chat service with LLM integration, tool calling, SSE streaming, and agent management.

## Build & Test

```bash
go test ./...
go vet ./...
```

## Key Files

- `chat.go` — core chat service logic
- `config.go` — service configuration
- `domain_events.go` — event definitions for chat domain
- `analytics.go` — analytics event tracking
