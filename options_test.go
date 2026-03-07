package chat

import (
	"context"
	"embed"
	"testing"

	fact "github.com/benaskins/axon-fact"
	loop "github.com/benaskins/axon-loop"
	"github.com/benaskins/axon/sse"
)

func TestNewServer_DefaultsWithLLMOnly(t *testing.T) {
	llm := &mockLLMClient{}
	srv := NewServer(llm)

	if srv.chat == nil {
		t.Fatal("chat handler should be initialized")
	}
	if srv.chat.llm != llm {
		t.Error("LLM client should be set")
	}
	if srv.chat.shutdownCtx == nil {
		t.Error("shutdownCtx should default to context.Background()")
	}
}

func TestNewServer_WithStore(t *testing.T) {
	store := newMemoryStore()
	srv := NewServer(&mockLLMClient{}, WithStore(store))

	if srv.chat.store != store {
		t.Error("store should be the one provided via WithStore")
	}
}

func TestNewServer_WithDefaultModel(t *testing.T) {
	srv := NewServer(&mockLLMClient{}, WithDefaultModel("claude-3"))

	if srv.chat.defaultModel != "claude-3" {
		t.Errorf("defaultModel = %q, want claude-3", srv.chat.defaultModel)
	}
}

func TestNewServer_WithCookieDomain(t *testing.T) {
	srv := NewServer(&mockLLMClient{}, WithCookieDomain("studio.internal"))

	if srv.config.CookieDomain != "studio.internal" {
		t.Errorf("CookieDomain = %q, want studio.internal", srv.config.CookieDomain)
	}
}

func TestNewServer_WithStaticFiles(t *testing.T) {
	var fs embed.FS
	srv := NewServer(&mockLLMClient{}, WithStaticFiles(&fs))

	if srv.staticFiles != &fs {
		t.Error("staticFiles should be set")
	}
}

func TestNewServer_WithEventBus(t *testing.T) {
	bus := sse.NewEventBus[Event]()
	srv := NewServer(&mockLLMClient{}, WithEventBus(bus))

	if srv.chat.eventBus != bus {
		t.Error("eventBus should be set")
	}
}

func TestNewServer_WithShutdownContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := NewServer(&mockLLMClient{}, WithShutdownContext(ctx))

	if srv.chat.shutdownCtx != ctx {
		t.Error("shutdownCtx should be set")
	}
}

func TestNewServer_WithModelLister(t *testing.T) {
	lister := &mockModelLister{}
	srv := NewServer(&mockLLMClient{}, WithModelLister(lister))

	if srv.ModelLister != lister {
		t.Error("ModelLister should be set")
	}
}

func TestNewServer_WithEventStore(t *testing.T) {
	store := newMemoryStore()
	projectors := DefaultProjectors(store, store)

	// Simulates composition root wiring: create EventStore with projectors
	var opts []fact.Option
	for _, p := range projectors {
		opts = append(opts, fact.WithProjector(p))
	}
	es := fact.NewMemoryStore(opts...)

	srv := NewServer(&mockLLMClient{},
		WithStore(store),
		WithEventStore(es),
	)

	if srv.chat.eventStore != es {
		t.Error("eventStore should be set")
	}
}

func TestNewServer_MultipleOptions(t *testing.T) {
	store := newMemoryStore()
	srv := NewServer(&mockLLMClient{},
		WithStore(store),
		WithDefaultModel("gpt-4"),
		WithCookieDomain("example.com"),
	)

	if srv.chat.store != store {
		t.Error("store mismatch")
	}
	if srv.chat.defaultModel != "gpt-4" {
		t.Errorf("defaultModel = %q", srv.chat.defaultModel)
	}
	if srv.config.CookieDomain != "example.com" {
		t.Errorf("CookieDomain = %q", srv.config.CookieDomain)
	}
}

type mockLLMClient struct{}

func (m *mockLLMClient) Chat(ctx context.Context, req *loop.Request, fn func(loop.Response) error) error {
	return fn(loop.Response{Content: "mock", Done: true})
}

func TestEventStore_AgentSaveEmitsEvent(t *testing.T) {
	store := newMemoryStore()
	projectors := DefaultProjectors(store, store)

	var opts []fact.Option
	for _, p := range projectors {
		opts = append(opts, fact.WithProjector(p))
	}
	es := fact.NewMemoryStore(opts...)

	// Emit an agent.created event through the event store
	agent := Agent{Slug: "writer", UserID: "u1", Name: "Writer"}
	evt, err := NewEvent("agent-u1-writer", AgentCreated{Agent: agent})
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if err := es.Append(context.Background(), "agent-u1-writer", []fact.Event{evt}); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Verify the projector updated the read model
	got, err := store.GetAgentByUser("u1", "writer")
	if err != nil {
		t.Fatalf("GetAgentByUser: %v", err)
	}
	if got.Name != "Writer" {
		t.Errorf("Name = %q, want Writer", got.Name)
	}
}

func TestEventStore_ConversationFlowEmitsEvents(t *testing.T) {
	store := newMemoryStore()
	projectors := DefaultProjectors(store, store)

	var opts []fact.Option
	for _, p := range projectors {
		opts = append(opts, fact.WithProjector(p))
	}
	es := fact.NewMemoryStore(opts...)

	// Create conversation
	evt, _ := NewEvent("conversation-c1", ConversationCreated{ID: "c1", AgentSlug: "writer", UserID: "u1"})
	es.Append(context.Background(), "conversation-c1", []fact.Event{evt})

	// Append message
	evt, _ = NewEvent("conversation-c1", MessageAppended{ID: "m1", Role: "user", Content: "hello"})
	es.Append(context.Background(), "conversation-c1", []fact.Event{evt})

	// Title
	evt, _ = NewEventWithMeta("conversation-c1", ConversationTitled{Title: "Greeting"}, map[string]string{"user_id": "u1"})
	es.Append(context.Background(), "conversation-c1", []fact.Event{evt})

	// Verify read model
	conv, err := store.GetConversationByUser("u1", "c1")
	if err != nil {
		t.Fatalf("GetConversationByUser: %v", err)
	}
	if conv.Title == nil || *conv.Title != "Greeting" {
		t.Errorf("Title = %v, want Greeting", conv.Title)
	}

	msgs, _ := store.GetMessages("c1")
	if len(msgs) != 1 || msgs[0].Content != "hello" {
		t.Errorf("messages = %+v", msgs)
	}
}

type mockModelLister struct{}

func (m *mockModelLister) ListModels(ctx context.Context) ([]string, error) {
	return []string{"test-model"}, nil
}
