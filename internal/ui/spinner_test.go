package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSpinnerModel_Init(t *testing.T) {
	actionCalled := false
	action := func() error {
		actionCalled = true
		return nil
	}

	m := initialSpinnerModel("Testing", action)
	cmd := m.Init()

	if cmd == nil {
		t.Error("expected Init to return a command batch")
	}

	// The action should be called when the command runs
	// We can't easily test this without running the full tea program
	_ = actionCalled
}

func TestSpinnerModel_Update_Quit(t *testing.T) {
	action := func() error { return nil }
	m := initialSpinnerModel("Testing", action)

	tests := []struct {
		name string
		key  string
	}{
		{"quit with q", "q"},
		{"quit with esc", "esc"},
		{"quit with ctrl+c", "ctrl+c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := initialSpinnerModel("Testing", action)
			var msg tea.KeyMsg
			switch tt.key {
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			case "ctrl+c":
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			updatedModel, cmd := model.Update(msg)

			sm := updatedModel.(spinnerModel)
			if !sm.quitting {
				t.Error("expected quitting to be true")
			}
			if cmd == nil {
				t.Error("expected tea.Quit command")
			}
		})
	}

	_ = m
}

func TestSpinnerModel_Update_Error(t *testing.T) {
	action := func() error { return nil }
	m := initialSpinnerModel("Testing", action)

	testErr := errors.New("test error")
	updatedModel, cmd := m.Update(errMsg(testErr))

	sm := updatedModel.(spinnerModel)
	if sm.err == nil {
		t.Error("expected error to be set")
	}
	if sm.err.Error() != "test error" {
		t.Errorf("expected error message 'test error', got '%s'", sm.err.Error())
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestSpinnerModel_Update_Success(t *testing.T) {
	action := func() error { return nil }
	m := initialSpinnerModel("Testing", action)

	// nil message indicates success
	updatedModel, cmd := m.Update(nil)

	sm := updatedModel.(spinnerModel)
	if sm.err != nil {
		t.Error("expected no error on success")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command on success")
	}
}

func TestSpinnerModel_View(t *testing.T) {
	action := func() error { return nil }
	m := initialSpinnerModel("Installing packages", action)

	view := m.View()

	if !strings.Contains(view, "Installing packages") {
		t.Errorf("expected view to contain 'Installing packages', got '%s'", view)
	}
}

func TestSpinnerModel_View_Error(t *testing.T) {
	action := func() error { return nil }
	m := initialSpinnerModel("Installing packages", action)
	m.err = errors.New("connection failed")

	view := m.View()

	if !strings.Contains(view, "Installing packages") {
		t.Errorf("expected view to contain 'Installing packages', got '%s'", view)
	}
	if !strings.Contains(view, "connection failed") {
		t.Errorf("expected view to contain error message, got '%s'", view)
	}
	if !strings.Contains(view, "✖") {
		t.Errorf("expected view to contain error icon, got '%s'", view)
	}
}

func TestSpinnerModel_View_Quitting(t *testing.T) {
	action := func() error { return nil }
	m := initialSpinnerModel("Testing", action)
	m.quitting = true

	view := m.View()

	if view != "" {
		t.Errorf("expected empty view when quitting, got '%s'", view)
	}
}
