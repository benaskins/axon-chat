package chat

import (
	"net/http"

	"github.com/benaskins/axon"
)

// galleryListHandler serves GET /api/agents/{slug}/gallery
type galleryListHandler struct {
	store Store
}

func (h *galleryListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteError(w, http.StatusBadRequest, "slug required")
		return
	}

	images, err := h.store.ListGalleryImagesByUser(userID, slug)
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to list images")
		return
	}

	if images == nil {
		images = []GalleryImage{}
	}

	// Transform to response format with URLs
	type imageResponse struct {
		ID             string  `json:"id"`
		URL            string  `json:"url"`
		ThumbnailURL   string  `json:"thumbnail_url"`
		Prompt         string  `json:"prompt"`
		Model          string  `json:"model"`
		ConversationID *string `json:"conversation_id"`
		IsBase         bool    `json:"is_base"`
		NSFWDetected   bool    `json:"nsfw_detected"`
		CreatedAt      string  `json:"created_at"`
	}

	response := make([]imageResponse, len(images))
	for i, img := range images {
		response[i] = imageResponse{
			ID:             img.ID,
			URL:            "/api/images/" + img.ID,
			ThumbnailURL:   "/api/images/" + img.ID + "?size=thumb",
			Prompt:         img.Prompt,
			Model:          img.Model,
			ConversationID: img.ConversationID,
			IsBase:         img.IsBase,
			NSFWDetected:   img.NSFWDetected,
			CreatedAt:      img.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	axon.WriteJSON(w, http.StatusOK, map[string]any{"images": response})
}

// getBaseImageHandler serves GET /api/agents/{slug}/gallery/base
type getBaseImageHandler struct {
	store Store
}

func (h *getBaseImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	slug := r.PathValue("slug")
	if slug == "" {
		axon.WriteError(w, http.StatusBadRequest, "slug required")
		return
	}

	img, err := h.store.GetBaseImageByUser(userID, slug)
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to get base image")
		return
	}

	if img == nil {
		axon.WriteError(w, http.StatusNotFound, "no base image set")
		return
	}

	axon.WriteJSON(w, http.StatusOK, map[string]any{
		"id":      img.ID,
		"url":     "/api/images/" + img.ID,
		"is_base": true,
	})
}

// setBaseImageHandler serves PUT /api/agents/{slug}/gallery/{id}/base
type setBaseImageHandler struct {
	store Store
}

func (h *setBaseImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		axon.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := axon.UserID(r.Context())
	slug := r.PathValue("slug")
	imageID := r.PathValue("id")

	if slug == "" || imageID == "" {
		axon.WriteError(w, http.StatusBadRequest, "slug and id required")
		return
	}

	// Verify image belongs to this user's agent
	img, err := h.store.GetGalleryImage(imageID)
	if err != nil || img.AgentSlug != slug || img.UserID != userID {
		axon.WriteError(w, http.StatusNotFound, "image not found")
		return
	}

	if err := h.store.SetBaseImage(userID, slug, imageID); err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to set base image")
		return
	}

	w.WriteHeader(http.StatusOK)
}
