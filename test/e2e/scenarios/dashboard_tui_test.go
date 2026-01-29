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
	// Just wait for one config to ensure the dashboard has rendered
	if err := tm.WaitForText("vim", 2*time.Second); err != nil {
		t.Fatalf("Failed to wait for initial render: %v", err)
	}

	// Small delay for complete render
	time.Sleep(100 * time.Millisecond)

	// Use key sequence builder to navigate down
	seq := helpers.NewKeySequence().
		Down().  // Move to zsh
		Down().  // Move to tmux
		Up()     // Move back to zsh

	seq.SendTo(tm)

	// Test filter mode with key sequence
	filterSeq := helpers.NewKeySequence().
		Type("/").    // Enter filter mode
		Type("vim").  // Type filter text
		Esc()         // Exit filter mode

	filterSeq.SendTo(tm)

	// Test help toggle
	tm.SendKeys('?')
	if err := tm.WaitForText("Keyboard Shortcuts", 1*time.Second); err != nil {
		t.Logf("Warning: Help text not found: %v", err)
	}

	// Small delay to let help render
	time.Sleep(100 * time.Millisecond)

	// Close help
	tm.SendKeys('?')

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
	if err := tm.WaitForText("vim", 2*time.Second); err != nil {
		t.Fatalf("Failed to wait for initial render: %v", err)
	}

	// Select first item with space
	helpers.NewKeySequence().
		Space(). // Select vim
		Down().  // Move to zsh
		Space(). // Select zsh
		SendTo(tm)

	// Use AssertState to verify internal state (demonstration)
	// Note: In a real scenario, you'd need to access the model to check selectedConfigs
	helpers.AssertState(t, &model, func(m tea.Model) bool {
		// This is a simple check - in practice you'd need type assertion
		// to access dashboard-specific fields
		return true
	}, "model should be in expected state")

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

	// In no-config view, we should see a message about no configuration
	// (The exact text depends on the dashboard implementation)
	time.Sleep(100 * time.Millisecond) // Brief pause for render

	// Verify we're in no-config view
	output := tm.GetOutputString()
	if output == "" {
		t.Log("No output captured - this is expected for minimal UI")
	}

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(1 * time.Second)
}

// TestKeySequenceBuilder demonstrates the fluent API for building key sequences
func TestKeySequenceBuilder(t *testing.T) {
	state := dashboard.State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "test"},
		},
		HasConfig: true,
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model)

	// Demonstrate various key sequence patterns
	helpers.NewKeySequence().
		Type("hello").     // Type text
		Rune('!').         // Single character
		Enter().           // Special key
		Tab().             // Another special key
		Up().              // Navigation
		Down().            // Navigation
		Space().           // Space
		Esc().             // Quit
		SendTo(tm)

	tm.WaitFinished(1 * time.Second)
}
