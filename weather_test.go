package chat

import (
	"testing"
)

func TestWeatherCodeToDescription(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Clear sky"},
		{1, "Mainly clear"},
		{2, "Partly cloudy"},
		{3, "Overcast"},
		{45, "Foggy"},
		{48, "Foggy"},
		{51, "Drizzle"},
		{55, "Drizzle"},
		{56, "Freezing drizzle"},
		{61, "Rain"},
		{65, "Rain"},
		{66, "Freezing rain"},
		{71, "Snow"},
		{77, "Snow grains"},
		{80, "Rain showers"},
		{85, "Snow showers"},
		{95, "Thunderstorm"},
		{96, "Thunderstorm with hail"},
		{99, "Thunderstorm with hail"},
		{100, "Unknown"},
		{-1, "Unknown"},
		{10, "Unknown"},
	}

	for _, tt := range tests {
		got := weatherCodeToDescription(tt.code)
		if got != tt.want {
			t.Errorf("weatherCodeToDescription(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestParseLocation(t *testing.T) {
	tests := []struct {
		input      string
		wantCity   string
		wantQuals  []string
	}{
		{"Gosford, Australia", "Gosford", []string{"Australia"}},
		{"New York, New York, USA", "New York", []string{"New York", "USA"}},
		{"Tokyo", "Tokyo", nil},
		{"London, , UK", "London", []string{"UK"}},
		{"", "", nil},
	}

	for _, tt := range tests {
		city, quals := parseLocation(tt.input)
		if city != tt.wantCity {
			t.Errorf("parseLocation(%q) city = %q, want %q", tt.input, city, tt.wantCity)
		}
		if len(quals) != len(tt.wantQuals) {
			t.Errorf("parseLocation(%q) qualifiers = %v, want %v", tt.input, quals, tt.wantQuals)
			continue
		}
		for i, q := range quals {
			if q != tt.wantQuals[i] {
				t.Errorf("parseLocation(%q) qualifier[%d] = %q, want %q", tt.input, i, q, tt.wantQuals[i])
			}
		}
	}
}

func TestBestGeoMatch_NoQualifiers(t *testing.T) {
	results := []geoResult{
		{Name: "London", Country: "United Kingdom", Admin1: "England"},
		{Name: "London", Country: "Canada", Admin1: "Ontario"},
	}
	got := bestGeoMatch(results, nil)
	if got.Country != "United Kingdom" {
		t.Errorf("expected first result (United Kingdom), got %s", got.Country)
	}
}

func TestBestGeoMatch_WithMatchingQualifier(t *testing.T) {
	results := []geoResult{
		{Name: "London", Country: "United Kingdom", Admin1: "England"},
		{Name: "London", Country: "Canada", Admin1: "Ontario"},
	}
	got := bestGeoMatch(results, []string{"Canada"})
	if got.Country != "Canada" {
		t.Errorf("expected Canada, got %s", got.Country)
	}
}

func TestBestGeoMatch_Admin1Match(t *testing.T) {
	results := []geoResult{
		{Name: "Springfield", Country: "United States", Admin1: "Illinois"},
		{Name: "Springfield", Country: "United States", Admin1: "Missouri"},
	}
	got := bestGeoMatch(results, []string{"Missouri"})
	if got.Admin1 != "Missouri" {
		t.Errorf("expected Missouri, got %s", got.Admin1)
	}
}

func TestBestGeoMatch_NoQualifierMatch_FallsBackToFirst(t *testing.T) {
	results := []geoResult{
		{Name: "Paris", Country: "France", Admin1: "Ile-de-France"},
		{Name: "Paris", Country: "United States", Admin1: "Texas"},
	}
	got := bestGeoMatch(results, []string{"Germany"})
	if got.Country != "France" {
		t.Errorf("expected fallback to first (France), got %s", got.Country)
	}
}

func TestBestGeoMatch_CaseInsensitive(t *testing.T) {
	results := []geoResult{
		{Name: "Sydney", Country: "Canada", Admin1: "Nova Scotia"},
		{Name: "Sydney", Country: "Australia", Admin1: "New South Wales"},
	}
	got := bestGeoMatch(results, []string{"australia"})
	if got.Country != "Australia" {
		t.Errorf("expected Australia (case insensitive), got %s", got.Country)
	}
}

func TestBestGeoMatch_MultipleQualifiers_HighestScore(t *testing.T) {
	results := []geoResult{
		{Name: "Portland", Country: "United States", Admin1: "Oregon"},
		{Name: "Portland", Country: "United States", Admin1: "Maine"},
	}
	// Both match "United States", but only second matches "Maine"
	got := bestGeoMatch(results, []string{"United States", "Maine"})
	if got.Admin1 != "Maine" {
		t.Errorf("expected Maine (higher score), got %s", got.Admin1)
	}
}
