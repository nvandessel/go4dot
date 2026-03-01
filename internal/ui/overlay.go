package ui

import (
	"fmt"
	"strconv"
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

// dimColorMap maps Catppuccin Mocha foreground colors to their dimmed
// counterparts. Each bright color is halved in intensity so the dashboard
// structure remains recognizable while clearly receding behind the modal.
var dimColorMap = map[string]string{
	// Primary (Lavender)
	"#b4befe": "#585b7f",
	// Secondary (Green)
	"#a6e3a1": "#536f50",
	// Warning (Yellow)
	"#f9e2af": "#7c7157",
	// Error (Red)
	"#f38ba8": "#794554",
	// Text
	"#cdd6f4": "#666b7a",
	// Subtle (Overlay2)
	"#9399b2": "#494c59",
	// Subtext1
	"#bac2de": "#5d616f",
	// Subtext0
	"#a6adc8": "#535664",
	// Overlay1
	"#7f849c": "#3f424e",
	// Overlay0
	"#6c7086": "#363843",
	// Surface2
	"#585b70": "#2c2d38",
	// Rosewater
	"#f5e0dc": "#7a706e",
	// Flamingo
	"#f2cdcd": "#796666",
	// Pink
	"#f5c2e7": "#7a6173",
	// Mauve
	"#cba6f7": "#65537b",
	// Maroon
	"#eba0a1": "#755050",
	// Peach
	"#fab387": "#7d5943",
	// Teal
	"#94e2d5": "#4a716a",
	// Sky
	"#89dceb": "#446e75",
	// Sapphire
	"#74c7ec": "#3a6376",
	// Blue
	"#89b4fa": "#445a7d",
}

// dimContent creates a dimmed version of the background that preserves the
// visual structure of the dashboard. Rather than stripping all ANSI color
// codes and applying a flat dim color, it replaces foreground colors with
// dimmer versions from dimColorMap. This keeps borders, titles, and status
// indicators distinguishable behind the modal overlay.
func dimContent(content string, width, height int, dimChar string, dimColor lipgloss.Color) string {
	lines := strings.Split(content, "\n")
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	var result []string
	for i := 0; i < height; i++ {
		if i < len(lines) {
			dimmedLine := dimAnsiColors(lines[i], dimColor)
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

// dimAnsiColors replaces foreground color codes in an ANSI-styled string with
// their dimmed counterparts. It walks through the string character by character,
// identifies SGR (Select Graphic Rendition) escape sequences, and rewrites
// foreground color parameters while leaving other parameters (bold, background,
// etc.) intact. Text segments without any foreground color are rendered in the
// fallback dimColor.
func dimAnsiColors(s string, fallback lipgloss.Color) string {
	var result strings.Builder
	fallbackStyle := lipgloss.NewStyle().Foreground(fallback)
	var plainBuf strings.Builder

	i := 0
	runes := []rune(s)
	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Flush any accumulated plain text with fallback dim color
			if plainBuf.Len() > 0 {
				result.WriteString(fallbackStyle.Render(plainBuf.String()))
				plainBuf.Reset()
			}

			// Parse the full SGR escape sequence
			seq, end := parseEscapeSeq(runes, i)
			if end > i {
				dimmed := dimSGRSequence(seq, fallback)
				result.WriteString(dimmed)
				i = end
				continue
			}
		}

		// Check for ESC without [ (other escape types) -- skip them
		if runes[i] == '\x1b' {
			// Consume until we find a letter terminator
			j := i + 1
			for j < len(runes) {
				r := runes[j]
				j++
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' {
					break
				}
			}
			i = j
			continue
		}

		plainBuf.WriteRune(runes[i])
		i++
	}

	// Flush remaining plain text
	if plainBuf.Len() > 0 {
		result.WriteString(fallbackStyle.Render(plainBuf.String()))
	}

	return result.String()
}

// parseEscapeSeq extracts an SGR escape sequence starting at position pos.
// It returns the full sequence string (e.g., "\x1b[38;2;180;190;254m") and
// the index just past the end of the sequence. If the sequence cannot be
// parsed, it returns ("", pos) to signal that the caller should skip it.
func parseEscapeSeq(runes []rune, pos int) (string, int) {
	if pos+1 >= len(runes) || runes[pos] != '\x1b' || runes[pos+1] != '[' {
		return "", pos
	}

	var seq strings.Builder
	seq.WriteRune('\x1b')
	seq.WriteRune('[')

	j := pos + 2
	for j < len(runes) {
		r := runes[j]
		seq.WriteRune(r)
		j++
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '~' {
			// Only process SGR sequences (terminated with 'm')
			if r == 'm' {
				return seq.String(), j
			}
			// Non-SGR escape (cursor movement, etc.) -- return as-is
			return seq.String(), j
		}
	}

	return seq.String(), j
}

// dimSGRSequence takes an SGR escape sequence and rewrites any foreground
// color parameters to their dimmed equivalents. It handles:
//   - 24-bit true color: ESC[38;2;R;G;Bm
//   - 256-color:         ESC[38;5;Nm
//   - Basic 8/16 colors: ESC[30-37m, ESC[90-97m
//   - Reset:             ESC[0m or ESC[m
//
// Background colors and non-color attributes (bold, underline, etc.) are
// preserved unchanged. If a foreground color is not found in dimColorMap,
// the fallback color is used.
func dimSGRSequence(seq string, fallback lipgloss.Color) string {
	// Strip ESC[ prefix and m suffix
	if len(seq) < 3 || !strings.HasSuffix(seq, "m") {
		return seq
	}
	inner := seq[2 : len(seq)-1] // content between "[" and "m"

	// Handle reset sequences
	if inner == "" || inner == "0" {
		// Return a reset followed by re-applying the fallback foreground
		return seq
	}

	params := strings.Split(inner, ";")
	var newParams []string
	i := 0
	for i < len(params) {
		p := params[i]

		// 24-bit true color foreground: 38;2;R;G;B
		if p == "38" && i+1 < len(params) && params[i+1] == "2" && i+4 < len(params) {
			r, _ := strconv.Atoi(params[i+2])
			g, _ := strconv.Atoi(params[i+3])
			b, _ := strconv.Atoi(params[i+4])

			hexColor := fmt.Sprintf("#%02x%02x%02x", r, g, b)
			dimHex := lookupDimColor(hexColor, fallback)
			dr, dg, db := hexToRGB(dimHex)

			newParams = append(newParams, "38", "2",
				strconv.Itoa(dr), strconv.Itoa(dg), strconv.Itoa(db))
			i += 5
			continue
		}

		// 256-color foreground: 38;5;N
		if p == "38" && i+1 < len(params) && params[i+1] == "5" && i+2 < len(params) {
			// Replace with fallback as a 24-bit color since we cannot map
			// 256-color indices reliably
			dimHex := string(fallback)
			dr, dg, db := hexToRGB(dimHex)
			newParams = append(newParams, "38", "2",
				strconv.Itoa(dr), strconv.Itoa(dg), strconv.Itoa(db))
			i += 3
			continue
		}

		// Basic foreground colors (30-37, 90-97)
		if n, err := strconv.Atoi(p); err == nil && ((n >= 30 && n <= 37) || (n >= 90 && n <= 97)) {
			dimHex := string(fallback)
			dr, dg, db := hexToRGB(dimHex)
			newParams = append(newParams, "38", "2",
				strconv.Itoa(dr), strconv.Itoa(dg), strconv.Itoa(db))
			i++
			continue
		}

		// 24-bit true color background: 48;2;R;G;B -- drop background colors
		// to avoid them interfering with the dimmed background
		if p == "48" && i+1 < len(params) && params[i+1] == "2" && i+4 < len(params) {
			i += 5
			continue
		}

		// 256-color background: 48;5;N -- also drop
		if p == "48" && i+1 < len(params) && params[i+1] == "5" && i+2 < len(params) {
			i += 3
			continue
		}

		// Basic background colors (40-47, 100-107) -- drop
		if n, err := strconv.Atoi(p); err == nil && ((n >= 40 && n <= 47) || (n >= 100 && n <= 107)) {
			i++
			continue
		}

		// Preserve other attributes (bold, underline, etc.)
		newParams = append(newParams, p)
		i++
	}

	if len(newParams) == 0 {
		return seq
	}

	return "\x1b[" + strings.Join(newParams, ";") + "m"
}

// lookupDimColor finds the dimmed version of a hex color. It performs a
// case-insensitive lookup against dimColorMap. If no mapping is found,
// the fallback color string is returned.
func lookupDimColor(hexColor string, fallback lipgloss.Color) string {
	lower := strings.ToLower(hexColor)
	if dim, ok := dimColorMap[lower]; ok {
		return dim
	}
	return string(fallback)
}

// hexToRGB parses a hex color string (#RRGGBB) into its RGB components.
// If the string cannot be parsed, it returns (69, 71, 90) which is the
// Catppuccin Mocha Surface1 color as a safe fallback.
func hexToRGB(hex string) (int, int, int) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 69, 71, 90 // Surface1 fallback
	}
	r, err1 := strconv.ParseInt(hex[0:2], 16, 32)
	g, err2 := strconv.ParseInt(hex[2:4], 16, 32)
	b, err3 := strconv.ParseInt(hex[4:6], 16, 32)
	if err1 != nil || err2 != nil || err3 != nil {
		return 69, 71, 90
	}
	return int(r), int(g), int(b)
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
		if w < 0 {
			w = 0
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
