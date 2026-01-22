package dashboard

import (
	"strings"
	"testing"
)

func TestHelp_View(t *testing.T) {
	h := NewHelp()
	view := h.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Errorf("expected view to contain 'Keyboard Shortcuts'")
	}
}
