package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestConfirm_New(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			id          string
			title       string
			description string
		}
		expected struct {
			id       string
			title    string
			selected int
		}
	}{
		{
			name: "creates confirm with correct ID and title",
			input: struct {
				id          string
				title       string
				description string
			}{
				id:          "test-id",
				title:       "Test Title",
				description: "Test description",
			},
			expected: struct {
				id       string
				title    string
				selected int
			}{
				id:       "test-id",
				title:    "Test Title",
				selected: 1, // Default to "No"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfirm(tt.input.id, tt.input.title, tt.input.description)

			if c.ID() != tt.expected.id {
				t.Errorf("expected ID %q, got %q", tt.expected.id, c.ID())
			}
			if c.title != tt.expected.title {
				t.Errorf("expected title %q, got %q", tt.expected.title, c.title)
			}
			if c.selected != tt.expected.selected {
				t.Errorf("expected selected %d, got %d", tt.expected.selected, c.selected)
			}
		})
	}
}

func TestConfirm_WithLabels(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			affirmative string
			negative    string
		}
		expected struct {
			affirmative string
			negative    string
		}
	}{
		{
			name: "sets custom labels",
			input: struct {
				affirmative string
				negative    string
			}{
				affirmative: "Yes, do it",
				negative:    "Cancel",
			},
			expected: struct {
				affirmative string
				negative    string
			}{
				affirmative: "Yes, do it",
				negative:    "Cancel",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfirm("test", "Title", "Desc").WithLabels(tt.input.affirmative, tt.input.negative)

			if c.affirmative != tt.expected.affirmative {
				t.Errorf("expected affirmative %q, got %q", tt.expected.affirmative, c.affirmative)
			}
			if c.negative != tt.expected.negative {
				t.Errorf("expected negative %q, got %q", tt.expected.negative, c.negative)
			}
		})
	}
}

func TestConfirm_KeyHandling(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			initialSelected int
			key             tea.KeyMsg
		}
		expected struct {
			selected      int
			hasResult     bool
			resultConfirm bool
		}
	}{
		{
			name: "left key selects Yes",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 1,
				key:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:  0,
				hasResult: false,
			},
		},
		{
			name: "right key selects No",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 0,
				key:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:  1,
				hasResult: false,
			},
		},
		{
			name: "tab toggles selection",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 1,
				key:             tea.KeyMsg{Type: tea.KeyTab},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:  0,
				hasResult: false,
			},
		},
		{
			name: "y key confirms",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 1,
				key:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:      1,
				hasResult:     true,
				resultConfirm: true,
			},
		},
		{
			name: "n key denies",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 0,
				key:             tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:      0,
				hasResult:     true,
				resultConfirm: false,
			},
		},
		{
			name: "enter confirms when Yes selected",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 0,
				key:             tea.KeyMsg{Type: tea.KeyEnter},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:      0,
				hasResult:     true,
				resultConfirm: true,
			},
		},
		{
			name: "enter denies when No selected",
			input: struct {
				initialSelected int
				key             tea.KeyMsg
			}{
				initialSelected: 1,
				key:             tea.KeyMsg{Type: tea.KeyEnter},
			},
			expected: struct {
				selected      int
				hasResult     bool
				resultConfirm bool
			}{
				selected:      1,
				hasResult:     true,
				resultConfirm: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfirm("test", "Title", "Desc")
			c.SetSize(80, 24)
			c.selected = tt.input.initialSelected

			model, cmd := c.Update(tt.input.key)
			confirm := model.(*Confirm)

			if !tt.expected.hasResult {
				if confirm.selected != tt.expected.selected {
					t.Errorf("expected selected %d, got %d", tt.expected.selected, confirm.selected)
				}
			}

			if tt.expected.hasResult {
				if cmd == nil {
					t.Fatal("expected command for result")
				}
				msg := cmd()
				result, ok := msg.(ConfirmResult)
				if !ok {
					t.Fatal("expected ConfirmResult message")
				}
				if result.Confirmed != tt.expected.resultConfirm {
					t.Errorf("expected Confirmed=%v, got %v", tt.expected.resultConfirm, result.Confirmed)
				}
			}
		})
	}
}

func TestConfirm_View(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			title       string
			description string
			width       int
			height      int
		}
		expected struct {
			containsTitle bool
			containsDesc  bool
		}
	}{
		{
			name: "renders title and description",
			input: struct {
				title       string
				description string
				width       int
				height      int
			}{
				title:       "Test Title",
				description: "Test description",
				width:       80,
				height:      24,
			},
			expected: struct {
				containsTitle bool
				containsDesc  bool
			}{
				containsTitle: true,
				containsDesc:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfirm("test", tt.input.title, tt.input.description)
			c.SetSize(tt.input.width, tt.input.height)
			view := c.View()

			if tt.expected.containsTitle && !strings.Contains(view, tt.input.title) {
				t.Errorf("expected view to contain title %q", tt.input.title)
			}
			if tt.expected.containsDesc && !strings.Contains(view, tt.input.description) {
				t.Errorf("expected view to contain description %q", tt.input.description)
			}
		})
	}
}
