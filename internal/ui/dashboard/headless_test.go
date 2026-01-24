package dashboard_test

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui/dashboard"
)

func TestDashboard_Headless(t *testing.T) {
	// Setup initial state
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim", Description: "Vim configuration"},
			{Name: "zsh", Description: "Zsh configuration"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	// Create model
	m := dashboard.New(s)

	// Initialize test model
	// We use a slightly larger size to ensure all components render
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	// Wait for the model to initialize and render specific content
	// We use WaitFor to ensure the UI has time to render the config names
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		s := string(out)
		return strings.Contains(s, "vim") && strings.Contains(s, "zsh")
	}, teatest.WithCheckInterval(time.Millisecond*50), teatest.WithDuration(time.Second))

	// Test interaction: Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Wait for program to finish
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
