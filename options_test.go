package chat

import (
	"context"
	"embed"
	"testing"

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

type mockModelLister struct{}

func (m *mockModelLister) ListModels(ctx context.Context) ([]string, error) {
	return []string{"test-model"}, nil
}
