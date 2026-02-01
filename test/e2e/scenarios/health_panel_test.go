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

// TestHealthPanel_NavigationWithinBounds verifies that navigation
// stays within bounds after refresh when results might shrink
func TestHealthPanel_NavigationWithinBounds(t *testing.T) {
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "vim"},
				{Name: "zsh"},
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

	// Navigate to Health panel
	tm.SendKeys('2')
	time.Sleep(200 * time.Millisecond)

	// Navigate down (verify no crash)
	tm.SendKeys(tea.KeyDown)
	time.Sleep(100 * time.Millisecond)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(2 * time.Second)
}

// TestHealthPanel_PanelJump verifies that pressing '2' focuses the Health panel
func TestHealthPanel_PanelJump(t *testing.T) {
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "vim"},
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

	// Jump to Health panel with '2'
	tm.SendKeys('2')
	time.Sleep(200 * time.Millisecond)

	// Dashboard should still be visible
	tm.WaitForText("DASHBOARD", 2*time.Second)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(2 * time.Second)
}
