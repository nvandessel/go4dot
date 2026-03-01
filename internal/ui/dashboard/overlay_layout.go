package dashboard

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

func overlayContentSize(width, height int, style ui.OverlayStyle) (int, int) {
	// Apply percentage constraints before subtracting chrome.
	w, h := ui.ClampToConstraints(width, height, style)

	border := 0
	if style.BorderStyle != (lipgloss.Border{}) {
		border = 1
	}

	contentWidth := w - 2*(border+style.PaddingH)
	contentHeight := h - 2*(border+style.PaddingV)

	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}

	return contentWidth, contentHeight
}
