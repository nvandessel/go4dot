package status

import (
	"fmt"
	"testing"
	"time"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
)

func newTestGatherer(p *platform.Platform, cfg *config.Config, configPath string, st *state.State, drift *stow.DriftSummary, depResult *deps.CheckResult) *Gatherer {
	return &Gatherer{
		PlatformDetector: func() (*platform.Platform, error) {
			return p, nil
		},
		ConfigLoader: func() (*config.Config, string, error) {
			if cfg == nil {
				return nil, "", fmt.Errorf("config not found")
			}
			return cfg, configPath, nil
		},
		StateLoader: func() (*state.State, error) {
			return st, nil
		},
		DriftChecker: func(_ *config.Config, _ string) (*stow.DriftSummary, error) {
			return drift, nil
		},
		DepsChecker: func(_ *config.Config, _ *platform.Platform) (*deps.CheckResult, error) {
			return depResult, nil
		},
	}
}

func TestGather_NoConfig(t *testing.T) {
	p := &platform.Platform{
		OS:             "linux",
		Distro:         "fedora",
		DistroVersion:  "41",
		PackageManager: "dnf",
		Architecture:   "amd64",
	}

	g := newTestGatherer(p, nil, "", nil, nil, nil)
	overview, err := g.Gather(GatherOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.Initialized {
		t.Error("expected Initialized to be false when no config is found")
	}
	if overview.Platform.OS != "linux" {
		t.Errorf("expected OS 'linux', got %q", overview.Platform.OS)
	}
	if overview.Platform.Distro != "fedora" {
		t.Errorf("expected Distro 'fedora', got %q", overview.Platform.Distro)
	}
}

func TestGather_PlatformError(t *testing.T) {
	g := &Gatherer{
		PlatformDetector: func() (*platform.Platform, error) {
			return nil, fmt.Errorf("cannot detect platform")
		},
	}
	_, err := g.Gather(GatherOptions{})
	if err == nil {
		t.Fatal("expected error for platform detection failure")
	}
}

func TestGather_FullStatus(t *testing.T) {
	p := &platform.Platform{
		OS:             "darwin",
		PackageManager: "brew",
		Architecture:   "arm64",
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "zsh", Path: "zsh"},
				{Name: "nvim", Path: "nvim"},
			},
			Optional: []config.ConfigItem{
				{Name: "tmux", Path: "tmux"},
			},
		},
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{{Name: "stow"}},
			Core:     []config.DependencyItem{{Name: "nvim"}},
			Optional: []config.DependencyItem{{Name: "tmux"}},
		},
	}

	syncTime := time.Now().Add(-2 * time.Hour)
	st := &state.State{
		LastUpdate: syncTime,
		Configs: []state.ConfigState{
			{Name: "zsh"},
			{Name: "nvim"},
		},
	}

	drift := &stow.DriftSummary{
		TotalConfigs:   2,
		DriftedConfigs: 1,
		Results: []stow.DriftResult{
			{ConfigName: "zsh", HasDrift: false},
			{ConfigName: "nvim", HasDrift: true, NewFiles: []string{"init.lua"}, ConflictFiles: []string{".config/nvim/lazy-lock.json"}},
		},
	}

	depResult := &deps.CheckResult{
		Critical: []deps.DependencyCheck{
			{Item: config.DependencyItem{Name: "stow"}, Status: deps.StatusInstalled},
		},
		Core: []deps.DependencyCheck{
			{Item: config.DependencyItem{Name: "nvim"}, Status: deps.StatusInstalled},
		},
		Optional: []deps.DependencyCheck{
			{Item: config.DependencyItem{Name: "tmux"}, Status: deps.StatusMissing},
		},
	}

	g := newTestGatherer(p, cfg, "/home/user/dotfiles/.go4dot.yaml", st, drift, depResult)
	overview, err := g.Gather(GatherOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !overview.Initialized {
		t.Error("expected Initialized to be true")
	}
	if overview.DotfilesPath != "/home/user/dotfiles" {
		t.Errorf("expected DotfilesPath '/home/user/dotfiles', got %q", overview.DotfilesPath)
	}
	if overview.ConfigCount != 3 {
		t.Errorf("expected 3 configs, got %d", overview.ConfigCount)
	}
	if overview.LastSync == nil {
		t.Fatal("expected LastSync to be set")
	}

	// Check config statuses
	if len(overview.Configs) != 3 {
		t.Fatalf("expected 3 config statuses, got %d", len(overview.Configs))
	}

	tests := []struct {
		name   string
		status SyncStatus
		isCore bool
	}{
		{"zsh", SyncStatusSynced, true},
		{"nvim", SyncStatusDrifted, true},
		{"tmux", SyncStatusNotInstalled, false},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := overview.Configs[i]
			if cs.Name != tt.name {
				t.Errorf("expected name %q, got %q", tt.name, cs.Name)
			}
			if cs.Status != tt.status {
				t.Errorf("expected status %q, got %q", tt.status, cs.Status)
			}
			if cs.IsCore != tt.isCore {
				t.Errorf("expected IsCore %v, got %v", tt.isCore, cs.IsCore)
			}
		})
	}

	// Check nvim drift details
	nvimStatus := overview.Configs[1]
	if nvimStatus.NewFiles != 1 {
		t.Errorf("expected 1 new file, got %d", nvimStatus.NewFiles)
	}
	if nvimStatus.Conflicts != 1 {
		t.Errorf("expected 1 conflict, got %d", nvimStatus.Conflicts)
	}

	// Check dependency summary
	if overview.Dependencies.Installed != 2 {
		t.Errorf("expected 2 installed deps, got %d", overview.Dependencies.Installed)
	}
	if overview.Dependencies.Missing != 1 {
		t.Errorf("expected 1 missing dep, got %d", overview.Dependencies.Missing)
	}
	if overview.Dependencies.Total != 3 {
		t.Errorf("expected 3 total deps, got %d", overview.Dependencies.Total)
	}
}

func TestGather_SkipDrift(t *testing.T) {
	p := &platform.Platform{OS: "linux", PackageManager: "apt", Architecture: "amd64"}
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{{Name: "zsh", Path: "zsh"}},
		},
	}
	st := &state.State{
		Configs: []state.ConfigState{{Name: "zsh"}},
	}

	driftCalled := false
	g := &Gatherer{
		PlatformDetector: func() (*platform.Platform, error) { return p, nil },
		ConfigLoader:     func() (*config.Config, string, error) { return cfg, "/tmp/.go4dot.yaml", nil },
		StateLoader:      func() (*state.State, error) { return st, nil },
		DriftChecker: func(_ *config.Config, _ string) (*stow.DriftSummary, error) {
			driftCalled = true
			return nil, nil
		},
		DepsChecker: func(_ *config.Config, _ *platform.Platform) (*deps.CheckResult, error) {
			return &deps.CheckResult{}, nil
		},
	}

	_, err := g.Gather(GatherOptions{SkipDrift: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if driftCalled {
		t.Error("drift checker should not be called when SkipDrift is true")
	}
}

func TestGather_SkipDeps(t *testing.T) {
	p := &platform.Platform{OS: "linux", PackageManager: "apt", Architecture: "amd64"}
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{{Name: "zsh", Path: "zsh"}},
		},
	}

	depsCalled := false
	g := &Gatherer{
		PlatformDetector: func() (*platform.Platform, error) { return p, nil },
		ConfigLoader:     func() (*config.Config, string, error) { return cfg, "/tmp/.go4dot.yaml", nil },
		StateLoader:      func() (*state.State, error) { return nil, nil },
		DriftChecker: func(_ *config.Config, _ string) (*stow.DriftSummary, error) {
			return &stow.DriftSummary{}, nil
		},
		DepsChecker: func(_ *config.Config, _ *platform.Platform) (*deps.CheckResult, error) {
			depsCalled = true
			return nil, nil
		},
	}

	_, err := g.Gather(GatherOptions{SkipDeps: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if depsCalled {
		t.Error("deps checker should not be called when SkipDeps is true")
	}
}

func TestGather_WSLPlatform(t *testing.T) {
	p := &platform.Platform{
		OS:             "linux",
		Distro:         "ubuntu",
		DistroVersion:  "22.04",
		PackageManager: "apt",
		Architecture:   "amd64",
		IsWSL:          true,
	}

	g := newTestGatherer(p, nil, "", nil, nil, nil)
	overview, err := g.Gather(GatherOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !overview.Platform.IsWSL {
		t.Error("expected IsWSL to be true")
	}
}

func TestSummarizeDeps(t *testing.T) {
	tests := []struct {
		name     string
		result   *deps.CheckResult
		expected DependencyStatus
	}{
		{
			name:     "empty",
			result:   &deps.CheckResult{},
			expected: DependencyStatus{},
		},
		{
			name: "all installed",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Status: deps.StatusInstalled},
				},
				Core: []deps.DependencyCheck{
					{Status: deps.StatusInstalled},
				},
			},
			expected: DependencyStatus{Installed: 2, Total: 2},
		},
		{
			name: "mixed statuses",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Status: deps.StatusInstalled},
				},
				Core: []deps.DependencyCheck{
					{Status: deps.StatusMissing},
				},
				Optional: []deps.DependencyCheck{
					{Status: deps.StatusVersionMismatch},
				},
			},
			expected: DependencyStatus{Installed: 1, Missing: 1, VersionMissing: 1, Total: 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds := summarizeDeps(tt.result)
			if ds != tt.expected {
				t.Errorf("got %+v, expected %+v", ds, tt.expected)
			}
		})
	}
}
