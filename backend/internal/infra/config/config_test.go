package config

import (
	"reflect"
	"testing"
)

func TestParseCORSOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"empty defaults to localhost:3000", "", []string{"http://localhost:3000"}},
		{"whitespace defaults to localhost:3000", "   ", []string{"http://localhost:3000"}},
		{"single origin", "https://app.example.com", []string{"https://app.example.com"}},
		{
			"multiple origins are split and trimmed",
			"https://a.example.com, https://b.example.com",
			[]string{"https://a.example.com", "https://b.example.com"},
		},
		{
			"blank entries are skipped",
			"https://a.example.com,,https://b.example.com",
			[]string{"https://a.example.com", "https://b.example.com"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := parseCORSOrigins(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("parseCORSOrigins(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestNewConfigReadsCORSOrigins(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://only.example.com")

	cfg := NewConfig()

	want := []string{"https://only.example.com"}
	if !reflect.DeepEqual(cfg.CORSAllowedOrigins, want) {
		t.Errorf("CORSAllowedOrigins = %v, want %v", cfg.CORSAllowedOrigins, want)
	}
}
