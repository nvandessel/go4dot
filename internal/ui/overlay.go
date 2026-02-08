package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
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
// Colors are drawn from the Catppuccin Mocha palette to match the
// purple-accented theme used throughout the dashboard.
func DefaultOverlayStyle() OverlayStyle {
	return OverlayStyle{
		BorderStyle: lipgloss.RoundedBorder(),
		BorderColor: PrimaryColor,
		PaddingH:    2,
		PaddingV:    1,
		Background:  lipgloss.Color("#1e1e2e"), // Catppuccin Mocha Base
		DimChar:     " ",
		DimColor:    lipgloss.Color("#45475a"), // Catppuccin Mocha Surface1
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

	// Apply background fill to the modal content. ANSI-styled text contains
	// reset codes (\x1b[0m) that kill any outer Background() applied by
	// lipgloss, leaving black gaps. We inject the background color after
	// every reset and pad lines to uniform width.
	modal = fillBackground(modal, style.Background)

	// Style the modal with border and padding
	modalStyle := lipgloss.NewStyle().
		Border(style.BorderStyle).
		BorderForeground(style.BorderColor).
		Padding(style.PaddingV, style.PaddingH).
		Background(style.Background)

	styledModal := modalStyle.Render(modal)

	// Composite the modal centered over the dimmed background
	return placeOverlay(dimmedBg, styledModal, width, height, style.DimColor)
}

// fillBackground applies a background color uniformly to content that contains
// pre-styled ANSI text. It injects the background after every ANSI reset so the
// color persists, and pads all lines to the same width.
func fillBackground(content string, bg lipgloss.Color) string {
	lines := strings.Split(content, "\n")

	// Inject background after every ANSI reset within each line so the
	// background persists through styled content. This is a no-op when
	// no terminal is attached (bgSeq is empty).
	bgSeq := colorToANSIBg(bg)
	if bgSeq != "" {
		for i, line := range lines {
			line = strings.ReplaceAll(line, "\x1b[0m", "\x1b[0m"+bgSeq)
			line = strings.ReplaceAll(line, "\x1b[m", "\x1b[m"+bgSeq)
			lines[i] = bgSeq + line
		}
	}

	// Find max visual width and pad shorter lines to uniform width
	maxWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			maxWidth = w
		}
	}

	if maxWidth == 0 {
		return strings.Join(lines, "\n")
	}

	bgStyle := lipgloss.NewStyle().Background(bg)
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < maxWidth {
			lines[i] = line + bgStyle.Render(strings.Repeat(" ", maxWidth-w))
		}
	}

	return strings.Join(lines, "\n")
}

// colorToANSIBg extracts the ANSI background escape sequence that lipgloss
// would produce for a given color. Using lipgloss's own rendering ensures
// the sequence matches the terminal's color profile (24-bit, 256-color, etc.).
func colorToANSIBg(c lipgloss.Color) string {
	rendered := lipgloss.NewStyle().Background(c).Render(" ")
	// The rendered string is: <ANSI_bg_sequence> <space> <ANSI_reset>
	// Extract everything before the first space character.
	idx := strings.Index(rendered, " ")
	if idx > 0 {
		return rendered[:idx]
	}
	return ""
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
func placeOverlay(bg, modal string, width, height int, dimColor lipgloss.Color) string {
	bgLines := strings.Split(bg, "\n")
	modalLines := strings.Split(modal, "\n")
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

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
		modalLineWidth := lipgloss.Width(modalLine)

		bgPlain := stripAnsi(bgLine)
		beforePlain := sliceByCells(bgPlain, 0, startX)
		afterPlain := sliceByCells(bgPlain, startX+modalLineWidth, width-(startX+modalLineWidth))

		before := ""
		if startX > 0 {
			before = dimStyle.Render(beforePlain)
		}

		after := ""
		if afterPlain != "" {
			after = dimStyle.Render(afterPlain)
		}

		bgLines[bgIdx] = before + modalLine + after
	}

	return strings.Join(bgLines[:height], "\n")
}

func sliceByCells(s string, start, length int) string {
	if length <= 0 {
		return ""
	}

	end := start + length
	col := 0
	var b strings.Builder
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if w < 1 {
			w = 1
		}
		runeStart := col
		runeEnd := col + w

		if runeEnd <= start {
			col = runeEnd
			continue
		}
		if runeStart >= end {
			break
		}

		if runeStart >= start && runeEnd <= end {
			b.WriteRune(r)
		} else {
			overlapStart := max(runeStart, start)
			overlapEnd := min(runeEnd, end)
			if overlapEnd > overlapStart {
				b.WriteString(strings.Repeat(" ", overlapEnd-overlapStart))
			}
		}

		col = runeEnd
	}

	current := runewidth.StringWidth(b.String())
	if current < length {
		b.WriteString(strings.Repeat(" ", length-current))
	}

	return b.String()
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
