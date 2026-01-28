package dashboard

import (
	"strings"
	"testing"
)

func TestMenu_View(t *testing.T) {
	m := NewMenu()
	view := m.View()
	if !strings.Contains(view, "↑/k up") {
		t.Errorf("expected view to contain '↑/k up'")
	}
}

func TestMenu_SetSize(t *testing.T) {
	m := NewMenu()

	// Initially 0x0
	if m.width != 0 || m.height != 0 {
		t.Errorf("expected initial size 0x0, got %dx%d", m.width, m.height)
	}

	// Set size
	m.SetSize(120, 50)

	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}
	if m.height != 50 {
		t.Errorf("expected height 50, got %d", m.height)
	}

	// View should now render properly with dimensions
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view after SetSize")
	}
}
