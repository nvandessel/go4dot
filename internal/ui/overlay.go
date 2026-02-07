package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// OverlayStyle defines the visual properties for a floating modal overlay.
type OverlayStyle struct {
	// Border style for the modal window
	BorderStyle lipgloss.Border
	// Border color
	BorderColor lipgloss.Color
	// Padding inside the modal
	PaddingH int
	PaddingV int
	// Background color for the modal content area
	Background lipgloss.Color
	// DimChar is the character used to fill the dimmed background
	DimChar string
	// DimColor is the foreground color for the dimmed background character
	DimColor lipgloss.Color
}

// DefaultOverlayStyle returns the standard floating modal style.
func DefaultOverlayStyle() OverlayStyle {
	return OverlayStyle{
		BorderStyle: lipgloss.RoundedBorder(),
		BorderColor: PrimaryColor,
		PaddingH:    2,
		PaddingV:    1,
		Background:  lipgloss.Color("#1a1a2e"),
		DimChar:     " ",
		DimColor:    lipgloss.Color("#333333"),
	}
}

// WarningOverlayStyle returns a floating modal style with warning-colored border.
func WarningOverlayStyle() OverlayStyle {
	s := DefaultOverlayStyle()
	s.BorderColor = WarningColor
	return s
}

// RenderOverlay composites a modal on top of a background view.
// The background is dimmed, and the modal content is centered with a styled frame.
func RenderOverlay(bg, modal string, width, height int, style OverlayStyle) string {
	if width < 10 || height < 5 {
		return modal
	}

	// Dim the background
	dimmedBg := dimContent(bg, width, height, style.DimChar, style.DimColor)

	// Style the modal with border, padding, background
	modalStyle := lipgloss.NewStyle().
		Border(style.BorderStyle).
		BorderForeground(style.BorderColor).
		Padding(style.PaddingV, style.PaddingH).
		Background(style.Background)

	styledModal := modalStyle.Render(modal)

	// Composite the modal centered over the dimmed background
	return placeOverlay(dimmedBg, styledModal, width, height)
}

// dimContent creates a dimmed version of the background.
func dimContent(content string, width, height int, dimChar string, dimColor lipgloss.Color) string {
	lines := strings.Split(content, "\n")
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	var result []string
	for i := 0; i < height; i++ {
		if i < len(lines) {
			dimmedLine := dimStyle.Render(stripAnsi(lines[i]))
			lineWidth := lipgloss.Width(dimmedLine)
			if lineWidth < width {
				padding := width - lineWidth
				if padding > 0 {
					dimmedLine += dimStyle.Render(strings.Repeat(dimChar, padding))
				}
			}
			result = append(result, dimmedLine)
		} else {
			result = append(result, dimStyle.Render(strings.Repeat(dimChar, width)))
		}
	}

	return strings.Join(result, "\n")
}

// placeOverlay places the modal content centered over the background.
func placeOverlay(bg, modal string, width, height int) string {
	bgLines := strings.Split(bg, "\n")
	modalLines := strings.Split(modal, "\n")

	modalWidth := 0
	for _, line := range modalLines {
		w := lipgloss.Width(line)
		if w > modalWidth {
			modalWidth = w
		}
	}
	modalHeight := len(modalLines)

	startX := (width - modalWidth) / 2
	startY := (height - modalHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	for len(bgLines) < height {
		bgLines = append(bgLines, strings.Repeat(" ", width))
	}

	for i, modalLine := range modalLines {
		bgIdx := startY + i
		if bgIdx >= len(bgLines) {
			break
		}

		bgLine := bgLines[bgIdx]
		bgRunes := []rune(stripAnsi(bgLine))

		for len(bgRunes) < width {
			bgRunes = append(bgRunes, ' ')
		}

		modalLineWidth := lipgloss.Width(modalLine)

		before := ""
		if startX > 0 {
			if startX <= len(bgRunes) {
				before = string(bgRunes[:startX])
			} else {
				before = string(bgRunes) + strings.Repeat(" ", startX-len(bgRunes))
			}
		}

		afterStart := startX + modalLineWidth
		after := ""
		if afterStart < len(bgRunes) {
			after = string(bgRunes[afterStart:])
		}

		bgLines[bgIdx] = before + modalLine + after
	}

	return strings.Join(bgLines[:height], "\n")
}

// stripAnsi removes ANSI escape sequences from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}

	return result.String()
}
