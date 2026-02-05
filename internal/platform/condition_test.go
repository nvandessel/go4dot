package platform

import (
	"testing"
)

func TestCheckCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition map[string]string
		platform  *Platform
		want      bool
	}{
		{
			name:      "empty condition always true",
			condition: map[string]string{},
			platform:  &Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "nil condition always true",
			condition: nil,
			platform:  &Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "matching os",
			condition: map[string]string{"os": "linux"},
			platform:  &Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "matching platform (alias for os)",
			condition: map[string]string{"platform": "darwin"},
			platform:  &Platform{OS: "darwin"},
			want:      true,
		},
		{
			name:      "non-matching os",
			condition: map[string]string{"os": "darwin"},
			platform:  &Platform{OS: "linux"},
			want:      false,
		},
		{
			name:      "matching distro",
			condition: map[string]string{"distro": "fedora"},
			platform:  &Platform{Distro: "fedora"},
			want:      true,
		},
		{
			name:      "non-matching distro",
			condition: map[string]string{"distro": "ubuntu"},
			platform:  &Platform{Distro: "fedora"},
			want:      false,
		},
		{
			name:      "matching package_manager",
			condition: map[string]string{"package_manager": "dnf"},
			platform:  &Platform{PackageManager: "dnf"},
			want:      true,
		},
		{
			name:      "matching arch",
			condition: map[string]string{"arch": "amd64"},
			platform:  &Platform{Architecture: "amd64"},
			want:      true,
		},
		{
			name:      "matching architecture (alias for arch)",
			condition: map[string]string{"architecture": "arm64"},
			platform:  &Platform{Architecture: "arm64"},
			want:      true,
		},
		{
			name:      "wsl true when is WSL",
			condition: map[string]string{"wsl": "true"},
			platform:  &Platform{IsWSL: true},
			want:      true,
		},
		{
			name:      "wsl true when not WSL",
			condition: map[string]string{"wsl": "true"},
			platform:  &Platform{IsWSL: false},
			want:      false,
		},
		{
			name:      "wsl false when not WSL",
			condition: map[string]string{"wsl": "false"},
			platform:  &Platform{IsWSL: false},
			want:      true,
		},
		{
			name:      "wsl false when is WSL",
			condition: map[string]string{"wsl": "false"},
			platform:  &Platform{IsWSL: true},
			want:      false,
		},
		{
			name:      "multiple conditions all match",
			condition: map[string]string{"os": "linux", "distro": "fedora"},
			platform:  &Platform{OS: "linux", Distro: "fedora"},
			want:      true,
		},
		{
			name:      "multiple conditions one fails",
			condition: map[string]string{"os": "linux", "distro": "ubuntu"},
			platform:  &Platform{OS: "linux", Distro: "fedora"},
			want:      false,
		},
		{
			name:      "comma-separated os values - match",
			condition: map[string]string{"os": "linux,darwin"},
			platform:  &Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "comma-separated os values - match second",
			condition: map[string]string{"os": "linux,darwin"},
			platform:  &Platform{OS: "darwin"},
			want:      true,
		},
		{
			name:      "comma-separated os values - no match",
			condition: map[string]string{"os": "linux,darwin"},
			platform:  &Platform{OS: "windows"},
			want:      false,
		},
		{
			name:      "unknown condition key ignored",
			condition: map[string]string{"unknown_key": "value"},
			platform:  &Platform{OS: "linux"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckCondition(tt.condition, tt.platform)
			if got != tt.want {
				t.Errorf("CheckCondition() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesValue(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		want     bool
	}{
		{
			name:     "exact match",
			actual:   "linux",
			expected: "linux",
			want:     true,
		},
		{
			name:     "no match",
			actual:   "linux",
			expected: "darwin",
			want:     false,
		},
		{
			name:     "comma-separated first match",
			actual:   "linux",
			expected: "linux,darwin",
			want:     true,
		},
		{
			name:     "comma-separated second match",
			actual:   "darwin",
			expected: "linux,darwin",
			want:     true,
		},
		{
			name:     "comma-separated no match",
			actual:   "windows",
			expected: "linux,darwin",
			want:     false,
		},
		{
			name:     "comma-separated with spaces",
			actual:   "darwin",
			expected: "linux, darwin, windows",
			want:     true,
		},
		{
			name:     "empty actual",
			actual:   "",
			expected: "linux",
			want:     false,
		},
		{
			name:     "empty expected",
			actual:   "linux",
			expected: "",
			want:     false,
		},
		{
			name:     "both empty",
			actual:   "",
			expected: "",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesValue(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("matchesValue(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}
