package config

import "testing"

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"NvChad Starter", "nvchad-starter"},
		{"  Trim Spaces  ", "trim-spaces"},
		{"Special@#Characters", "special-characters"},
		{"MixedCASE123", "mixedcase123"},
		{"---dashes---", "dashes"},
	}

	for _, tt := range tests {
		result := slugify(tt.input)
		if result != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
