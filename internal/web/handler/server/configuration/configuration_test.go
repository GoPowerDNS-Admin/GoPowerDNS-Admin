package configuration

import (
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "test",
			substr: "test",
			want:   true,
		},
		{
			name:   "contains substring",
			s:      "testing",
			substr: "test",
			want:   true,
		},
		{
			name:   "case insensitive match",
			s:      "Testing",
			substr: "test",
			want:   true,
		},
		{
			name:   "not contained",
			s:      "hello",
			substr: "world",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "test",
			substr: "",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "all uppercase",
			input: "HELLO",
			want:  "hello",
		},
		{
			name:  "mixed case",
			input: "HeLLo",
			want:  "hello",
		},
		{
			name:  "all lowercase",
			input: "hello",
			want:  "hello",
		},
		{
			name:  "with numbers",
			input: "Hello123",
			want:  "hello123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toLower(tt.input); got != tt.want {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIndexIgnoreCase(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   int
	}{
		{
			name:   "found at start",
			s:      "Hello World",
			substr: "hello",
			want:   0,
		},
		{
			name:   "found in middle",
			s:      "Hello World",
			substr: "world",
			want:   6,
		},
		{
			name:   "not found",
			s:      "Hello World",
			substr: "test",
			want:   -1,
		},
		{
			name:   "case insensitive",
			s:      "PowerDNS Server",
			substr: "powerdns",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := indexIgnoreCase(tt.s, tt.substr); got != tt.want {
				t.Errorf("indexIgnoreCase(%q, %q) = %d, want %d", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}
