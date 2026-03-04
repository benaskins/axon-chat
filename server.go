package chat

import (
	"context"
	"embed"
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
	tool "github.com/benaskins/axon-tool"
	"github.com/benaskins/axon/sse"
)

// Server is the chat service HTTP server.
type Server struct {
	mux         *http.ServeMux
	config      Config
	chat        *chatHandler
	staticFiles *embed.FS

	// Optional dependencies — set before calling Handler().
	Searcher        Searcher
	ToolRouter      *ToolRouter
	TaskRunner      TaskRunner
	Weather         WeatherProvider
	PageFetcher     *PageFetcher
	SearchQualifier *SearchQualifier

	// MemoryRecaller provides memory recall for the recall_memory tool.
	MemoryRecaller MemoryRecaller

	// ExtraTools are additional tool definitions registered by the composition root.
	// They are included in the tool map alongside built-in tools when the agent
	// has the matching skill enabled.
	ExtraTools map[string]tool.ToolDef
}

// NewServer creates a chat server with required dependencies.
func NewServer(
	cfg Config,
	store Store,
	client ChatClient,
	staticFiles *embed.FS,
	eventBus *sse.EventBus[Event],
	shutdownCtx context.Context,
) *Server {
	chat := newChatHandler(cfg.DefaultModel, client, store, shutdownCtx, eventBus)

	return &Server{
		config:      cfg,
		chat:        chat,
		staticFiles: staticFiles,
	}
}

// Handler returns an http.Handler with all chat routes except /health and /metrics.
// Call this after setting optional dependencies.
func (s *Server) Handler(authMiddleware func(http.Handler) http.Handler) http.Handler {
	// Wire optional dependencies into the chat handler
	s.chat.searcher = s.Searcher
	s.chat.toolRouter = s.ToolRouter
	s.chat.taskRunner = s.TaskRunner
	s.chat.weather = s.Weather
	s.chat.pageFetcher = s.PageFetcher
	s.chat.searchQualifier = s.SearchQualifier
	s.chat.memoryRecaller = s.MemoryRecaller
	s.chat.extraTools = s.ExtraTools

	mux := http.NewServeMux()

	auth := authMiddleware

	// Auth check endpoint (returns current user info)
	mux.Handle("GET /api/me", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		axon.WriteJSON(w, http.StatusOK, map[string]string{"user_id": axon.UserID(r.Context()), "username": axon.Username(r.Context())})
	})))

	// Protected API routes
	models := &modelsHandler{lister: s.chat.client.(ModelLister)}

	agentsList := &agentsListHandler{store: s.chat.store}
	agentDetail := &agentDetailHandler{store: s.chat.store}
	agentSave := &agentSaveHandler{store: s.chat.store, taskRunner: s.TaskRunner}
	agentDelete := &agentDeleteHandler{store: s.chat.store}

	convoList := &conversationListHandler{store: s.chat.store}
	convoCreate := &conversationCreateHandler{store: s.chat.store}
	convoGet := &conversationGetHandler{store: s.chat.store}
	convoDelete := &conversationDeleteHandler{store: s.chat.store}

	mux.Handle("/api/chat", auth(s.chat))
	mux.Handle("/api/models", auth(models))
	mux.Handle("GET /api/agents/{slug}", auth(agentDetail))
	mux.Handle("PUT /api/agents/{slug}", auth(agentSave))
	mux.Handle("DELETE /api/agents/{slug}", auth(agentDelete))
	mux.Handle("GET /api/agents/{slug}/conversations", auth(convoList))
	mux.Handle("POST /api/agents/{slug}/conversations", auth(convoCreate))
	mux.Handle("GET /api/conversations/{id}", auth(convoGet))
	mux.Handle("DELETE /api/conversations/{id}", auth(convoDelete))
	mux.Handle("GET /api/agents", auth(agentsList))
	mux.Handle("GET /api/events", auth(&eventsHandler{bus: s.chat.eventBus}))

	// Task list proxy (forwards to task runner)
	taskRunner := s.TaskRunner
	mux.Handle("GET /api/tasks", auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	})))

	// Logout — clears session cookie and redirects to login
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

	// SPA fallback for Svelte frontend
	if s.staticFiles != nil {
		mux.Handle("/", axon.SPAHandler(*s.staticFiles, "static"))
	}

	s.mux = mux
	return mux
}

// UserCreatedHandler returns an http.Handler for the user-created webhook.
func (s *Server) UserCreatedHandler() http.Handler {
	return &userCreatedHandler{store: s.chat.store, defaultModel: s.config.DefaultModel}
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
