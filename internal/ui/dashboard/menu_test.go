package dashboard

import (
	"strings"
	"testing"
)

func TestMenu(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *Menu
		assert func(t *testing.T, m *Menu)
	}{
		{
			name: "View contains navigation hint",
			setup: func() *Menu {
				m := NewMenu()
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				view := m.View()
				if !strings.Contains(view, "↑/k up") {
					t.Errorf("expected view to contain '↑/k up'")
				}
			},
		},
		{
			name: "Initial size is 0x0",
			setup: func() *Menu {
				m := NewMenu()
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				if m.width != 0 || m.height != 0 {
					t.Errorf("expected initial size 0x0, got %dx%d", m.width, m.height)
				}
			},
		},
		{
			name: "SetSize updates dimensions",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(120, 50)
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				if m.width != 120 {
					t.Errorf("expected width 120, got %d", m.width)
				}
				if m.height != 50 {
					t.Errorf("expected height 50, got %d", m.height)
				}
			},
		},
		{
			name: "View non-empty after SetSize",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(120, 50)
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				view := m.View()
				if view == "" {
					t.Error("expected non-empty view after SetSize")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			tt.assert(t, m)
		})
	}
}
