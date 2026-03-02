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
			name: "SetSize clamps to compact bounds",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(120, 50)
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				if m.width != menuMaxWidth {
					t.Errorf("expected width %d, got %d", menuMaxWidth, m.width)
				}
				if m.height != menuCompactHeight {
					t.Errorf("expected height %d, got %d", menuCompactHeight, m.height)
				}
			},
		},
		{
			name: "SetSize respects small terminal width",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(30, 50)
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				expectedWidth := 20 // 30 - 10
				if m.width != expectedWidth {
					t.Errorf("expected width %d, got %d", expectedWidth, m.width)
				}
			},
		},
		{
			name: "SetSize respects small terminal height",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(120, 15)
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				expectedHeight := 6 // 15 - 10 = 5, but min is 6
				if m.height != expectedHeight {
					t.Errorf("expected height %d, got %d", expectedHeight, m.height)
				}
			},
		},
		{
			name: "SetSize enforces minimum width",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(15, 50) // 15 - 10 = 5, clamped to 20
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				if m.width < 20 {
					t.Errorf("expected width >= 20, got %d", m.width)
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
		{
			name: "View contains More Commands title",
			setup: func() *Menu {
				m := NewMenu()
				m.SetSize(120, 50)
				return &m
			},
			assert: func(t *testing.T, m *Menu) {
				view := m.View()
				if !strings.Contains(view, "More Commands") {
					t.Error("expected view to contain 'More Commands' title")
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

func TestCompactWidth(t *testing.T) {
	tests := []struct {
		name      string
		termWidth int
		expected  int
	}{
		{name: "large terminal", termWidth: 200, expected: menuMaxWidth},
		{name: "exact boundary", termWidth: menuMaxWidth + 10, expected: menuMaxWidth},
		{name: "slightly small", termWidth: menuMaxWidth + 5, expected: menuMaxWidth + 5 - 10},
		{name: "very small", termWidth: 25, expected: 20}, // 25 - 10 = 15, clamped to 20
		{name: "tiny terminal", termWidth: 10, expected: 20},
		{name: "zero width", termWidth: 0, expected: menuMaxWidth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompactWidth(tt.termWidth)
			if result != tt.expected {
				t.Errorf("CompactWidth(%d) = %d, want %d", tt.termWidth, result, tt.expected)
			}
		})
	}
}

func TestCompactHeight(t *testing.T) {
	tests := []struct {
		name       string
		termHeight int
		expected   int
	}{
		{name: "large terminal", termHeight: 100, expected: menuCompactHeight},
		{name: "exact boundary", termHeight: menuCompactHeight + 10, expected: menuCompactHeight},
		{name: "slightly small", termHeight: menuCompactHeight + 5, expected: menuCompactHeight + 5 - 10},
		{name: "very small", termHeight: 12, expected: 6}, // 12 - 10 = 2, clamped to 6
		{name: "tiny terminal", termHeight: 5, expected: 6},
		{name: "zero height", termHeight: 0, expected: menuCompactHeight},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompactHeight(tt.termHeight)
			if result != tt.expected {
				t.Errorf("CompactHeight(%d) = %d, want %d", tt.termHeight, result, tt.expected)
			}
		})
	}
}
