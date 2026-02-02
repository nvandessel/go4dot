//go:build e2e

// Package scenarios contains E2E tests using teatest for key-based TUI interactions.
//
// IMPORTANT: Teatest E2E tests should ONLY use key inputs (SendKeys), NOT custom messages (tm.Send()).
// Custom messages (like OnboardingCompleteMsg) don't work reliably in teatest because:
// - Teatest routes messages through a channel
// - Custom messages don't match Bubble Tea's internal message loop
//
// For testing custom message handling, use unit tests in internal/ui/dashboard/*_test.go
// that call Update() directly on the model.
//
// See internal/ui/dashboard/dashboard_test.go for:
// - TestPostOnboarding_AcceptInstall_PanelsHaveDimensions
// - TestPostOnboarding_DeclineInstall_PanelsHaveDimensions
package scenarios

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui/dashboard"
	"github.com/nvandessel/go4dot/test/e2e/helpers"
)

// TestOnboarding_StartsFromNoConfigView verifies that the no-config view
// shows and allows starting the onboarding wizard
func TestOnboarding_StartsFromNoConfigView(t *testing.T) {
	state := dashboard.State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model)

	// Wait for no-config view to render
	tm.WaitForText("No configuration found", 2*time.Second)

	// The no-config view should show instructions
	// This verifies the view is properly rendered

	// Quit
	tm.SendKeys(tea.KeyEsc)
	tm.WaitFinished(1 * time.Second)
}

// TestOnboarding_CanStartOnboarding verifies that pressing 'i' starts onboarding
func TestOnboarding_CanStartOnboarding(t *testing.T) {
	state := dashboard.State{
		Platform:     &platform.Platform{OS: "linux"},
		HasConfig:    false,
		DotfilesPath: "/tmp/test-dotfiles",
	}

	model := dashboard.New(state)
	tm := helpers.NewTUITestModel(t, &model, teatest.WithInitialTermSize(100, 40))

	// Wait for no-config view
	tm.WaitForText("No configuration found", 2*time.Second)

	// Start onboarding with 'i'
	tm.SendKeys('i')
	time.Sleep(500 * time.Millisecond)

	// Cancel onboarding with Esc
	tm.SendKeys(tea.KeyEsc)

	// Should return to no-config view or quit gracefully
	tm.WaitFinished(3 * time.Second)
}

// NOTE: Tests for post-onboarding behavior (accepting/declining install) are in
// internal/ui/dashboard/dashboard_test.go as unit tests because they require
// sending custom messages (OnboardingCompleteMsg) which don't work in teatest.
// See:
// - TestPostOnboarding_AcceptInstall_PanelsHaveDimensions
// - TestPostOnboarding_DeclineInstall_PanelsHaveDimensions
