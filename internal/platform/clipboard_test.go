package platform

import (
	"testing"
)

func TestDetectDisplayServer(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name:     "wayland via XDG_SESSION_TYPE",
			envVars:  map[string]string{"XDG_SESSION_TYPE": "wayland"},
			expected: "wayland",
		},
		{
			name:     "x11 via XDG_SESSION_TYPE",
			envVars:  map[string]string{"XDG_SESSION_TYPE": "x11"},
			expected: "x11",
		},
		{
			name:     "wayland via WAYLAND_DISPLAY fallback",
			envVars:  map[string]string{"WAYLAND_DISPLAY": "wayland-0"},
			expected: "wayland",
		},
		{
			name:     "x11 via DISPLAY fallback",
			envVars:  map[string]string{"DISPLAY": ":0"},
			expected: "x11",
		},
		{
			name:     "no display server",
			envVars:  map[string]string{},
			expected: "",
		},
		{
			name:     "unknown XDG_SESSION_TYPE falls through",
			envVars:  map[string]string{"XDG_SESSION_TYPE": "tty"},
			expected: "",
		},
		{
			name: "XDG_SESSION_TYPE takes priority over DISPLAY",
			envVars: map[string]string{
				"XDG_SESSION_TYPE": "wayland",
				"DISPLAY":         ":0",
			},
			expected: "wayland",
		},
		{
			name: "WAYLAND_DISPLAY takes priority over DISPLAY",
			envVars: map[string]string{
				"WAYLAND_DISPLAY": "wayland-0",
				"DISPLAY":         ":0",
			},
			expected: "wayland",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars first
			t.Setenv("XDG_SESSION_TYPE", "")
			t.Setenv("WAYLAND_DISPLAY", "")
			t.Setenv("DISPLAY", "")

			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			got := DetectDisplayServer()
			if got != tt.expected {
				t.Errorf("DetectDisplayServer() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetectClipboard(t *testing.T) {
	tests := []struct {
		name          string
		platform      *Platform
		wantCopyCmd   string
		wantPasteCmd  string
	}{
		{
			name: "darwin always has pbcopy/pbpaste",
			platform: &Platform{
				OS: "darwin",
			},
			wantCopyCmd:  "pbcopy",
			wantPasteCmd: "pbpaste",
		},
		{
			name: "linux wayland suggests wl-copy",
			platform: &Platform{
				OS:             "linux",
				DisplayServer:  "wayland",
				PackageManager: "apt",
			},
			wantCopyCmd:  "wl-copy",
			wantPasteCmd: "wl-paste",
		},
		{
			name: "linux x11 suggests xclip",
			platform: &Platform{
				OS:             "linux",
				DisplayServer:  "x11",
				PackageManager: "pacman",
			},
			wantCopyCmd:  "xclip -selection clipboard",
			wantPasteCmd: "xclip -selection clipboard -o",
		},
		{
			name: "linux no display server returns empty",
			platform: &Platform{
				OS:             "linux",
				DisplayServer:  "",
				PackageManager: "dnf",
			},
			wantCopyCmd:  "",
			wantPasteCmd: "",
		},
		{
			name: "windows returns empty",
			platform: &Platform{
				OS: "windows",
			},
			wantCopyCmd:  "",
			wantPasteCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectClipboard(tt.platform)
			if info.CopyCmd != tt.wantCopyCmd {
				t.Errorf("CopyCmd = %q, want %q", info.CopyCmd, tt.wantCopyCmd)
			}
			if info.PasteCmd != tt.wantPasteCmd {
				t.Errorf("PasteCmd = %q, want %q", info.PasteCmd, tt.wantPasteCmd)
			}
		})
	}
}

func TestInstallHint(t *testing.T) {
	tests := []struct {
		name     string
		platform *Platform
		pkg      string
		expected string
	}{
		{
			name:     "apt",
			platform: &Platform{PackageManager: "apt"},
			pkg:      "wl-clipboard",
			expected: "sudo apt install wl-clipboard",
		},
		{
			name:     "dnf",
			platform: &Platform{PackageManager: "dnf"},
			pkg:      "wl-clipboard",
			expected: "sudo dnf install wl-clipboard",
		},
		{
			name:     "pacman",
			platform: &Platform{PackageManager: "pacman"},
			pkg:      "xclip",
			expected: "sudo pacman -S xclip",
		},
		{
			name:     "brew",
			platform: &Platform{PackageManager: "brew"},
			pkg:      "xclip",
			expected: "brew install xclip",
		},
		{
			name:     "unknown falls back to generic",
			platform: &Platform{PackageManager: "unknown"},
			pkg:      "xclip",
			expected: "install xclip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := installHint(tt.platform, tt.pkg)
			if got != tt.expected {
				t.Errorf("installHint() = %q, want %q", got, tt.expected)
			}
		})
	}
}
