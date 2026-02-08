package status

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestRender_JSON(t *testing.T) {
	syncTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	overview := &Overview{
		Platform: PlatformInfo{
			OS:             "linux",
			Distro:         "fedora",
			DistroVersion:  "41",
			PackageManager: "dnf",
			Architecture:   "amd64",
		},
		DotfilesPath: "/home/user/dotfiles",
		ConfigCount:  2,
		Configs: []ConfigStatus{
			{Name: "zsh", IsCore: true, Status: SyncStatusSynced},
			{Name: "nvim", IsCore: true, Status: SyncStatusDrifted, NewFiles: 1},
		},
		Dependencies: DependencyStatus{
			Installed: 3,
			Missing:   1,
			Total:     4,
		},
		LastSync:    &syncTime,
		Initialized: true,
	}

	output, err := Render(overview, RenderOptions{JSON: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify key fields
	plat, ok := parsed["platform"].(map[string]interface{})
	if !ok {
		t.Fatal("missing platform in JSON output")
	}
	if plat["os"] != "linux" {
		t.Errorf("expected os 'linux', got %v", plat["os"])
	}
	if plat["distro"] != "fedora" {
		t.Errorf("expected distro 'fedora', got %v", plat["distro"])
	}

	configs, ok := parsed["configs"].([]interface{})
	if !ok {
		t.Fatal("missing configs in JSON output")
	}
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}
}

func TestRender_TextNoConfig(t *testing.T) {
	overview := &Overview{
		Platform: PlatformInfo{
			OS:             "darwin",
			PackageManager: "brew",
			Architecture:   "arm64",
		},
		Initialized: false,
	}

	output, err := Render(overview, RenderOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "go4dot status") {
		t.Error("expected header in output")
	}
	if !strings.Contains(output, "No .go4dot.yaml found") {
		t.Error("expected warning about missing config")
	}
	if !strings.Contains(output, "g4d init") {
		t.Error("expected hint about g4d init")
	}
}

func TestRender_TextFullStatus(t *testing.T) {
	syncTime := time.Now().Add(-30 * time.Minute)
	overview := &Overview{
		Platform: PlatformInfo{
			OS:             "linux",
			Distro:         "fedora",
			DistroVersion:  "41",
			PackageManager: "dnf",
			Architecture:   "amd64",
		},
		DotfilesPath: "/home/user/dotfiles",
		ConfigCount:  3,
		Configs: []ConfigStatus{
			{Name: "zsh", IsCore: true, Status: SyncStatusSynced},
			{Name: "nvim", IsCore: true, Status: SyncStatusDrifted, NewFiles: 2, Conflicts: 1},
			{Name: "tmux", IsCore: false, Status: SyncStatusNotInstalled},
		},
		Dependencies: DependencyStatus{
			Installed: 5,
			Missing:   2,
			Total:     7,
		},
		LastSync:    &syncTime,
		Initialized: true,
	}

	output, err := Render(overview, RenderOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Section headers
	checks := []string{
		"go4dot status",
		"Platform",
		"fedora 41",
		"dnf",
		"Dotfiles",
		"/home/user/dotfiles",
		"3 total",
		"Configs",
		"zsh",
		"nvim",
		"tmux",
		"Dependencies",
		"5/7",
		"2 missing",
	}

	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("expected output to contain %q", check)
		}
	}

	// Drift details for nvim
	if !strings.Contains(output, "+2 new") {
		t.Error("expected drift detail '+2 new' for nvim")
	}
	if !strings.Contains(output, "!1 conflicts") {
		t.Error("expected drift detail '!1 conflicts' for nvim")
	}
}

func TestRender_TextWSL(t *testing.T) {
	overview := &Overview{
		Platform: PlatformInfo{
			OS:             "linux",
			Distro:         "ubuntu",
			PackageManager: "apt",
			Architecture:   "amd64",
			IsWSL:          true,
		},
		Initialized: false,
	}

	output, err := Render(overview, RenderOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "WSL") {
		t.Error("expected WSL indicator in output")
	}
}

func TestRender_TextNoDeps(t *testing.T) {
	overview := &Overview{
		Platform: PlatformInfo{
			OS:             "linux",
			Distro:         "arch",
			PackageManager: "pacman",
			Architecture:   "amd64",
		},
		DotfilesPath: "/home/user/dotfiles",
		ConfigCount:  1,
		Configs: []ConfigStatus{
			{Name: "zsh", IsCore: true, Status: SyncStatusSynced},
		},
		Dependencies: DependencyStatus{},
		Initialized:  true,
	}

	output, err := Render(overview, RenderOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "none defined") {
		t.Error("expected 'none defined' when no dependencies")
	}
}

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{"just now", 30 * time.Second, "just now"},
		{"1 minute", 90 * time.Second, "1 minute ago"},
		{"minutes", 15 * time.Minute, "15 minutes ago"},
		{"1 hour", 90 * time.Minute, "1 hour ago"},
		{"hours", 5 * time.Hour, "5 hours ago"},
		{"1 day", 36 * time.Hour, "1 day ago"},
		{"days", 72 * time.Hour, "3 days ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(time.Now().Add(-tt.duration))
			if result != tt.expected {
				t.Errorf("got %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFormatOS(t *testing.T) {
	tests := []struct {
		name     string
		platform PlatformInfo
		expected string
	}{
		{
			name:     "linux with distro and version",
			platform: PlatformInfo{OS: "linux", Distro: "fedora", DistroVersion: "41"},
			expected: "fedora 41",
		},
		{
			name:     "linux with distro only",
			platform: PlatformInfo{OS: "linux", Distro: "arch"},
			expected: "arch",
		},
		{
			name:     "darwin no distro",
			platform: PlatformInfo{OS: "darwin"},
			expected: "darwin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatOS(tt.platform)
			if result != tt.expected {
				t.Errorf("got %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestCountStatuses(t *testing.T) {
	configs := []ConfigStatus{
		{Status: SyncStatusSynced},
		{Status: SyncStatusSynced},
		{Status: SyncStatusDrifted},
		{Status: SyncStatusNotInstalled},
		{Status: SyncStatusNotInstalled},
		{Status: SyncStatusNotInstalled},
	}

	synced, drifted, notInstalled := countStatuses(configs)
	if synced != 2 {
		t.Errorf("expected 2 synced, got %d", synced)
	}
	if drifted != 1 {
		t.Errorf("expected 1 drifted, got %d", drifted)
	}
	if notInstalled != 3 {
		t.Errorf("expected 3 not installed, got %d", notInstalled)
	}
}

func TestDriftDetails(t *testing.T) {
	tests := []struct {
		name     string
		cs       ConfigStatus
		expected string
	}{
		{
			name:     "no drift details",
			cs:       ConfigStatus{},
			expected: "",
		},
		{
			name:     "new files only",
			cs:       ConfigStatus{NewFiles: 3},
			expected: "(+3 new)",
		},
		{
			name:     "multiple details",
			cs:       ConfigStatus{NewFiles: 1, MissingFiles: 2, Conflicts: 3},
			expected: "(+1 new, -2 missing, !3 conflicts)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := driftDetails(tt.cs)
			if result != tt.expected {
				t.Errorf("got %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestRender_TextVersionMismatch(t *testing.T) {
	overview := &Overview{
		Platform: PlatformInfo{
			OS:             "linux",
			Distro:         "fedora",
			PackageManager: "dnf",
			Architecture:   "amd64",
		},
		DotfilesPath: "/home/user/dotfiles",
		ConfigCount:  1,
		Configs: []ConfigStatus{
			{Name: "zsh", IsCore: true, Status: SyncStatusSynced},
		},
		Dependencies: DependencyStatus{
			Installed:      2,
			VersionMissing: 1,
			Total:          3,
		},
		Initialized: true,
	}

	output, err := Render(overview, RenderOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "1 version mismatch") {
		t.Error("expected 'version mismatch' in output")
	}
}
