package chat

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"time"

	"github.com/benaskins/axon"
	fact "github.com/benaskins/axon-fact"
	loop "github.com/benaskins/axon-loop"
	tool "github.com/benaskins/axon-tool"
)

// Server is the chat service HTTP server.
type Server struct {
	mux         *http.ServeMux
	config      Config
	chat        *chatHandler
	staticFiles *embed.FS

	// Optional dependencies — set before calling Handler().
	ModelLister     ModelLister
	Searcher        Searcher
	ToolRouter      *ToolRouter
	TaskRunner      TaskRunner
	Weather         WeatherProvider
	PageFetcher     *PageFetcher
	SearchQualifier *SearchQualifier

	// MemoryRecaller provides memory recall for the recall_memory tool.
	MemoryRecaller MemoryRecaller

	// MemoryExtractor triggers memory extraction for idle conversations.
	MemoryExtractor MemoryExtractor

	// Analytics emits analytics events to the analytics service.
	Analytics AnalyticsEmitter

	// ExtraTools are additional tool definitions registered by the composition root.
	// They are included in the tool map alongside built-in tools when the agent
	// has the matching skill enabled.
	ExtraTools map[string]tool.ToolDef
}

// ModelLister lists available models from the LLM backend.
type ModelLister interface {
	ListModels(ctx context.Context) ([]string, error)
}

// NewServer creates a chat server. The LLM client is the only required
// dependency; everything else is configured via Option functions.
func NewServer(llm loop.LLMClient, opts ...Option) *Server {
	srv := &Server{
		chat: newChatHandler("", llm, nil, context.Background(), nil),
	}
	for _, opt := range opts {
		opt(srv)
	}

	// Default to in-memory event store with projectors when a store is
	// provided but no explicit event store was configured.
	if srv.chat.store != nil && srv.chat.eventStore == nil {
		projectors := DefaultProjectors(srv.chat.store, srv.chat.store)
		var factOpts []fact.Option
		for _, p := range projectors {
			factOpts = append(factOpts, fact.WithProjector(p))
		}
		srv.chat.eventStore = fact.NewMemoryStore(factOpts...)
	}

	return srv
}

// Handler returns an http.Handler with all API routes.
// Auth is not applied here — the composition root wraps with auth middleware.
// Call this after setting optional dependencies.
func (s *Server) Handler() http.Handler {
	s.wireDependencies()

	mux := http.NewServeMux()

	// Auth check endpoint (returns current user info)
	mux.Handle("GET /api/me", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		axon.WriteJSON(w, http.StatusOK, map[string]string{"user_id": axon.UserID(r.Context()), "username": axon.Username(r.Context())})
	}))

	// Models endpoint
	if s.ModelLister != nil {
		mux.Handle("/api/models", &modelsHandler{lister: s.ModelLister})
	} else {
		mux.Handle("/api/models", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			axon.WriteError(w, http.StatusNotImplemented, "model listing not supported")
		}))
	}

	agentsList := &agentsListHandler{store: s.chat.store}
	agentDetail := &agentDetailHandler{store: s.chat.store}
	agentSave := &agentSaveHandler{store: s.chat.store, eventStore: s.chat.eventStore, taskRunner: s.TaskRunner}
	agentDelete := &agentDeleteHandler{store: s.chat.store, eventStore: s.chat.eventStore}

	convoList := &conversationListHandler{store: s.chat.store}
	convoCreate := &conversationCreateHandler{store: s.chat.store, eventStore: s.chat.eventStore}
	convoGet := &conversationGetHandler{store: s.chat.store}
	convoDelete := &conversationDeleteHandler{store: s.chat.store, eventStore: s.chat.eventStore}

	mux.Handle("/api/chat/sync", &syncChatHandler{chat: s.chat})
	mux.Handle("/api/chat", s.chat)
	mux.Handle("GET /api/agents/{slug}", agentDetail)
	mux.Handle("PUT /api/agents/{slug}", agentSave)
	mux.Handle("DELETE /api/agents/{slug}", agentDelete)
	mux.Handle("GET /api/agents/{slug}/conversations", convoList)
	mux.Handle("POST /api/agents/{slug}/conversations", convoCreate)
	mux.Handle("GET /api/conversations/{id}", convoGet)
	mux.Handle("DELETE /api/conversations/{id}", convoDelete)
	mux.Handle("GET /api/agents", agentsList)
	mux.Handle("GET /api/events", &eventsHandler{bus: s.chat.eventBus})

	// Task list proxy (forwards to task runner)
	taskRunner := s.TaskRunner
	mux.Handle("GET /api/tasks", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if taskRunner == nil {
			axon.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "task runner not available"})
			return
		}
		agent := r.URL.Query().Get("agent")
		if agent == "" {
			axon.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "agent query parameter is required"})
			return
		}
		tasks, err := taskRunner.ListTasks(r.Context(), agent, 50, 0)
		if err != nil {
			slog.Error("failed to list tasks", "agent", agent, "error", err)
			axon.WriteJSON(w, http.StatusBadGateway, map[string]string{"error": "failed to fetch tasks"})
			return
		}
		axon.WriteJSON(w, http.StatusOK, tasks)
	}))

	// Logout — clears session cookie
	mux.HandleFunc("POST /api/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			Domain:   s.config.CookieDomain,
			Secure:   true,
			HttpOnly: true,
		})
		axon.WriteJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
	})

	s.mux = mux
	return mux
}

// SPAHandler returns an http.Handler that serves the embedded SvelteKit
// frontend with client-side routing fallback. Mount without auth.
func (s *Server) SPAHandler() http.Handler {
	if s.staticFiles == nil {
		return http.NotFoundHandler()
	}
	spa := axon.SPAHandler(*s.staticFiles, "static", axon.WithStaticPrefix("/_app/"))
	if s.config.AuthLoginURL != "" {
		spa = withAuthConfig(spa, *s.staticFiles, s.config.AuthLoginURL)
	}
	return spa
}

// wireDependencies connects optional dependencies to the chat handler.
func (s *Server) wireDependencies() {
	s.chat.searcher = s.Searcher
	s.chat.toolRouter = s.ToolRouter
	s.chat.taskRunner = s.TaskRunner
	s.chat.weather = s.Weather
	s.chat.pageFetcher = s.PageFetcher
	s.chat.searchQualifier = s.SearchQualifier
	s.chat.memoryRecaller = s.MemoryRecaller
	if s.MemoryExtractor != nil {
		s.chat.idleExtractor = NewIdleExtractor(s.chat.shutdownCtx, s.MemoryExtractor, 1*time.Hour)
	}
	s.chat.extraTools = s.ExtraTools
	if s.Analytics != nil {
		s.chat.analytics = s.Analytics
	} else {
		s.chat.analytics = NoopAnalytics{}
	}
}

// UserCreatedHandler returns an http.Handler for the user-created webhook.
func (s *Server) UserCreatedHandler() http.Handler {
	return &userCreatedHandler{eventStore: s.chat.eventStore, defaultModel: s.config.DefaultModel}
}

// InternalMessagesHandler returns an http.Handler for fetching conversation messages
// without auth, for internal service-to-service calls.
func (s *Server) InternalMessagesHandler() http.Handler {
	return &internalMessagesHandler{store: s.chat.store}
}

// InternalAgentHandler returns an http.Handler for fetching agent info by slug
// without auth, for internal service-to-service calls.
func (s *Server) InternalAgentHandler() http.Handler {
	return &internalAgentHandler{store: s.chat.store}
}

// WaitForBackgroundTasks blocks until all background tasks complete.
func (s *Server) WaitForBackgroundTasks() {
	s.chat.WaitForBackgroundTasks()
}

// withAuthConfig wraps an SPA handler to inject the auth login URL into index.html.
// It reads the original index.html once at startup and injects a <script> tag
// that sets window.__AUTH_URL__ before any other scripts run.
func withAuthConfig(next http.Handler, files embed.FS, authLoginURL string) http.Handler {
	staticSub, err := fs.Sub(files, "static")
	if err != nil {
		slog.Error("withAuthConfig: failed to create sub filesystem", "error", err)
		return next
	}

	original, err := fs.ReadFile(staticSub, "index.html")
	if err != nil {
		slog.Error("withAuthConfig: failed to read index.html", "error", err)
		return next
	}

	configScript := []byte(`<script>window.__AUTH_URL__="` + authLoginURL + `";</script>`)
	injected := bytes.Replace(original, []byte("<head>"), append([]byte("<head>"), configScript...), 1)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve static assets directly via the SPA handler
		if _, err := fs.Stat(staticSub, r.URL.Path[1:]); err == nil && r.URL.Path != "/" {
			next.ServeHTTP(w, r)
			return
		}
		// All other paths are SPA fallback — serve injected index.html
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(injected)
	})
}
