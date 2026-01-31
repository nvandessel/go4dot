package dashboard

import (
	"strconv"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
)

func TestSidebar_Update_Navigation(t *testing.T) {
	initialState := State{
		Configs: []config.ConfigItem{
			{Name: "one"},
			{Name: "two"},
			{Name: "three"},
		},
	}
	initialSelected := make(map[string]bool)

	tests := []struct {
		name                string
		initialSidebar      Sidebar
		msg                 tea.Msg
		expectedSelectedIdx int
	}{
		{
			name:                "Move Down",
			initialSidebar:      NewSidebar(initialState, initialSelected),
			msg:                 tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}, // Assuming 'j' is Down
			expectedSelectedIdx: 1,
		},
		{
			name: "Move Up",
			initialSidebar: func() Sidebar {
				s := NewSidebar(initialState, initialSelected)
				s.selectedIdx = 1
				return s
			}(),
			msg:                 tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, // Assuming 'k' is Up
			expectedSelectedIdx: 0,
		},
		{
			name:                "Move Down to End",
			initialSidebar:      NewSidebar(initialState, initialSelected),
			msg:                 tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			expectedSelectedIdx: 1, // First move
		},
		{
			name: "Move Up to Top",
			initialSidebar: func() Sidebar {
				s := NewSidebar(initialState, initialSelected)
				s.selectedIdx = 1
				return s
			}(),
			msg:                 tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}},
			expectedSelectedIdx: 0, // First move
		},
		{
			name: "Stay at Top",
			initialSidebar: func() Sidebar {
				s := NewSidebar(initialState, initialSelected)
				s.selectedIdx = 0
				return s
			}(),
			msg:                 tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}},
			expectedSelectedIdx: 0,
		},
		{
			name: "Stay at Bottom",
			initialSidebar: func() Sidebar {
				s := NewSidebar(initialState, initialSelected)
				s.selectedIdx = 2
				return s
			}(),
			msg:                 tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}},
			expectedSelectedIdx: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override default keys for testing
			originalUp := keys.Up.Keys()
			originalDown := keys.Down.Keys()
			keys.Up.SetKeys("k")
			keys.Down.SetKeys("j")
			defer func() {
				keys.Up.SetKeys(originalUp...)
				keys.Down.SetKeys(originalDown...)
			}()

			s := tt.initialSidebar
			s.Update(tt.msg)
			if s.selectedIdx != tt.expectedSelectedIdx {
				t.Errorf("expected selected index to be %d, but got %d", tt.expectedSelectedIdx, s.selectedIdx)
			}
		})
	}
}

func TestSidebar_MouseScrolling(t *testing.T) {
	initialState := State{
		Configs: make([]config.ConfigItem, 20), // 20 items
	}
	for i := 0; i < 20; i++ {
		initialState.Configs[i] = config.ConfigItem{Name: strconv.Itoa(i)}
	}

	tests := []struct {
		name               string
		height             int
		initialListOffset  int
		mouseButton        tea.MouseButton
		expectedListOffset int
	}{
		{
			name:               "Wheel up from offset 5",
			height:             10,
			initialListOffset:  5,
			mouseButton:        tea.MouseButtonWheelUp,
			expectedListOffset: 4,
		},
		{
			name:               "Wheel up at top stays at 0",
			height:             10,
			initialListOffset:  0,
			mouseButton:        tea.MouseButtonWheelUp,
			expectedListOffset: 0,
		},
		{
			name:               "Wheel down from offset 0",
			height:             10,
			initialListOffset:  0,
			mouseButton:        tea.MouseButtonWheelDown,
			expectedListOffset: 1,
		},
		{
			name:               "Wheel down at max stays at max",
			height:             10,
			initialListOffset:  10, // max offset for 20 items with height 10
			mouseButton:        tea.MouseButtonWheelDown,
			expectedListOffset: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSidebar(initialState, make(map[string]bool))
			s.height = tt.height
			s.listOffset = tt.initialListOffset

			msg := tea.MouseMsg{
				Button: tt.mouseButton,
				Action: tea.MouseActionPress,
				X:      10,
				Y:      5,
			}
			s.Update(msg)

			if s.listOffset != tt.expectedListOffset {
				t.Errorf("expected list offset to be %d, but got %d", tt.expectedListOffset, s.listOffset)
			}
		})
	}
}

func TestSidebar_ensureVisible(t *testing.T) {
	initialState := State{
		Configs: make([]config.ConfigItem, 20), // 20 items
	}
	for i := 0; i < 20; i++ {
		initialState.Configs[i] = config.ConfigItem{Name: strconv.Itoa(i)}
	}

	tests := []struct {
		name               string
		height             int
		selectedIdx        int
		initialListOffset  int
		expectedListOffset int
	}{
		{
			name:               "Selection within view, no scroll",
			height:             10,
			selectedIdx:        5,
			initialListOffset:  0,
			expectedListOffset: 0,
		},
		{
			name:               "Scroll down when selection goes below view",
			height:             10,
			selectedIdx:        10,
			initialListOffset:  0,
			expectedListOffset: 1, // 10 - 10 + 1
		},
		{
			name:               "Scroll up when selection goes above view",
			height:             10,
			selectedIdx:        4,
			initialListOffset:  5,
			expectedListOffset: 4,
		},
		{
			name:               "Jump to bottom",
			height:             10,
			selectedIdx:        19,
			initialListOffset:  0,
			expectedListOffset: 10, // 19 - 10 + 1
		},
		{
			name:               "Jump to top",
			height:             10,
			selectedIdx:        0,
			initialListOffset:  10,
			expectedListOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSidebar(initialState, make(map[string]bool))
			s.height = tt.height
			s.selectedIdx = tt.selectedIdx
			s.listOffset = tt.initialListOffset
			s.ensureVisible()
			if s.listOffset != tt.expectedListOffset {
				t.Errorf("expected list offset to be %d, but got %d", tt.expectedListOffset, s.listOffset)
			}
		})
	}
}
