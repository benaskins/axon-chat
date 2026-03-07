package chat

import (
	"sort"
	"testing"
)

func TestParseToolResponse_ValidJSON(t *testing.T) {
	available := map[string]bool{
		"web_search":    true,
		"check_weather": true,
		"use_claude":    true,
	}

	response := `{"tools": ["web_search", "check_weather"]}`
	got := parseToolResponse(response, available)

	sort.Strings(got)
	if len(got) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(got))
	}
	if got[0] != "check_weather" || got[1] != "web_search" {
		t.Errorf("got %v, want [check_weather, web_search]", got)
	}
}

func TestParseToolResponse_EmptyTools(t *testing.T) {
	available := map[string]bool{"web_search": true}

	response := `{"tools": []}`
	got := parseToolResponse(response, available)
	if len(got) != 0 {
		t.Errorf("expected 0 tools, got %v", got)
	}
}

func TestParseToolResponse_FiltersUnavailable(t *testing.T) {
	available := map[string]bool{"web_search": true}

	response := `{"tools": ["web_search", "nonexistent"]}`
	got := parseToolResponse(response, available)
	if len(got) != 1 || got[0] != "web_search" {
		t.Errorf("expected [web_search], got %v", got)
	}
}

func TestParseToolResponse_DeduplicatesTools(t *testing.T) {
	available := map[string]bool{"web_search": true}

	response := `{"tools": ["web_search", "web_search"]}`
	got := parseToolResponse(response, available)
	if len(got) != 1 {
		t.Errorf("expected 1 tool (deduplicated), got %v", got)
	}
}

func TestParseToolResponse_MalformedJSON_FallsBackToScan(t *testing.T) {
	available := map[string]bool{
		"web_search": true,
		"use_claude": true,
	}

	// Not valid JSON but mentions a tool name
	response := `I think you should use web_search for this`
	got := parseToolResponse(response, available)
	if len(got) != 1 || got[0] != "web_search" {
		t.Errorf("expected [web_search] via fallback scan, got %v", got)
	}
}

func TestScanForToolNames_MatchesKnownTools(t *testing.T) {
	available := map[string]bool{
		"web_search":    true,
		"check_weather": true,
		"use_claude":    true,
	}

	response := `The user wants to check_weather and also web_search for info`
	got := scanForToolNames(response, available)

	sort.Strings(got)
	if len(got) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(got))
	}
	if got[0] != "check_weather" || got[1] != "web_search" {
		t.Errorf("got %v, want [check_weather, web_search]", got)
	}
}

func TestScanForToolNames_NoMatches(t *testing.T) {
	available := map[string]bool{
		"web_search": true,
		"use_claude": true,
	}

	response := `just a normal conversation, nothing special`
	got := scanForToolNames(response, available)
	if len(got) != 0 {
		t.Errorf("expected 0 tools, got %v", got)
	}
}

func TestScanForToolNames_CaseInsensitive(t *testing.T) {
	available := map[string]bool{"web_search": true}

	response := `Maybe WEB_SEARCH would help here`
	got := scanForToolNames(response, available)
	if len(got) != 1 || got[0] != "web_search" {
		t.Errorf("expected [web_search], got %v", got)
	}
}
