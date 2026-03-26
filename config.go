package chat

// Config holds configuration for the chat server.
type Config struct {
	DefaultModel string
	CookieDomain string
	AuthLoginURL string

	// InternalAPIKey is required for service-to-service endpoints
	// (InternalMessagesHandler, InternalAgentHandler). Requests must
	// present this value in the X-Internal-API-Key header.
	InternalAPIKey string
}
