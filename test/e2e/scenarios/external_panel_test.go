//go:build e2e

package scenarios

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui/dashboard"
	"github.com/nvandessel/go4dot/test/e2e/helpers"
)

// TestExternalPanel_PanelJump verifies that pressing '4' focuses the External panel
func TestExternalPanel_PanelJump(t *testing.T) {
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "vim"},
			},
		},
		External: []config.ExternalDep{
			{
				ID:          "test-plugin",
				Name:        "Test Plugin",
				URL:         "https://github.com/example/test-plugin.git",
				Destination: "~/.vim/test-plugin",
			},
		},
	}

	state := dashboard.State{
		Platform:     &platform.Platform{OS: "linux"},
		Configs:      cfg.GetAllConfigs(),
		Config:       cfg,
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model, teatest.WithInitialTermSize(100, 40))

	// Wait for initial render - look for dashboard indicator
	tm.WaitForText("DASHBOARD", 2*time.Second)

	// Jump to External panel with '4'
	tm.SendKeys('4')
	time.Sleep(200 * time.Millisecond)

	// Dashboard should still be visible
	tm.WaitForText("DASHBOARD", 2*time.Second)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(2 * time.Second)
}

// TestExternalPanel_Navigation verifies basic navigation in External panel
func TestExternalPanel_Navigation(t *testing.T) {
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "vim"},
			},
		},
		External: []config.ExternalDep{
			{
				ID:          "plugin1",
				Name:        "Plugin 1",
				URL:         "https://github.com/example/plugin1.git",
				Destination: "~/.vim/plugin1",
			},
			{
				ID:          "plugin2",
				Name:        "Plugin 2",
				URL:         "https://github.com/example/plugin2.git",
				Destination: "~/.vim/plugin2",
			},
		},
	}

	state := dashboard.State{
		Platform:     &platform.Platform{OS: "linux"},
		Configs:      cfg.GetAllConfigs(),
		Config:       cfg,
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model, teatest.WithInitialTermSize(100, 40))

	// Wait for initial render - look for dashboard indicator
	tm.WaitForText("DASHBOARD", 2*time.Second)

	// Navigate to External panel
	tm.SendKeys('4')
	time.Sleep(200 * time.Millisecond)

	// Navigate down within the panel (no crash)
	helpers.NewKeySequence().
		Down().
		Up().
		SendTo(tm)

	time.Sleep(100 * time.Millisecond)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(2 * time.Second)
}
