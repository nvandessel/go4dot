package dashboard

import (
	"strings"
	"testing"
)

func TestHeader_View(t *testing.T) {
	h := NewHeader(State{})
	view := h.View()
	expectedTitle := "GO4DOT DASHBOARD"
	if !strings.Contains(view, expectedTitle) {
		t.Errorf("expected view to contain '%s', but it didn't", expectedTitle)
	}
}
