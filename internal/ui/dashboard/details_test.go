package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

func TestDetails_View(t *testing.T) {
	state := State{
		Configs: []config.ConfigItem{
			{
				Name:        "test-config",
				Description: "A test configuration.",
			},
		},
	}
	d := NewDetails(state)
	// Must call SetSize to initialize the viewport before View returns content
	d.SetSize(80, 24)
	view := d.View()
	if !strings.Contains(view, "TEST-CONFIG") {
		t.Errorf("expected view to contain 'TEST-CONFIG'")
	}
	if !strings.Contains(view, "A test configuration.") {
		t.Errorf("expected view to contain 'A test configuration.'")
	}
}
