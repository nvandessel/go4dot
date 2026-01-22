package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/stow"
)

func TestSummary_View(t *testing.T) {
	tests := []struct {
		name         string
		state        State
		expectedText string
	}{
		{
			name: "Initial state, no baseline",
			state: State{
				HasBaseline: false,
			},
			expectedText: "Not synced yet",
		},
		{
			name: "Baseline, no drift",
			state: State{
				HasBaseline:  true,
				DriftSummary: &stow.DriftSummary{},
			},
			expectedText: "All synced",
		},
		{
			name: "Baseline, with drift",
			state: State{
				HasBaseline: true,
				DriftSummary: &stow.DriftSummary{
					DriftedConfigs: 2,
					Results: []stow.DriftResult{
						{HasDrift: true},
						{HasDrift: true},
					},
				},
			},
			expectedText: "2 config(s) need syncing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSummary(tt.state)
			view := s.View()
			if !strings.Contains(view, tt.expectedText) {
				t.Errorf("expected view to contain '%s', but got '%s'", tt.expectedText, view)
			}
		})
	}
}
