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
	m := New(&platform.Platform{}, nil, nil, nil, []config.ConfigItem{}, "", "", false)

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

func TestModel_Update_Refresh(t *testing.T) {
	m := New(&platform.Platform{}, nil, nil, nil, []config.ConfigItem{}, "", "", false)

	// Press 'r' to refresh
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(Model)

	if m.result == nil || m.result.Action != ActionRefresh {
		t.Errorf("expected ActionRefresh, got %v", m.result)
	}

	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestModel_View_Layout(t *testing.T) {
	// This test ensures View doesn't crash and returns something
	m := New(&platform.Platform{}, nil, nil, nil, []config.ConfigItem{
		{Name: "test"},
	}, "", "", false)
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
			m := New(&platform.Platform{}, tt.driftSummary, nil, nil, []config.ConfigItem{}, "", "", tt.hasBaseline)
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
	m := New(&platform.Platform{}, nil, nil, nil, configs, "", "", true)

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
