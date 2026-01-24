package dashboard

import (
	"strings"
	"testing"
)

func TestNoConfig_View(t *testing.T) {
	nc := NewNoConfig()
	view := nc.View()
	if !strings.Contains(view, "No configuration found") {
		t.Errorf("expected view to contain 'No configuration found'")
	}
}
