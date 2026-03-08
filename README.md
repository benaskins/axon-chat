# axon-chat

> Domain package · Part of the [lamina](https://github.com/benaskins/lamina-mono) workspace

Chat service with LLM-powered agents, SSE streaming, and tool calling. Provides domain types, store interfaces, HTTP handlers, and an embedded SvelteKit frontend (`//go:embed`). Persistence implementations live in the consuming application's composition root.

## Getting started

```
go get github.com/benaskins/axon-chat@latest
```

Requires Go 1.25+.

axon-chat is a domain package — it provides handlers and types but no `main`. You assemble it in your own composition root alongside an LLM client, store, and auth middleware:

```go
srv := chat.NewServer(llm,
    chat.WithStore(store),
    chat.WithDefaultModel("llama3.3:70b-instruct-q8_0"),
    chat.WithEventBus(eventBus),
    chat.WithStaticFiles(&chat.StaticFiles),
    chat.WithShutdownContext(ctx),
)

handler := srv.Handler(authMiddleware)
```

See [`example/main.go`](example/main.go) for a minimal wiring example, or the full composition root in [`examples/chat/`](https://github.com/benaskins/lamina-mono/tree/main/examples/chat) in the lamina workspace.

## Key types

- **`Server`** — HTTP server with SSE streaming and tool dispatch. Configured via `Option` functions.
- **`Agent`** — configurable LLM agent with system prompt, model parameters, and tools.
- **`Conversation`**, **`Message`** — chat domain types.
- **`Store`** — persistence interface combining `ReadStore` and `ReadModelWriter`.
- **`ToolRouter`** — routes tool calls to the relevant subset based on user message.
- **`PageFetcher`** — rate-limited web page fetcher with LLM extraction.
- **`chattest.MemoryStore`** — in-memory store for testing.

## License

MIT — see [LICENSE](LICENSE).
