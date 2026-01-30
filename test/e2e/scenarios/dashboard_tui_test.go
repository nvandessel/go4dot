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

// TestDashboard_Navigation demonstrates the new teatest helpers
// This test validates navigation within the dashboard TUI
func TestDashboard_Navigation(t *testing.T) {
	// Create a test state with sample configs
	state := dashboard.State{
		Platform: &platform.Platform{OS: "linux", Distro: "fedora"},
		Configs: []config.ConfigItem{
			{Name: "vim", Description: "Vim configuration"},
			{Name: "zsh", Description: "Zsh shell configuration"},
			{Name: "tmux", Description: "Tmux terminal multiplexer"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	model := dashboard.New(state)

	// Create TUI test model with extended helpers
	tm := helpers.NewTUITestModel(
		t,
		&model,
		teatest.WithInitialTermSize(100, 40),
	)

	// Wait for initial render - dashboard should show config list
	tm.WaitForText("vim", 2*time.Second)

	// Use key sequence builder to navigate down
	seq := helpers.NewKeySequence().
		Down().  // Move to zsh
		Down().  // Move to tmux
		Up()     // Move back to zsh

	seq.SendTo(tm)

	// Verify we can see zsh after navigation
	tm.WaitForText("zsh", 1*time.Second)

	// Test filter mode with key sequence
	filterSeq := helpers.NewKeySequence().
		Type("/").    // Enter filter mode
		Type("vim").  // Type filter text
		Esc()         // Exit filter mode

	filterSeq.SendTo(tm)

	// Test help toggle
	tm.SendKeys('?')
	tm.WaitForText("Keyboard Shortcuts", 1*time.Second)

	// Close help
	tm.SendKeys('?')

	// Wait for help to close
	tm.WaitForNotText("Keyboard Shortcuts", 1*time.Second)

	// Quit the dashboard
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(2 * time.Second)
}

// TestDashboard_Selection demonstrates selection testing with teatest helpers
func TestDashboard_Selection(t *testing.T) {
	state := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	tm.WaitForText("vim", 2*time.Second)

	// Select first item with space
	helpers.NewKeySequence().
		Space(). // Select vim
		Down().  // Move to zsh
		Space(). // Select zsh
		SendTo(tm)

	// Verify we're on zsh after navigation
	tm.WaitForText("zsh", 1*time.Second)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(1 * time.Second)
}

// TestDashboard_NoConfig demonstrates testing the no-config view
func TestDashboard_NoConfig(t *testing.T) {
	state := dashboard.State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model)

	// Wait for no-config view to render
	tm.WaitForText("No configuration found", 1*time.Second)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(1 * time.Second)
}

// TestKeySequenceBuilder demonstrates the fluent API for building key sequences
func TestKeySequenceBuilder(t *testing.T) {
	state := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
		HasConfig: true,
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model)

	// Wait for initial render
	tm.WaitForText("vim", 2*time.Second)

	// Test comprehensive key sequence with navigation and selection
	helpers.NewKeySequence().
		Down().            // Move down to zsh
		Down().            // Move down to tmux
		Up().              // Move back up to zsh
		Space().           // Select zsh
		SendTo(tm)

	// Verify we're on zsh
	tm.WaitForText("zsh", 1*time.Second)

	// Test filter mode with backspace
	helpers.NewKeySequence().
		Type("/").         // Enter filter mode
		Type("vim").       // Type "vim"
		Backspace().       // Delete 'm'
		Backspace().       // Delete 'i'
		Type("zsh").       // Type "zsh"
		Esc().             // Exit filter mode
		SendTo(tm)

	// Test SendToWithDelay with custom timing (no delay)
	helpers.NewKeySequence().
		Tab().             // Open menu
		Esc().             // Close menu
		SendToWithDelay(tm, 0)  // No delay between keys

	// Verify we can still see configs
	tm.WaitForText("vim", 1*time.Second)

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(1 * time.Second)
}
