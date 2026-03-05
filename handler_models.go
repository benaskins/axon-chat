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

	models, err := h.lister.ListModels(r.Context())
	if err != nil {
		slog.Error("failed to list models", "error", err)
		axon.WriteError(w, http.StatusBadGateway, "failed to list models")
		return
	}

	type modelInfo struct {
		Name string `json:"name"`
	}

	result := make([]modelInfo, len(models))
	for i, name := range models {
		result[i] = modelInfo{Name: name}
	}

	axon.WriteJSON(w, http.StatusOK, result)
}
