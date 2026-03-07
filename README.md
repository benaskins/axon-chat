# axon-chat

A chat service library with LLM integration, tool calling, SSE streaming, and agent management. Part of [lamina](https://github.com/benaskins/lamina) — each axon package can be used independently.

Defines domain types, store interfaces, HTTP handlers, and integration clients. PostgreSQL implementations live in the consuming application's composition root.

## Install

```
go get github.com/benaskins/axon-chat@latest
```

Requires Go 1.24+.

## Usage

```go
srv := chat.NewServer(ollamaClient,
    chat.WithStore(store),
    chat.WithDefaultModel("llama3.3:70b-instruct-q8_0"),
    chat.WithCookieDomain(".example.com"),
    chat.WithStaticFiles(staticFiles),
    chat.WithEventBus(eventBus),
    chat.WithShutdownContext(ctx),
)

http.Handle("/", srv.Handler(authMiddleware))
```

### Key types

- `Agent` — configurable LLM agent with system prompt and tools
- `Conversation`, `Message` — chat domain types
- `Store` — persistence interface for conversations, messages, and agents
- `Server` — HTTP server with SSE streaming and tool dispatch
- `OllamaAdapter` — Ollama LLM adapter
- `PageFetcher` — rate-limited web page fetcher with LLM extraction

### Sub-packages

- `chattest` — in-memory mock store for testing

## License

MIT — see [LICENSE](LICENSE).
