package ui

import (
	"testing"

	"github.com/nvandessel/go4dot/internal/platform"
)

func TestIsPlatformMatch(t *testing.T) {
	tests := []struct {
		name      string
		platforms []string
		platform  *platform.Platform
		want      bool
	}{
		{
			name:      "exact OS match",
			platforms: []string{"linux"},
			platform:  &platform.Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "OS in list",
			platforms: []string{"darwin", "linux"},
			platform:  &platform.Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "no OS match",
			platforms: []string{"darwin", "windows"},
			platform:  &platform.Platform{OS: "linux"},
			want:      false,
		},
		{
			name:      "all matches any platform",
			platforms: []string{"all"},
			platform:  &platform.Platform{OS: "linux"},
			want:      true,
		},
		{
			name:      "distro match",
			platforms: []string{"fedora"},
			platform:  &platform.Platform{OS: "linux", Distro: "fedora"},
			want:      true,
		},
		{
			name:      "distro in list",
			platforms: []string{"ubuntu", "fedora", "arch"},
			platform:  &platform.Platform{OS: "linux", Distro: "fedora"},
			want:      true,
		},
		{
			name:      "no distro match",
			platforms: []string{"ubuntu", "debian"},
			platform:  &platform.Platform{OS: "linux", Distro: "fedora"},
			want:      false,
		},
		{
			name:      "darwin platform",
			platforms: []string{"darwin", "macos"},
			platform:  &platform.Platform{OS: "darwin"},
			want:      true,
		},
		{
			name:      "empty platforms list",
			platforms: []string{},
			platform:  &platform.Platform{OS: "linux"},
			want:      false,
		},
		{
			name:      "case sensitive - no match",
			platforms: []string{"Linux"},
			platform:  &platform.Platform{OS: "linux"},
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPlatformMatch(tt.platforms, tt.platform)
			if got != tt.want {
				t.Errorf("isPlatformMatch(%v, %+v) = %v, want %v", tt.platforms, tt.platform, got, tt.want)
			}
		})
	}
}
