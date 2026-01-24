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
