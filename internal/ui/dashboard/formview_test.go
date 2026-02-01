package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func createTestForm() (*huh.Form, *string) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)
	return form, &value
}

func TestFormView_New(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			id    string
			title string
		}
		expected struct {
			id         string
			isFinished bool
			isCanceled bool
		}
	}{
		{
			name: "creates form view with correct ID",
			input: struct {
				id    string
				title string
			}{
				id:    "test-form",
				title: "Test Form",
			},
			expected: struct {
				id         string
				isFinished bool
				isCanceled bool
			}{
				id:         "test-form",
				isFinished: false,
				isCanceled: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form, _ := createTestForm()
			fv := NewFormView(tt.input.id, tt.input.title, form)

			if fv.ID() != tt.expected.id {
				t.Errorf("expected ID %q, got %q", tt.expected.id, fv.ID())
			}
			if fv.IsFinished() != tt.expected.isFinished {
				t.Errorf("expected IsFinished=%v, got %v", tt.expected.isFinished, fv.IsFinished())
			}
			if fv.IsCanceled() != tt.expected.isCanceled {
				t.Errorf("expected IsCanceled=%v, got %v", tt.expected.isCanceled, fv.IsCanceled())
			}
		})
	}
}

func TestFormView_SetSize(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			width  int
			height int
		}
		expected struct {
			width  int
			height int
		}
	}{
		{
			name: "sets normal size",
			input: struct {
				width  int
				height int
			}{
				width:  80,
				height: 24,
			},
			expected: struct {
				width  int
				height int
			}{
				width:  80,
				height: 24,
			},
		},
		{
			name: "handles very small size",
			input: struct {
				width  int
				height int
			}{
				width:  5,
				height: 3,
			},
			expected: struct {
				width  int
				height int
			}{
				width:  5,
				height: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form, _ := createTestForm()
			fv := NewFormView("test", "Test", form)
			fv.SetSize(tt.input.width, tt.input.height)

			if fv.width != tt.expected.width {
				t.Errorf("expected width %d, got %d", tt.expected.width, fv.width)
			}
			if fv.height != tt.expected.height {
				t.Errorf("expected height %d, got %d", tt.expected.height, fv.height)
			}
		})
	}
}

func TestFormView_View(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			width  int
			height int
		}
		expected struct {
			isEmpty bool
		}
	}{
		{
			name: "renders content with normal size",
			input: struct {
				width  int
				height int
			}{
				width:  80,
				height: 24,
			},
			expected: struct {
				isEmpty bool
			}{
				isEmpty: false,
			},
		},
		{
			name: "returns empty for too small size",
			input: struct {
				width  int
				height int
			}{
				width:  5,
				height: 3,
			},
			expected: struct {
				isEmpty bool
			}{
				isEmpty: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form, _ := createTestForm()
			fv := NewFormView("test", "Test", form)
			fv.SetSize(tt.input.width, tt.input.height)

			view := fv.View()

			if tt.expected.isEmpty && view != "" {
				t.Error("expected empty view")
			}
			if !tt.expected.isEmpty && view == "" {
				t.Error("expected non-empty view")
			}
		})
	}
}

func TestFormViewOverlay(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			width  int
			height int
		}
		expected struct {
			width           int
			height          int
			hasNonEmptyView bool
		}
	}{
		{
			name: "sets overlay size correctly",
			input: struct {
				width  int
				height int
			}{
				width:  100,
				height: 40,
			},
			expected: struct {
				width           int
				height          int
				hasNonEmptyView bool
			}{
				width:           100,
				height:          40,
				hasNonEmptyView: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form, _ := createTestForm()
			fv := NewFormView("test", "Test", form)
			overlay := NewFormViewOverlay(fv)
			overlay.SetSize(tt.input.width, tt.input.height)

			if overlay.width != tt.expected.width {
				t.Errorf("expected width %d, got %d", tt.expected.width, overlay.width)
			}
			if overlay.height != tt.expected.height {
				t.Errorf("expected height %d, got %d", tt.expected.height, overlay.height)
			}

			// Form should have non-zero dimensions
			if fv.width == 0 || fv.height == 0 {
				t.Error("form should have non-zero dimensions after overlay resize")
			}

			// Verify FormView() returns correct underlying view
			if overlay.FormView() != fv {
				t.Error("FormView() should return the underlying FormView")
			}

			view := overlay.View()
			if tt.expected.hasNonEmptyView && view == "" {
				t.Error("expected non-empty overlay view")
			}
		})
	}
}

func TestFormViewOverlay_Update(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			width  int
			height int
		}
		expected struct {
			width  int
			height int
		}
	}{
		{
			name: "updates size on window size message",
			input: struct {
				width  int
				height int
			}{
				width:  120,
				height: 50,
			},
			expected: struct {
				width  int
				height int
			}{
				width:  120,
				height: 50,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form, _ := createTestForm()
			fv := NewFormView("test", "Test", form)
			overlay := NewFormViewOverlay(fv)

			msg := tea.WindowSizeMsg{Width: tt.input.width, Height: tt.input.height}
			_, _ = overlay.Update(msg)

			if overlay.width != tt.expected.width {
				t.Errorf("expected width %d after update, got %d", tt.expected.width, overlay.width)
			}
			if overlay.height != tt.expected.height {
				t.Errorf("expected height %d after update, got %d", tt.expected.height, overlay.height)
			}
		})
	}
}
