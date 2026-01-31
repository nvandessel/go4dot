package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

func TestFormView_New(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)

	if fv.ID() != "test-form" {
		t.Errorf("expected ID 'test-form', got '%s'", fv.ID())
	}

	if fv.IsFinished() {
		t.Error("new form should not be finished")
	}

	if fv.IsCanceled() {
		t.Error("new form should not be canceled")
	}
}

func TestFormView_SetSize(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	fv.SetSize(80, 24)

	if fv.width != 80 {
		t.Errorf("expected width 80, got %d", fv.width)
	}

	if fv.height != 24 {
		t.Errorf("expected height 24, got %d", fv.height)
	}
}

func TestFormView_View(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	fv.SetSize(80, 24)

	view := fv.View()

	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestFormView_ViewTooSmall(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	fv.SetSize(5, 3) // Too small

	view := fv.View()

	if view != "" {
		t.Error("expected empty view for small size")
	}
}

func TestFormViewOverlay_SetSize(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	overlay := NewFormViewOverlay(fv)
	overlay.SetSize(100, 40)

	if overlay.width != 100 {
		t.Errorf("expected width 100, got %d", overlay.width)
	}

	if overlay.height != 40 {
		t.Errorf("expected height 40, got %d", overlay.height)
	}

	// Form should be sized proportionally
	if fv.width == 0 || fv.height == 0 {
		t.Error("form should have non-zero dimensions after overlay resize")
	}
}

func TestFormViewOverlay_View(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	overlay := NewFormViewOverlay(fv)
	overlay.SetSize(100, 40)

	view := overlay.View()

	if view == "" {
		t.Error("expected non-empty overlay view")
	}
}

func TestFormViewOverlay_FormView(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	overlay := NewFormViewOverlay(fv)

	if overlay.FormView() != fv {
		t.Error("FormView() should return the underlying FormView")
	}
}

func TestFormViewOverlay_Update_WindowSize(t *testing.T) {
	var value string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Test Input").
				Value(&value),
		),
	)

	fv := NewFormView("test-form", "Test Form", form)
	overlay := NewFormViewOverlay(fv)

	msg := tea.WindowSizeMsg{Width: 120, Height: 50}
	_, _ = overlay.Update(msg)

	if overlay.width != 120 {
		t.Errorf("expected width 120 after update, got %d", overlay.width)
	}

	if overlay.height != 50 {
		t.Errorf("expected height 50 after update, got %d", overlay.height)
	}
}
