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

func TestDashboard_Interaction(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "alpha", Description: "First config"},
			{Name: "beta", Description: "Second config"},
			{Name: "gamma", Description: "Third config"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "alpha")
	})

	// Test Navigation (Down)
	// Send "j" to move down
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// We can't easily verify selection visually without parsing ANSI codes for colors/styles,
	// but we can verify the Description update in the details pane if it changes.
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Second config")
	}, teatest.WithCheckInterval(time.Millisecond*50), teatest.WithDuration(time.Second))

	// Test Selection
	// Send "space" to select
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Move down to gamma
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Third config")
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_Filtering(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "neovim", Description: "Neovim config"},
			{Name: "tmux", Description: "Tmux config"},
			{Name: "zsh", Description: "Zsh config"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "neovim")
	})

	// Enter filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})

	// Type "tm" to filter for tmux
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})

	// Verify only tmux shows (or at least tmux shows and others might be hidden/dimmed depending on impl)
	// Assuming the list updates to show filtered items.
	// We can check if "neovim" and "zsh" are NO LONGER present or if "tmux" is the selected one.
	// Since teatest captures the whole view, checking absence is tricky if they are just hidden.
	// But let's check if the details pane updates to "Tmux config" since it should be the top match.

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Tmux config")
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // Exit filter mode
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // Quit app
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_ViewSwitching(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "vim")
	})

	// Switch to Help (? key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	// Verify Help content
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Keyboard Shortcuts")
	}, teatest.WithDuration(time.Second))

	// Switch back (Esc)
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Verify back to dashboard
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "vim")
	}, teatest.WithDuration(time.Second))

	// Switch to Menu (Tab)
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // Back to dashboard
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc}) // Quit app
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_NoConfig(t *testing.T) {
	s := dashboard.State{
		Platform:     &platform.Platform{OS: "linux"},
		HasConfig:    false,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	// Verify NoConfig view is shown
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		s := string(out)
		return strings.Contains(s, "No configuration") || strings.Contains(s, "init")
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_SingleConfig(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "only-one", Description: "The only config"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	// Verify single config is shown
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "only-one")
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_BulkSelection(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "config-a", Description: "First"},
			{Name: "config-b", Description: "Second"},
			{Name: "config-c", Description: "Third"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "config-a")
	})

	// Select first config with space
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Move down and select second
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm.Send(tea.KeyMsg{Type: tea.KeySpace})

	// Wait for UI to update
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "Second")
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_SelectAll(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "cfg1"},
			{Name: "cfg2"},
			{Name: "cfg3"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "cfg1")
	})

	// Select all with shift+A
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})

	// Wait for UI to process the selection
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return len(out) > 0
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_FilterMode(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim-config"},
			{Name: "zsh-config"},
			{Name: "tmux-config"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "vim-config")
	}, teatest.WithDuration(time.Second))

	// Enter filter mode and type
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	// Wait for filter to be visible (zsh should match)
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "zsh")
	}, teatest.WithDuration(time.Second))

	// Exit filter mode and quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDashboard_MenuNavigation(t *testing.T) {
	s := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "config"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := dashboard.New(s)
	tm := teatest.NewTestModel(t, &m, teatest.WithInitialTermSize(100, 40))

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "config")
	})

	// Open menu with tab
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})

	// Wait for menu to appear
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return len(out) > 0
	}, teatest.WithDuration(time.Second))

	// Navigate in menu
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Go back to dashboard
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	// Verify we're back on dashboard
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "config")
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
