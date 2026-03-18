//go:build ignore

// Example: wiring up axon-chat handlers in a composition root.
//
// This shows the minimal setup for a chat server with an in-memory store,
// SSE event bus, and embedded SvelteKit frontend. A real composition root
// would add a Postgres-backed store, auth middleware, and additional options.
//
// For a full working example, see examples/chat/ in the lamina workspace:
// https://github.com/benaskins/lamina-mono/tree/main/examples/chat
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	chat "github.com/benaskins/axon-chat"
	"github.com/benaskins/axon-chat/chattest"
	"github.com/benaskins/axon/sse"
	loop "github.com/benaskins/axon-loop"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var llm loop.LLMClient // supply your own implementation

	store := chattest.NewMemoryStore()
	eventBus := sse.NewEventBus[chat.Event]()
	staticFiles := &chat.StaticFiles

	srv := chat.NewServer(llm,
		chat.WithStore(store),
		chat.WithDefaultModel("llama3.3:70b-instruct-q8_0"),
		chat.WithCookieDomain(".example.com"),
		chat.WithStaticFiles(staticFiles),
		chat.WithEventBus(eventBus),
		chat.WithShutdownContext(ctx),
	)

	mux := http.NewServeMux()
	mux.Handle("/api/", srv.Handler())
	mux.Handle("/", srv.SPAHandler())

	slog.Info("listening", "addr", ":8090")
	http.ListenAndServe(":8090", mux)
}
