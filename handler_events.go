package chat

import (
	"net/http"

	"github.com/benaskins/axon"
	"github.com/benaskins/axon/sse"

	"github.com/google/uuid"
)

type eventsHandler struct {
	bus *sse.EventBus[Event]
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		axon.WriteError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	clientID := uuid.New().String()
	ch := h.bus.Subscribe(clientID)
	defer h.bus.Unsubscribe(clientID)

	sse.SetSSEHeaders(w)
	flusher.Flush()

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			sse.SendEvent(w, flusher, ev)
		case <-r.Context().Done():
			return
		}
	}
}
