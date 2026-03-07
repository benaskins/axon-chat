package chat

import (
	"sort"
	"testing"

	tool "github.com/benaskins/axon-tool"
)

func TestDescribeToolUse(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		args     map[string]any
		want     string
	}{
		{
			name:     "web_search with query",
			toolName: "web_search",
			args:     map[string]any{"query": "golang testing"},
			want:     "\n\n*Searching for \"golang testing\"...*\n\n",
		},
		{
			name:     "web_search without query",
			toolName: "web_search",
			args:     map[string]any{},
			want:     "\n\n*Searching the web...*\n\n",
		},
		{
			name:     "web_search with non-string query",
			toolName: "web_search",
			args:     map[string]any{"query": 42},
			want:     "\n\n*Searching the web...*\n\n",
		},
		{
			name:     "fetch_page with url",
			toolName: "fetch_page",
			args:     map[string]any{"url": "https://example.com"},
			want:     "\n\n*Reading https://example.com...*\n\n",
		},
		{
			name:     "fetch_page without url",
			toolName: "fetch_page",
			args:     map[string]any{},
			want:     "\n\n*Reading a web page...*\n\n",
		},
		{
			name:     "check_weather with location",
			toolName: "check_weather",
			args:     map[string]any{"location": "Tokyo"},
			want:     "\n\n*Checking weather for Tokyo...*\n\n",
		},
		{
			name:     "check_weather without location",
			toolName: "check_weather",
			args:     map[string]any{},
			want:     "\n\n*Checking the weather...*\n\n",
		},
		{
			name:     "current_time",
			toolName: "current_time",
			args:     nil,
			want:     "\n\n*Checking the time...*\n\n",
		},
		{
			name:     "use_claude",
			toolName: "use_claude",
			args:     nil,
			want:     "\n\n*Submitting a code change...*\n\n",
		},
		{
			name:     "unknown tool",
			toolName: "take_photo",
			args:     nil,
			want:     "\n\n*Using take_photo...*\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := describeToolUse(tt.toolName, tt.args)
			if got != tt.want {
				t.Errorf("describeToolUse(%q, %v) = %q, want %q", tt.toolName, tt.args, got, tt.want)
			}
		})
	}
}

func TestDayOrNight(t *testing.T) {
	if got := dayOrNight(true); got != "Daytime" {
		t.Errorf("dayOrNight(true) = %q, want %q", got, "Daytime")
	}
	if got := dayOrNight(false); got != "Nighttime" {
		t.Errorf("dayOrNight(false) = %q, want %q", got, "Nighttime")
	}
}

func TestToolDefsFromMap(t *testing.T) {
	m := map[string]tool.ToolDef{
		"a": {Name: "a", Description: "tool a"},
		"b": {Name: "b", Description: "tool b"},
	}

	defs := toolDefsFromMap(m)
	if len(defs) != 2 {
		t.Fatalf("expected 2 defs, got %d", len(defs))
	}

	// Sort for deterministic comparison
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	if defs[0].Name != "a" || defs[1].Name != "b" {
		t.Errorf("expected names [a, b], got [%s, %s]", defs[0].Name, defs[1].Name)
	}
}

func TestToolDefsFromMap_Empty(t *testing.T) {
	defs := toolDefsFromMap(map[string]tool.ToolDef{})
	if len(defs) != 0 {
		t.Errorf("expected 0 defs, got %d", len(defs))
	}
}

func TestFilterToolNames(t *testing.T) {
	all := map[string]tool.ToolDef{
		"web_search":    {Name: "web_search"},
		"check_weather": {Name: "check_weather"},
		"use_claude":    {Name: "use_claude"},
	}

	result := filterToolNames(all, []string{"web_search", "use_claude"})
	if len(result) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result))
	}
	if _, ok := result["web_search"]; !ok {
		t.Error("expected web_search in result")
	}
	if _, ok := result["use_claude"]; !ok {
		t.Error("expected use_claude in result")
	}
	if _, ok := result["check_weather"]; ok {
		t.Error("check_weather should not be in result")
	}
}

func TestFilterToolNames_NoMatch(t *testing.T) {
	all := map[string]tool.ToolDef{
		"web_search": {Name: "web_search"},
	}

	result := filterToolNames(all, []string{"nonexistent"})
	if len(result) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result))
	}
}

func TestFilterToolNames_EmptyRouted(t *testing.T) {
	all := map[string]tool.ToolDef{
		"web_search": {Name: "web_search"},
	}

	result := filterToolNames(all, []string{})
	if len(result) != 0 {
		t.Errorf("expected 0 tools, got %d", len(result))
	}
}
