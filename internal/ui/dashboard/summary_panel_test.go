package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestSummaryPanel_NewSummaryPanel(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
	}

	panel := NewSummaryPanel(state)

	if panel == nil {
		t.Fatal("expected panel to be created")
	}
	if panel.GetID() != PanelSummary {
		t.Errorf("expected PanelSummary ID, got %v", panel.GetID())
	}
	if panel.GetTitle() != "1 Summary" {
		t.Errorf("expected title '1 Summary', got '%s'", panel.GetTitle())
	}
	if panel.selectedCount != 0 {
		t.Errorf("expected selectedCount 0, got %d", panel.selectedCount)
	}
}

func TestSummaryPanel_Init(t *testing.T) {
	panel := NewSummaryPanel(State{})
	cmd := panel.Init()
	if cmd != nil {
		t.Error("expected nil command from Init")
	}
}

func TestSummaryPanel_Update(t *testing.T) {
	panel := NewSummaryPanel(State{})
	cmd := panel.Update(nil)
	if cmd != nil {
		t.Error("expected nil command from Update")
	}
}

func TestSummaryPanel_GetSelectedItem(t *testing.T) {
	panel := NewSummaryPanel(State{})
	item := panel.GetSelectedItem()
	if item != nil {
		t.Error("expected nil from GetSelectedItem (summary is not navigable)")
	}
}

func TestSummaryPanel_View_TooSmall(t *testing.T) {
	panel := NewSummaryPanel(State{})
	panel.SetSize(4, 2) // Too small
	view := panel.View()
	if view != "" {
		t.Errorf("expected empty view for small panel, got '%s'", view)
	}
}

func TestSummaryPanel_View_ConfigCount(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "3") {
		t.Errorf("expected view to contain config count '3', got:\n%s", view)
	}
	if !strings.Contains(view, "configs") {
		t.Errorf("expected view to contain 'configs', got:\n%s", view)
	}
}

func TestSummaryPanel_View_SelectedCount(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)
	panel.SetSelectedCount(2)

	view := panel.View()
	if !strings.Contains(view, "2") {
		t.Errorf("expected view to contain selected count '2', got:\n%s", view)
	}
	if !strings.Contains(view, "of 3 selected") {
		t.Errorf("expected view to contain 'of 3 selected', got:\n%s", view)
	}
}

func TestSummaryPanel_View_NotSynced(t *testing.T) {
	state := State{
		Platform:    &platform.Platform{OS: "linux"},
		Configs:     []config.ConfigItem{{Name: "vim"}},
		HasBaseline: false,
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "Not synced") {
		t.Errorf("expected view to contain 'Not synced', got:\n%s", view)
	}
}

func TestSummaryPanel_View_SyncStatusCounts(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
		LinkStatus: map[string]*stow.ConfigLinkStatus{
			"vim": {
				ConfigName:  "vim",
				LinkedCount: 5,
				TotalCount:  5,
			},
			"zsh": {
				ConfigName:  "zsh",
				LinkedCount: 3,
				TotalCount:  5,
			},
		},
		DriftSummary: &stow.DriftSummary{
			TotalConfigs:   3,
			DriftedConfigs: 1,
			Results: []stow.DriftResult{
				{ConfigName: "zsh", HasDrift: true, NewFiles: []string{"new.zsh"}},
			},
		},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(40, 12)

	view := panel.View()
	if !strings.Contains(view, "1 synced") {
		t.Errorf("expected view to contain '1 synced', got:\n%s", view)
	}
	if !strings.Contains(view, "1 drifted") {
		t.Errorf("expected view to contain '1 drifted', got:\n%s", view)
	}
	if !strings.Contains(view, "1 unlinked") {
		t.Errorf("expected view to contain '1 unlinked', got:\n%s", view)
	}
}

func TestSummaryPanel_View_AllSynced(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		LinkStatus: map[string]*stow.ConfigLinkStatus{
			"vim": {
				ConfigName:  "vim",
				LinkedCount: 3,
				TotalCount:  3,
			},
			"zsh": {
				ConfigName:  "zsh",
				LinkedCount: 2,
				TotalCount:  2,
			},
		},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "All synced") {
		t.Errorf("expected view to contain 'All synced', got:\n%s", view)
	}
}

func TestSummaryPanel_View_PlatformInfo_Linux(t *testing.T) {
	state := State{
		Platform: &platform.Platform{
			OS:             "linux",
			Distro:         "fedora",
			PackageManager: "dnf",
		},
		Configs: []config.ConfigItem{{Name: "vim"}},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "linux/fedora") {
		t.Errorf("expected view to contain 'linux/fedora', got:\n%s", view)
	}
	if !strings.Contains(view, "dnf") {
		t.Errorf("expected view to contain 'dnf', got:\n%s", view)
	}
}

func TestSummaryPanel_View_PlatformInfo_MacOS(t *testing.T) {
	state := State{
		Platform: &platform.Platform{
			OS:             "darwin",
			PackageManager: "brew",
		},
		Configs: []config.ConfigItem{{Name: "vim"}},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "darwin") {
		t.Errorf("expected view to contain 'darwin', got:\n%s", view)
	}
	if !strings.Contains(view, "brew") {
		t.Errorf("expected view to contain 'brew', got:\n%s", view)
	}
}

func TestSummaryPanel_View_PlatformInfo_NoPkgMgr(t *testing.T) {
	state := State{
		Platform: &platform.Platform{
			OS:             "linux",
			Distro:         "alpine",
			PackageManager: "none",
		},
		Configs: []config.ConfigItem{{Name: "vim"}},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "linux/alpine") {
		t.Errorf("expected view to contain 'linux/alpine', got:\n%s", view)
	}
	// "none" should not appear
	if strings.Contains(view, "none") {
		t.Errorf("expected view NOT to contain 'none', got:\n%s", view)
	}
}

func TestSummaryPanel_View_NilPlatform(t *testing.T) {
	state := State{
		Platform: nil,
		Configs:  []config.ConfigItem{{Name: "vim"}},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	// Should not panic
	view := panel.View()
	if view == "" {
		t.Error("expected non-empty view even without platform")
	}
}

func TestSummaryPanel_View_Dependencies(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs:  []config.ConfigItem{{Name: "vim"}},
		Config: &config.Config{
			Dependencies: config.Dependencies{
				Critical: []config.DependencyItem{
					{Name: "git"},
					{Name: "stow"},
				},
				Core: []config.DependencyItem{
					{Name: "curl"},
				},
				Optional: []config.DependencyItem{
					{Name: "fzf"},
				},
			},
		},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "4 dependencies") {
		t.Errorf("expected view to contain '4 dependencies', got:\n%s", view)
	}
}

func TestSummaryPanel_View_NoDependencies(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs:  []config.ConfigItem{{Name: "vim"}},
		Config:   &config.Config{},
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	view := panel.View()
	// Should not contain "dependencies" when there are none
	if strings.Contains(view, "dependencies") {
		t.Errorf("expected view NOT to contain 'dependencies' when none exist, got:\n%s", view)
	}
}

func TestSummaryPanel_View_SourcePath(t *testing.T) {
	state := State{
		Platform:     &platform.Platform{OS: "linux"},
		Configs:      []config.ConfigItem{{Name: "vim"}},
		DotfilesPath: "/home/user/dotfiles",
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(40, 12)

	view := panel.View()
	if !strings.Contains(view, "/home/user/dotfiles") {
		t.Errorf("expected view to contain source path, got:\n%s", view)
	}
}

func TestSummaryPanel_View_SourcePathTruncated(t *testing.T) {
	state := State{
		Platform:     &platform.Platform{OS: "linux"},
		Configs:      []config.ConfigItem{{Name: "vim"}},
		DotfilesPath: "/very/long/path/to/my/special/dotfiles/directory/that/is/really/long",
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(20, 12) // Very narrow panel

	view := panel.View()
	if !strings.Contains(view, "...") {
		t.Errorf("expected truncated path to contain '...', got:\n%s", view)
	}
}

func TestSummaryPanel_View_NoSourcePath(t *testing.T) {
	state := State{
		Platform:     &platform.Platform{OS: "linux"},
		Configs:      []config.ConfigItem{{Name: "vim"}},
		DotfilesPath: "",
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(30, 12)

	// Should not panic and should produce output
	view := panel.View()
	if view == "" {
		t.Error("expected non-empty view even without dotfiles path")
	}
}

func TestSummaryPanel_UpdateState(t *testing.T) {
	state1 := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs:  []config.ConfigItem{{Name: "vim"}},
	}
	panel := NewSummaryPanel(state1)

	state2 := State{
		Platform: &platform.Platform{OS: "darwin"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
	}
	panel.UpdateState(state2)
	panel.SetSize(30, 12)

	view := panel.View()
	if !strings.Contains(view, "3") {
		t.Errorf("expected updated config count '3', got:\n%s", view)
	}
	if !strings.Contains(view, "darwin") {
		t.Errorf("expected updated platform 'darwin', got:\n%s", view)
	}
}

func TestSummaryPanel_SetSelectedCount(t *testing.T) {
	state := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
	}
	panel := NewSummaryPanel(state)

	panel.SetSelectedCount(1)
	if panel.selectedCount != 1 {
		t.Errorf("expected selectedCount 1, got %d", panel.selectedCount)
	}

	panel.SetSelectedCount(0)
	if panel.selectedCount != 0 {
		t.Errorf("expected selectedCount 0, got %d", panel.selectedCount)
	}
}

func TestSummaryPanel_computeSyncCounts(t *testing.T) {
	tests := []struct {
		name               string
		state              State
		expectSynced       int
		expectDrifted      int
		expectNotInstalled int
	}{
		{
			name: "all synced",
			state: State{
				Configs: []config.ConfigItem{
					{Name: "vim"},
					{Name: "zsh"},
				},
				LinkStatus: map[string]*stow.ConfigLinkStatus{
					"vim": {LinkedCount: 3, TotalCount: 3},
					"zsh": {LinkedCount: 2, TotalCount: 2},
				},
			},
			expectSynced:       2,
			expectDrifted:      0,
			expectNotInstalled: 0,
		},
		{
			name: "one drifted via drift summary",
			state: State{
				Configs: []config.ConfigItem{
					{Name: "vim"},
					{Name: "zsh"},
				},
				LinkStatus: map[string]*stow.ConfigLinkStatus{
					"vim": {LinkedCount: 3, TotalCount: 3},
					"zsh": {LinkedCount: 2, TotalCount: 2},
				},
				DriftSummary: &stow.DriftSummary{
					Results: []stow.DriftResult{
						{ConfigName: "zsh", HasDrift: true},
					},
				},
			},
			expectSynced:       1,
			expectDrifted:      1,
			expectNotInstalled: 0,
		},
		{
			name: "partially linked counts as drifted",
			state: State{
				Configs: []config.ConfigItem{
					{Name: "vim"},
				},
				LinkStatus: map[string]*stow.ConfigLinkStatus{
					"vim": {LinkedCount: 2, TotalCount: 5},
				},
			},
			expectSynced:       0,
			expectDrifted:      1,
			expectNotInstalled: 0,
		},
		{
			name: "no link status is not installed",
			state: State{
				Configs: []config.ConfigItem{
					{Name: "vim"},
					{Name: "zsh"},
				},
				LinkStatus: map[string]*stow.ConfigLinkStatus{},
			},
			expectSynced:       0,
			expectDrifted:      0,
			expectNotInstalled: 2,
		},
		{
			name: "mixed statuses",
			state: State{
				Configs: []config.ConfigItem{
					{Name: "vim"},
					{Name: "zsh"},
					{Name: "tmux"},
					{Name: "git"},
				},
				LinkStatus: map[string]*stow.ConfigLinkStatus{
					"vim":  {LinkedCount: 3, TotalCount: 3},
					"zsh":  {LinkedCount: 1, TotalCount: 3},
					"tmux": {LinkedCount: 0, TotalCount: 0},
				},
				DriftSummary: &stow.DriftSummary{
					Results: []stow.DriftResult{
						{ConfigName: "vim", HasDrift: false},
					},
				},
			},
			expectSynced:       1, // vim
			expectDrifted:      1, // zsh (partially linked)
			expectNotInstalled: 2, // tmux (0/0) and git (no entry)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewSummaryPanel(tt.state)
			synced, drifted, notInstalled := panel.computeSyncCounts()

			if synced != tt.expectSynced {
				t.Errorf("expected synced=%d, got %d", tt.expectSynced, synced)
			}
			if drifted != tt.expectDrifted {
				t.Errorf("expected drifted=%d, got %d", tt.expectDrifted, drifted)
			}
			if notInstalled != tt.expectNotInstalled {
				t.Errorf("expected notInstalled=%d, got %d", tt.expectNotInstalled, notInstalled)
			}
		})
	}
}

func TestSummaryPanel_renderConfigLine(t *testing.T) {
	tests := []struct {
		name          string
		configs       []config.ConfigItem
		selectedCount int
		expectContain string
	}{
		{
			name:          "no selection shows count",
			configs:       []config.ConfigItem{{Name: "vim"}, {Name: "zsh"}},
			selectedCount: 0,
			expectContain: "configs",
		},
		{
			name:          "with selection",
			configs:       []config.ConfigItem{{Name: "vim"}, {Name: "zsh"}, {Name: "tmux"}},
			selectedCount: 2,
			expectContain: "of 3 selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewSummaryPanel(State{Configs: tt.configs})
			panel.selectedCount = tt.selectedCount
			panel.SetSize(40, 12)

			view := panel.View()
			if !strings.Contains(view, tt.expectContain) {
				t.Errorf("expected view to contain '%s', got:\n%s", tt.expectContain, view)
			}
		})
	}
}

func TestSummaryPanel_renderPlatformLine(t *testing.T) {
	tests := []struct {
		name          string
		platform      *platform.Platform
		expectContain string
		expectMissing string
	}{
		{
			name: "linux with distro and pkg manager",
			platform: &platform.Platform{
				OS:             "linux",
				Distro:         "ubuntu",
				PackageManager: "apt",
			},
			expectContain: "linux/ubuntu",
		},
		{
			name: "darwin with brew",
			platform: &platform.Platform{
				OS:             "darwin",
				PackageManager: "brew",
			},
			expectContain: "darwin",
		},
		{
			name: "unknown pkg manager hidden",
			platform: &platform.Platform{
				OS:             "linux",
				Distro:         "custom",
				PackageManager: "unknown",
			},
			expectContain: "linux/custom",
			expectMissing: "unknown",
		},
		{
			name:     "nil platform produces empty",
			platform: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				Platform: tt.platform,
				Configs:  []config.ConfigItem{{Name: "vim"}},
			}
			panel := NewSummaryPanel(state)
			panel.SetSize(40, 12)

			view := panel.View()
			if tt.expectContain != "" && !strings.Contains(view, tt.expectContain) {
				t.Errorf("expected view to contain '%s', got:\n%s", tt.expectContain, view)
			}
			if tt.expectMissing != "" && strings.Contains(view, tt.expectMissing) {
				t.Errorf("expected view NOT to contain '%s', got:\n%s", tt.expectMissing, view)
			}
		})
	}
}

func TestSummaryPanel_View_FullState(t *testing.T) {
	// Test with a complete state to verify all lines render
	state := State{
		Platform: &platform.Platform{
			OS:             "linux",
			Distro:         "fedora",
			PackageManager: "dnf",
		},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
		LinkStatus: map[string]*stow.ConfigLinkStatus{
			"vim": {LinkedCount: 3, TotalCount: 3},
			"zsh": {LinkedCount: 2, TotalCount: 2},
		},
		Config: &config.Config{
			Dependencies: config.Dependencies{
				Critical: []config.DependencyItem{
					{Name: "git"},
					{Name: "stow"},
				},
			},
		},
		DotfilesPath: "/home/user/dotfiles",
		HasBaseline:  true,
	}
	panel := NewSummaryPanel(state)
	panel.SetSize(40, 14)
	panel.SetSelectedCount(1)

	view := panel.View()

	// All sections should be present
	expectations := []string{
		"1",                    // selected count
		"of 3 selected",       // config selection
		"2 synced",            // sync status
		"1 unlinked",          // not installed
		"linux/fedora",        // platform
		"dnf",                 // package manager
		"2 dependencies",      // deps
		"/home/user/dotfiles", // source path
	}

	for _, expected := range expectations {
		if !strings.Contains(view, expected) {
			t.Errorf("expected view to contain '%s', got:\n%s", expected, view)
		}
	}
}

func TestSummaryPanel_View_HeightLimiting(t *testing.T) {
	// Test with very limited height - should not panic
	state := State{
		Platform: &platform.Platform{
			OS:             "linux",
			Distro:         "fedora",
			PackageManager: "dnf",
		},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		Config: &config.Config{
			Dependencies: config.Dependencies{
				Critical: []config.DependencyItem{{Name: "git"}},
			},
		},
		DotfilesPath: "/home/user/dotfiles",
	}
	panel := NewSummaryPanel(state)
	// Height=5 means ContentHeight()=3 (5-2 for borders)
	// We have up to 5 lines of content, so some will be trimmed
	panel.SetSize(30, 5)

	view := panel.View()
	if view == "" {
		t.Error("expected non-empty view with limited height")
	}

	// Count lines - should not exceed content height
	lines := strings.Split(view, "\n")
	maxLines := panel.ContentHeight()
	if len(lines) > maxLines {
		t.Errorf("expected at most %d lines, got %d", maxLines, len(lines))
	}
}

func TestSummaryPanel_View_EmptyState(t *testing.T) {
	// Minimal state - should still render without panic
	panel := NewSummaryPanel(State{})
	panel.SetSize(30, 12)

	view := panel.View()
	// Should show at least config count (0)
	if !strings.Contains(view, "0") {
		t.Errorf("expected view to contain '0' for empty config count, got:\n%s", view)
	}
}
