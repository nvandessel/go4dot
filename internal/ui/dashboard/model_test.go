package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestModel_Update_Help(t *testing.T) {
	m := New(&platform.Platform{}, nil, nil, nil, []config.ConfigItem{}, "", "", false, "", "")

	// Initially help should be hidden
	if m.showHelp {
		t.Error("expected showHelp to be false initially")
	}

	// Press '?' to show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	if !m.showHelp {
		t.Error("expected showHelp to be true after pressing '?'")
	}

	// Press '?' again to hide help
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	if m.showHelp {
		t.Error("expected showHelp to be false after pressing '?' again")
	}

	// Press 'q' to hide help when it's shown
	m.showHelp = true
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	if m.showHelp {
		t.Error("expected showHelp to be false after pressing 'q'")
	}
}

func TestModel_View_Layout(t *testing.T) {
	// This test ensures View doesn't crash and returns something
	m := New(&platform.Platform{}, nil, nil, nil, []config.ConfigItem{
		{Name: "test"},
	}, "", "", false, "", "")
	m.width = 80
	m.height = 24

	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// Check if help overlay is rendered when showHelp is true
	m.showHelp = true
	view = m.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("expected help overlay in view")
	}
}

func TestModel_renderStatus(t *testing.T) {
	tests := []struct {
		name         string
		hasBaseline  bool
		driftSummary *stow.DriftSummary
		expected     string
	}{
		{
			name:        "No baseline",
			hasBaseline: false,
			expected:    "Not synced yet",
		},
		{
			name:        "All synced",
			hasBaseline: true,
			driftSummary: &stow.DriftSummary{
				DriftedConfigs: 0,
			},
			expected: "All synced",
		},
		{
			name:        "Needs syncing",
			hasBaseline: true,
			driftSummary: &stow.DriftSummary{
				DriftedConfigs: 2,
			},
			expected: "2 config(s) need syncing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(&platform.Platform{}, tt.driftSummary, nil, nil, []config.ConfigItem{}, "", "", tt.hasBaseline, "", "")
			status := m.renderStatus()
			if !strings.Contains(status, tt.expected) {
				t.Errorf("expected status to contain %q, got %q", tt.expected, status)
			}
		})
	}
}

func TestModel_Update_Navigation(t *testing.T) {
	configs := []config.ConfigItem{
		{Name: "config1"},
		{Name: "config2"},
		{Name: "config3"},
	}
	m := New(&platform.Platform{}, nil, nil, nil, configs, "", "", true, "", "")

	// Initial selection should be 0
	if m.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", m.selectedIdx)
	}

	// Press 'j' to move down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	if m.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1, got %d", m.selectedIdx)
	}

	// Press 'k' to move up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}
	updatedModel, _ = m.Update(msg)
	m = updatedModel.(Model)

	if m.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", m.selectedIdx)
	}
}

func TestModel_StatePreservation(t *testing.T) {
	configs := []config.ConfigItem{
		{Name: "config1"},
		{Name: "config2"},
		{Name: "config3"},
	}

	// Test initial state restoration
	m := New(&platform.Platform{}, nil, nil, nil, configs, "", "", true, "config", "config2")

	if m.filterText != "config" {
		t.Errorf("expected filterText 'config', got %q", m.filterText)
	}

	// config2 should be selected
	if m.configs[m.selectedIdx].Name != "config2" {
		t.Errorf("expected selected config 'config2', got %q", m.configs[m.selectedIdx].Name)
	}

	// Test state capture in result
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(Model)

	if m.result.FilterText != "config" {
		t.Errorf("expected result FilterText 'config', got %q", m.result.FilterText)
	}

	if m.result.SelectedConfig != "config2" {
		t.Errorf("expected result SelectedConfig 'config2', got %q", m.result.SelectedConfig)
	}
}
