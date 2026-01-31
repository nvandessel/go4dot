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
		name     string
		input    struct {
			initialIdx int
			key        rune
		}
		expected struct {
			selectedIdx int
		}
	}{
		{
			name: "move down from start",
			input: struct {
				initialIdx int
				key        rune
			}{
				initialIdx: 0,
				key:        'j',
			},
			expected: struct {
				selectedIdx int
			}{
				selectedIdx: 1,
			},
		},
		{
			name: "move up from middle",
			input: struct {
				initialIdx int
				key        rune
			}{
				initialIdx: 1,
				key:        'k',
			},
			expected: struct {
				selectedIdx int
			}{
				selectedIdx: 0,
			},
		},
		{
			name: "stay at top when pressing up",
			input: struct {
				initialIdx int
				key        rune
			}{
				initialIdx: 0,
				key:        'k',
			},
			expected: struct {
				selectedIdx int
			}{
				selectedIdx: 0,
			},
		},
		{
			name: "stay at bottom when pressing down",
			input: struct {
				initialIdx int
				key        rune
			}{
				initialIdx: 2,
				key:        'j',
			},
			expected: struct {
				selectedIdx int
			}{
				selectedIdx: 2,
			},
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

			s := NewSidebar(initialState, initialSelected)
			s.selectedIdx = tt.input.initialIdx
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.input.key}}
			s.Update(msg)

			if s.selectedIdx != tt.expected.selectedIdx {
				t.Errorf("expected selected index to be %d, but got %d", tt.expected.selectedIdx, s.selectedIdx)
			}
		})
	}
}

func TestSidebar_MouseScrolling(t *testing.T) {
	initialState := State{
		Configs: make([]config.ConfigItem, 20),
	}
	for i := 0; i < 20; i++ {
		initialState.Configs[i] = config.ConfigItem{Name: strconv.Itoa(i)}
	}

	tests := []struct {
		name     string
		input    struct {
			height      int
			listOffset  int
			mouseButton tea.MouseButton
		}
		expected struct {
			listOffset int
		}
	}{
		{
			name: "wheel up from offset 5",
			input: struct {
				height      int
				listOffset  int
				mouseButton tea.MouseButton
			}{
				height:      10,
				listOffset:  5,
				mouseButton: tea.MouseButtonWheelUp,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 4,
			},
		},
		{
			name: "wheel up at top stays at 0",
			input: struct {
				height      int
				listOffset  int
				mouseButton tea.MouseButton
			}{
				height:      10,
				listOffset:  0,
				mouseButton: tea.MouseButtonWheelUp,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 0,
			},
		},
		{
			name: "wheel down from offset 0",
			input: struct {
				height      int
				listOffset  int
				mouseButton tea.MouseButton
			}{
				height:      10,
				listOffset:  0,
				mouseButton: tea.MouseButtonWheelDown,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 1,
			},
		},
		{
			name: "wheel down at max stays at max",
			input: struct {
				height      int
				listOffset  int
				mouseButton tea.MouseButton
			}{
				height:      10,
				listOffset:  10,
				mouseButton: tea.MouseButtonWheelDown,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSidebar(initialState, make(map[string]bool))
			s.height = tt.input.height
			s.listOffset = tt.input.listOffset

			msg := tea.MouseMsg{
				Button: tt.input.mouseButton,
				Action: tea.MouseActionPress,
				X:      10,
				Y:      5,
			}
			s.Update(msg)

			if s.listOffset != tt.expected.listOffset {
				t.Errorf("expected list offset to be %d, but got %d", tt.expected.listOffset, s.listOffset)
			}
		})
	}
}

func TestSidebar_ensureVisible(t *testing.T) {
	initialState := State{
		Configs: make([]config.ConfigItem, 20),
	}
	for i := 0; i < 20; i++ {
		initialState.Configs[i] = config.ConfigItem{Name: strconv.Itoa(i)}
	}

	tests := []struct {
		name     string
		input    struct {
			height      int
			selectedIdx int
			listOffset  int
		}
		expected struct {
			listOffset int
		}
	}{
		{
			name: "selection within view, no scroll",
			input: struct {
				height      int
				selectedIdx int
				listOffset  int
			}{
				height:      10,
				selectedIdx: 5,
				listOffset:  0,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 0,
			},
		},
		{
			name: "scroll down when selection goes below view",
			input: struct {
				height      int
				selectedIdx int
				listOffset  int
			}{
				height:      10,
				selectedIdx: 10,
				listOffset:  0,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 1,
			},
		},
		{
			name: "scroll up when selection goes above view",
			input: struct {
				height      int
				selectedIdx int
				listOffset  int
			}{
				height:      10,
				selectedIdx: 4,
				listOffset:  5,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 4,
			},
		},
		{
			name: "jump to bottom",
			input: struct {
				height      int
				selectedIdx int
				listOffset  int
			}{
				height:      10,
				selectedIdx: 19,
				listOffset:  0,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 10,
			},
		},
		{
			name: "jump to top",
			input: struct {
				height      int
				selectedIdx int
				listOffset  int
			}{
				height:      10,
				selectedIdx: 0,
				listOffset:  10,
			},
			expected: struct {
				listOffset int
			}{
				listOffset: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSidebar(initialState, make(map[string]bool))
			s.height = tt.input.height
			s.selectedIdx = tt.input.selectedIdx
			s.listOffset = tt.input.listOffset
			s.ensureVisible()

			if s.listOffset != tt.expected.listOffset {
				t.Errorf("expected list offset to be %d, but got %d", tt.expected.listOffset, s.listOffset)
			}
		})
	}
}
