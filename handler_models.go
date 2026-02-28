package chat

import (
	"log/slog"
	"net/http"

	"github.com/benaskins/axon"
)

type modelsHandler struct {
	lister ModelLister
}

func (h *modelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	resp, err := h.lister.List(r.Context())
	if err != nil {
		slog.Error("failed to list models", "error", err)
		axon.WriteError(w, http.StatusBadGateway, "failed to list models")
		return
	}

	type modelInfo struct {
		Name string `json:"name"`
	}

	models := make([]modelInfo, len(resp.Models))
	for i, m := range resp.Models {
		models[i] = modelInfo{Name: m.Name}
	}

	axon.WriteJSON(w, http.StatusOK, models)
}
