package dashboard

import (
	"strings"
	"testing"
)

func TestFooter_View(t *testing.T) {
	f := NewFooter()
	view := f.View()
	// The default view should at least have the help key
	if !strings.Contains(view, "?") {
		t.Errorf("expected view to contain '?' for help")
	}
}
