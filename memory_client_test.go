package chat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMemoryClient_RecallMemories(t *testing.T) {
	expected := &MemoryRecallResponse{
		Memories: []RecalledMemory{
			{Type: "episodic", Content: "We went hiking", Importance: 0.8, RelevanceScore: 0.9},
		},
		RelationshipContext: &RelationshipContext{Trust: 0.7, Intimacy: 0.5},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/memory/recall" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("agent") != "helper" {
			t.Errorf("expected agent=helper, got %s", r.URL.Query().Get("agent"))
		}
		if r.URL.Query().Get("user") != "user1" {
			t.Errorf("expected user=user1, got %s", r.URL.Query().Get("user"))
		}
		if r.URL.Query().Get("query") != "hiking" {
			t.Errorf("expected query=hiking, got %s", r.URL.Query().Get("query"))
		}
		json.NewEncoder(w).Encode(expected)
	}))
	defer server.Close()

	client := NewMemoryClient(server.URL)
	resp, err := client.RecallMemories(context.Background(), "helper", "user1", "hiking", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Memories) != 1 {
		t.Fatalf("expected 1 memory, got %d", len(resp.Memories))
	}
	if resp.Memories[0].Content != "We went hiking" {
		t.Errorf("unexpected content: %s", resp.Memories[0].Content)
	}
	if resp.RelationshipContext.Trust != 0.7 {
		t.Errorf("expected trust 0.7, got %f", resp.RelationshipContext.Trust)
	}
}

func TestMemoryClient_RecallMemories_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewMemoryClient(server.URL)
	_, err := client.RecallMemories(context.Background(), "helper", "user1", "hiking", 5)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestMemoryClient_ExtractMemories(t *testing.T) {
	var receivedBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/memory/extract" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewMemoryClient(server.URL)
	err := client.ExtractMemories(context.Background(), "conv-123", "helper", "user1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBody["conversation_id"] != "conv-123" {
		t.Errorf("expected conversation_id conv-123, got %s", receivedBody["conversation_id"])
	}
	if receivedBody["agent_slug"] != "helper" {
		t.Errorf("expected agent_slug helper, got %s", receivedBody["agent_slug"])
	}
}
