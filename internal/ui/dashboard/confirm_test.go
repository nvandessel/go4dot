package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirm_New(t *testing.T) {
	c := NewConfirm("test-id", "Test Title", "Test description")

	if c.ID() != "test-id" {
		t.Errorf("expected ID 'test-id', got '%s'", c.ID())
	}

	if c.title != "Test Title" {
		t.Errorf("expected title 'Test Title', got '%s'", c.title)
	}

	if c.selected != 1 {
		t.Error("expected default selection to be 'No' (1)")
	}
}

func TestConfirm_WithLabels(t *testing.T) {
	c := NewConfirm("test", "Title", "Desc").WithLabels("Yes, do it", "Cancel")

	if c.affirmative != "Yes, do it" {
		t.Errorf("expected affirmative 'Yes, do it', got '%s'", c.affirmative)
	}

	if c.negative != "Cancel" {
		t.Errorf("expected negative 'Cancel', got '%s'", c.negative)
	}
}

func TestConfirm_Navigation(t *testing.T) {
	c := NewConfirm("test", "Title", "Desc")
	c.SetSize(80, 24)

	// Default is "No" (selected = 1)
	if c.selected != 1 {
		t.Errorf("expected initial selection 1, got %d", c.selected)
	}

	// Press left to select "Yes"
	_, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if c.selected != 0 {
		t.Errorf("expected selection 0 after left, got %d", c.selected)
	}

	// Press right to select "No"
	_, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if c.selected != 1 {
		t.Errorf("expected selection 1 after right, got %d", c.selected)
	}

	// Press tab to toggle
	_, _ = c.Update(tea.KeyMsg{Type: tea.KeyTab})
	if c.selected != 0 {
		t.Errorf("expected selection 0 after tab, got %d", c.selected)
	}
}

func TestConfirm_YesKey(t *testing.T) {
	c := NewConfirm("test", "Title", "Desc")
	c.SetSize(80, 24)

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if cmd == nil {
		t.Fatal("expected command after 'y' key")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResult)
	if !ok {
		t.Fatal("expected ConfirmResult message")
	}

	if !result.Confirmed {
		t.Error("expected Confirmed to be true")
	}

	if result.ID != "test" {
		t.Errorf("expected ID 'test', got '%s'", result.ID)
	}
}

func TestConfirm_NoKey(t *testing.T) {
	c := NewConfirm("test", "Title", "Desc")
	c.SetSize(80, 24)

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	if cmd == nil {
		t.Fatal("expected command after 'n' key")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResult)
	if !ok {
		t.Fatal("expected ConfirmResult message")
	}

	if result.Confirmed {
		t.Error("expected Confirmed to be false")
	}
}

func TestConfirm_EnterKey(t *testing.T) {
	c := NewConfirm("test", "Title", "Desc")
	c.SetSize(80, 24)

	// Select "Yes" first
	c.selected = 0

	_, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("expected command after enter key")
	}

	msg := cmd()
	result, ok := msg.(ConfirmResult)
	if !ok {
		t.Fatal("expected ConfirmResult message")
	}

	if !result.Confirmed {
		t.Error("expected Confirmed to be true when Yes is selected")
	}
}

func TestConfirm_View(t *testing.T) {
	c := NewConfirm("test", "Test Title", "Test description")
	c.SetSize(80, 24)

	view := c.View()

	if !strings.Contains(view, "Test Title") {
		t.Error("expected view to contain title")
	}

	if !strings.Contains(view, "Test description") {
		t.Error("expected view to contain description")
	}
}
