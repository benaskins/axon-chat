package chat

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestImageStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewImageStore(dir)

	data := []byte("fake png data")
	id, err := store.Save(data)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if string(loaded) != string(data) {
		t.Errorf("loaded data mismatch")
	}
}

func TestImageStore_SaveWithID(t *testing.T) {
	dir := t.TempDir()
	store := NewImageStore(dir)

	data := []byte("test image")
	err := store.SaveWithID("my-id", data)
	if err != nil {
		t.Fatalf("SaveWithID failed: %v", err)
	}

	loaded, err := store.Load("my-id")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if string(loaded) != "test image" {
		t.Errorf("data mismatch")
	}
}

func TestImageStore_LoadSize(t *testing.T) {
	dir := t.TempDir()
	store := NewImageStore(dir)

	// Save original
	os.WriteFile(filepath.Join(dir, "test.png"), []byte("original"), 0644)
	// Save thumb variant
	os.WriteFile(filepath.Join(dir, "test_thumb.png"), []byte("thumb"), 0644)

	// Load thumb
	data, err := store.LoadSize("test", "thumb")
	if err != nil {
		t.Fatalf("LoadSize thumb failed: %v", err)
	}
	if string(data) != "thumb" {
		t.Errorf("expected thumb data, got %s", string(data))
	}

	// Load nonexistent variant falls back to original
	data, err = store.LoadSize("test", "lg")
	if err != nil {
		t.Fatalf("LoadSize fallback failed: %v", err)
	}
	if string(data) != "original" {
		t.Errorf("expected fallback to original, got %s", string(data))
	}
}

func TestImageStore_InvalidID(t *testing.T) {
	dir := t.TempDir()
	store := NewImageStore(dir)

	_, err := store.Load("../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal ID")
	}
}

func TestImageHandler_NotFound(t *testing.T) {
	dir := t.TempDir()
	store := NewImageStore(dir)
	handler := &imageHandler{store: store}

	req := httptest.NewRequest(http.MethodGet, "/api/images/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestImageHandler_Success(t *testing.T) {
	dir := t.TempDir()
	store := NewImageStore(dir)
	store.SaveWithID("test-img", []byte("png data"))

	handler := &imageHandler{store: store}
	req := httptest.NewRequest(http.MethodGet, "/api/images/test-img", nil)
	req.SetPathValue("id", "test-img")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected Content-Type image/png, got %s", w.Header().Get("Content-Type"))
	}
}
