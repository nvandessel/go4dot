package deps

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

func TestInstallManualSkipped(t *testing.T) {
	// Config with only a manual dep that is missing; no auto-installable deps missing
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"}, // exists everywhere
			},
			Core: []config.DependencyItem{
				{Name: "fake-manual-dep-xyz", Binary: "fake-manual-dep-xyz", Manual: true},
			},
		},
	}

	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	var progressMessages []string
	opts := InstallOptions{
		OnlyMissing: true,
		DryRun:      true,
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	result, err := Install(cfg, p, opts)
	if err != nil {
		t.Fatalf("Install() failed: %v", err)
	}

	// The manual dep should be in ManualSkipped
	if len(result.ManualSkipped) != 1 {
		t.Errorf("len(ManualSkipped) = %d, want 1", len(result.ManualSkipped))
	}
	if len(result.ManualSkipped) > 0 && result.ManualSkipped[0].Name != "fake-manual-dep-xyz" {
		t.Errorf("ManualSkipped[0].Name = %q, want %q", result.ManualSkipped[0].Name, "fake-manual-dep-xyz")
	}

	// Should not be in Installed or Failed
	if len(result.Installed) != 0 {
		t.Errorf("len(Installed) = %d, want 0 (no auto-installable deps missing)", len(result.Installed))
	}
	if len(result.Failed) != 0 {
		t.Errorf("len(Failed) = %d, want 0", len(result.Failed))
	}

	// Progress should mention skipping manual dep
	foundManualMsg := false
	for _, msg := range progressMessages {
		if strings.Contains(msg, "Skipping manual dependency") && strings.Contains(msg, "fake-manual-dep-xyz") {
			foundManualMsg = true
			break
		}
	}
	if !foundManualMsg {
		t.Errorf("Expected progress message about skipping manual dep, got: %v", progressMessages)
	}
}

func TestInstallManualSkippedNoProgressFunc(t *testing.T) {
	// Ensures nil ProgressFunc doesn't panic when manual deps exist
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"},
			},
			Core: []config.DependencyItem{
				{Name: "fake-manual-dep-xyz", Binary: "fake-manual-dep-xyz", Manual: true},
			},
		},
	}

	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	opts := InstallOptions{
		OnlyMissing: true,
		DryRun:      true,
	}

	result, err := Install(cfg, p, opts)
	if err != nil {
		t.Fatalf("Install() failed: %v", err)
	}

	if len(result.ManualSkipped) != 1 {
		t.Errorf("len(ManualSkipped) = %d, want 1", len(result.ManualSkipped))
	}
}

func TestInstallDryRunWithMissingDeps(t *testing.T) {
	// Config with both a missing auto-installable dep and a manual dep
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"},
			},
			Core: []config.DependencyItem{
				{Name: "fake-auto-dep-xyz", Binary: "fake-auto-dep-xyz"},
				{Name: "fake-manual-dep-xyz", Binary: "fake-manual-dep-xyz", Manual: true},
			},
		},
	}

	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	var progressMessages []string
	opts := InstallOptions{
		OnlyMissing: true,
		DryRun:      true,
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	result, err := Install(cfg, p, opts)
	if err != nil {
		t.Fatalf("Install() failed: %v", err)
	}

	// Manual dep should be skipped
	if len(result.ManualSkipped) != 1 {
		t.Errorf("len(ManualSkipped) = %d, want 1", len(result.ManualSkipped))
	}

	// Auto dep should be "installed" in dry-run mode
	if len(result.Installed) != 1 {
		t.Errorf("len(Installed) = %d, want 1 (dry-run installs missing auto deps)", len(result.Installed))
	}

	// Progress should include both a manual skip message and an install message
	hasManualMsg := false
	hasInstallMsg := false
	for _, msg := range progressMessages {
		if strings.Contains(msg, "Skipping manual dependency") {
			hasManualMsg = true
		}
		if strings.Contains(msg, "Installing") {
			hasInstallMsg = true
		}
	}
	if !hasManualMsg {
		t.Error("Expected progress message about skipping manual dep")
	}
	if !hasInstallMsg {
		t.Error("Expected progress message about installing")
	}
}

func TestInstallAllInstalledNoWork(t *testing.T) {
	// All deps are installed; nothing to do
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"},
			},
		},
	}

	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	result, err := Install(cfg, p, InstallOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Install() failed: %v", err)
	}

	if len(result.Installed) != 0 {
		t.Errorf("len(Installed) = %d, want 0", len(result.Installed))
	}
	if len(result.ManualSkipped) != 0 {
		t.Errorf("len(ManualSkipped) = %d, want 0", len(result.ManualSkipped))
	}
}

func TestGetPackageNameForPlatform(t *testing.T) {
	tests := []struct {
		name    string
		dep     config.DependencyItem
		manager string
		want    string
	}{
		{
			name:    "no package map",
			dep:     config.DependencyItem{Name: "curl"},
			manager: "apt",
			want:    "",
		},
		{
			name: "package map with matching manager",
			dep: config.DependencyItem{
				Name:    "fd",
				Package: map[string]string{"apt": "fd-find", "dnf": "fd-find"},
			},
			manager: "apt",
			want:    "fd-find",
		},
		{
			name: "package map without matching manager",
			dep: config.DependencyItem{
				Name:    "fd",
				Package: map[string]string{"apt": "fd-find"},
			},
			manager: "brew",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPackageNameForPlatform(tt.dep, tt.manager)
			if got != tt.want {
				t.Errorf("getPackageNameForPlatform() = %q, want %q", got, tt.want)
			}
		})
	}
}
