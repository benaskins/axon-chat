package chat

import (
	"context"
	"embed"

	fact "github.com/benaskins/axon-fact"
	tool "github.com/benaskins/axon-tool"
	"github.com/benaskins/axon/sse"
)

// Option configures a Server during construction.
type Option func(*Server)

// WithStore sets the read/write store for the chat server.
func WithStore(store Store) Option {
	return func(s *Server) {
		s.chat.store = store
	}
}

// WithDefaultModel sets the default LLM model name.
func WithDefaultModel(model string) Option {
	return func(s *Server) {
		s.config.DefaultModel = model
		s.chat.defaultModel = model
	}
}

// WithCookieDomain sets the cookie domain for session management.
func WithCookieDomain(domain string) Option {
	return func(s *Server) {
		s.config.CookieDomain = domain
	}
}

// WithAuthLoginURL sets the external auth service login URL.
// The SvelteKit frontend redirects here when the user is not authenticated.
func WithAuthLoginURL(url string) Option {
	return func(s *Server) {
		s.config.AuthLoginURL = url
	}
}

// WithStaticFiles sets the embedded filesystem for the SvelteKit frontend.
func WithStaticFiles(fs *embed.FS) Option {
	return func(s *Server) {
		s.staticFiles = fs
	}
}

// WithEventBus sets the SSE event bus for real-time streaming.
func WithEventBus(bus *sse.EventBus[Event]) Option {
	return func(s *Server) {
		s.chat.eventBus = bus
	}
}

// WithShutdownContext sets the context used for graceful shutdown coordination.
func WithShutdownContext(ctx context.Context) Option {
	return func(s *Server) {
		s.chat.shutdownCtx = ctx
	}
}

// WithModelLister sets the model listing backend.
func WithModelLister(lister ModelLister) Option {
	return func(s *Server) {
		s.ModelLister = lister
	}
}

// WithSearcher sets the web search provider.
func WithSearcher(searcher Searcher) Option {
	return func(s *Server) {
		s.Searcher = searcher
	}
}

// WithToolRouter sets the tool routing provider.
func WithToolRouter(router *ToolRouter) Option {
	return func(s *Server) {
		s.ToolRouter = router
	}
}

// WithTaskRunner sets the async task runner.
func WithTaskRunner(runner TaskRunner) Option {
	return func(s *Server) {
		s.TaskRunner = runner
	}
}

// WithWeather sets the weather provider.
func WithWeather(provider WeatherProvider) Option {
	return func(s *Server) {
		s.Weather = provider
	}
}

// WithPageFetcher sets the page fetcher for URL content extraction.
func WithPageFetcher(fetcher *PageFetcher) Option {
	return func(s *Server) {
		s.PageFetcher = fetcher
	}
}

// WithSearchQualifier sets the search qualifier for query refinement.
func WithSearchQualifier(qualifier *SearchQualifier) Option {
	return func(s *Server) {
		s.SearchQualifier = qualifier
	}
}

// WithMemoryRecaller sets the memory recall provider.
func WithMemoryRecaller(recaller MemoryRecaller) Option {
	return func(s *Server) {
		s.MemoryRecaller = recaller
	}
}

// WithMemoryExtractor sets the memory extractor for idle conversations.
func WithMemoryExtractor(extractor MemoryExtractor) Option {
	return func(s *Server) {
		s.MemoryExtractor = extractor
	}
}

// WithAnalytics sets the analytics event emitter.
func WithAnalytics(analytics AnalyticsEmitter) Option {
	return func(s *Server) {
		s.Analytics = analytics
	}
}

// WithExtraTools registers additional tool definitions.
func WithExtraTools(tools map[string]tool.ToolDef) Option {
	return func(s *Server) {
		s.ExtraTools = tools
	}
}

// WithEventStore overrides the default in-memory event store.
// Use this to provide a durable event store (e.g., Postgres-backed).
// The caller is responsible for registering projectors on the provided store.
func WithEventStore(es fact.EventStore) Option {
	return func(s *Server) {
		s.chat.eventStore = es
	}
}
