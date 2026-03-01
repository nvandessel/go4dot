package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestDefaultOverlayStyle(t *testing.T) {
	style := DefaultOverlayStyle()

	if style.BorderColor != PrimaryColor {
		t.Errorf("expected border color %v, got %v", PrimaryColor, style.BorderColor)
	}
	if style.PaddingH != 2 {
		t.Errorf("expected horizontal padding 2, got %d", style.PaddingH)
	}
	if style.PaddingV != 1 {
		t.Errorf("expected vertical padding 1, got %d", style.PaddingV)
	}
	if style.DimChar != " " {
		t.Errorf("expected dim char space, got %q", style.DimChar)
	}
}

func TestWarningOverlayStyle(t *testing.T) {
	style := WarningOverlayStyle()

	if style.BorderColor != WarningColor {
		t.Errorf("expected border color %v, got %v", WarningColor, style.BorderColor)
	}
	if style.PaddingH != 2 {
		t.Errorf("expected horizontal padding 2, got %d", style.PaddingH)
	}
}

func TestStripAnsi(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text unchanged",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "removes color codes",
			input:    "\x1b[31mred text\x1b[0m",
			expected: "red text",
		},
		{
			name:     "removes multiple sequences",
			input:    "\x1b[1m\x1b[32mbold green\x1b[0m normal",
			expected: "bold green normal",
		},
		{
			name:     "handles cursor movement codes",
			input:    "\x1b[2Ahello\x1b[3B",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripAnsi(tt.input)
			if result != tt.expected {
				t.Errorf("stripAnsi(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDimContent(t *testing.T) {
	tests := []struct {
		name          string
		content       string
		width         int
		height        int
		expectedLines int
	}{
		{
			name:          "creates correct number of lines",
			content:       "line1\nline2\nline3",
			width:         20,
			height:        5,
			expectedLines: 5,
		},
		{
			name:          "pads short content to fill height",
			content:       "short",
			width:         10,
			height:        3,
			expectedLines: 3,
		},
		{
			name:          "handles empty content",
			content:       "",
			width:         10,
			height:        2,
			expectedLines: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dimContent(tt.content, tt.width, tt.height, " ", lipgloss.Color("#333333"))
			lines := strings.Split(result, "\n")
			if len(lines) != tt.expectedLines {
				t.Errorf("expected %d lines, got %d", tt.expectedLines, len(lines))
			}
		})
	}
}

func TestPlaceOverlay(t *testing.T) {
	tests := []struct {
		name          string
		bgWidth       int
		bgHeight      int
		modalContent  string
		expectedLines int
	}{
		{
			name:          "overlay preserves background height",
			bgWidth:       40,
			bgHeight:      10,
			modalContent:  "modal",
			expectedLines: 10,
		},
		{
			name:          "modal content appears in output",
			bgWidth:       40,
			bgHeight:      10,
			modalContent:  "MODAL TEXT",
			expectedLines: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bgLines []string
			for i := 0; i < tt.bgHeight; i++ {
				bgLines = append(bgLines, strings.Repeat(".", tt.bgWidth))
			}
			bg := strings.Join(bgLines, "\n")

			result := placeOverlay(bg, tt.modalContent, tt.bgWidth, tt.bgHeight, lipgloss.Color("#333333"))
			lines := strings.Split(result, "\n")

			if len(lines) != tt.expectedLines {
				t.Errorf("expected %d lines, got %d", tt.expectedLines, len(lines))
			}

			if !strings.Contains(result, tt.modalContent) {
				t.Errorf("expected output to contain modal content %q", tt.modalContent)
			}
		})
	}
}

func TestPlaceOverlay_Centering(t *testing.T) {
	bgWidth := 20
	bgHeight := 10
	var bgLines []string
	for i := 0; i < bgHeight; i++ {
		bgLines = append(bgLines, strings.Repeat(".", bgWidth))
	}
	bg := strings.Join(bgLines, "\n")

	modal := "XX"
	result := placeOverlay(bg, modal, bgWidth, bgHeight, lipgloss.Color("#333333"))

	lines := strings.Split(result, "\n")

	expectedY := (bgHeight - 1) / 2
	found := false
	for i, line := range lines {
		if strings.Contains(line, "XX") {
			if i != expectedY {
				t.Errorf("expected modal at line %d, found at line %d", expectedY, i)
			}
			found = true
			break
		}
	}
	if !found {
		t.Error("expected modal content 'XX' to appear in the output")
	}
}

func TestRenderOverlay_SmallTerminal(t *testing.T) {
	result := RenderOverlay("bg", "modal", 5, 3, DefaultOverlayStyle())
	if !strings.Contains(result, "modal") {
		t.Error("expected small terminal to return modal content")
	}
}

func TestRenderOverlay_ContainsModal(t *testing.T) {
	bg := strings.Repeat(strings.Repeat(".", 80)+"\n", 24)
	modal := "Test Modal Content"
	style := DefaultOverlayStyle()

	result := RenderOverlay(bg, modal, 80, 24, style)

	if !strings.Contains(result, modal) {
		t.Error("expected overlay output to contain modal content")
	}
}

func TestRenderOverlay_DifferentStyles(t *testing.T) {
	tests := []struct {
		name  string
		style OverlayStyle
	}{
		{
			name:  "default style",
			style: DefaultOverlayStyle(),
		},
		{
			name:  "warning style",
			style: WarningOverlayStyle(),
		},
		{
			name: "custom style",
			style: OverlayStyle{
				BorderStyle: lipgloss.DoubleBorder(),
				BorderColor: ErrorColor,
				PaddingH:    1,
				PaddingV:    0,
				Background:  lipgloss.Color("#000000"),
				DimChar:     ".",
				DimColor:    lipgloss.Color("#111111"),
			},
		},
	}

	bg := strings.Repeat(strings.Repeat("X", 60)+"\n", 20)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("RenderOverlay panicked with %s: %v", tt.name, r)
				}
			}()

			result := RenderOverlay(bg, "modal content", 60, 20, tt.style)
			if result == "" {
				t.Errorf("expected non-empty result for %s", tt.name)
			}
			if !strings.Contains(result, "modal content") {
				t.Errorf("expected result to contain 'modal content' for %s", tt.name)
			}
		})
	}
}

func TestPlaceOverlay_MultilineModal(t *testing.T) {
	bgWidth := 40
	bgHeight := 15
	var bgLines []string
	for i := 0; i < bgHeight; i++ {
		bgLines = append(bgLines, strings.Repeat(".", bgWidth))
	}
	bg := strings.Join(bgLines, "\n")

	modal := "Line 1\nLine 2\nLine 3"
	result := placeOverlay(bg, modal, bgWidth, bgHeight, lipgloss.Color("#333333"))

	lines := strings.Split(result, "\n")
	if len(lines) != bgHeight {
		t.Errorf("expected %d lines, got %d", bgHeight, len(lines))
	}

	if !strings.Contains(result, "Line 1") {
		t.Error("expected result to contain 'Line 1'")
	}
	if !strings.Contains(result, "Line 2") {
		t.Error("expected result to contain 'Line 2'")
	}
	if !strings.Contains(result, "Line 3") {
		t.Error("expected result to contain 'Line 3'")
	}
}

func TestPlaceOverlay_WideBackground(t *testing.T) {
	bgWidth := 10
	bgHeight := 3
	line := strings.Repeat("界", 5) // width 10
	var bgLines []string
	for i := 0; i < bgHeight; i++ {
		bgLines = append(bgLines, line)
	}
	bg := strings.Join(bgLines, "\n")

	modal := "X"
	result := placeOverlay(bg, modal, bgWidth, bgHeight, lipgloss.Color("#333333"))

	lines := strings.Split(result, "\n")
	if len(lines) != bgHeight {
		t.Errorf("expected %d lines, got %d", bgHeight, len(lines))
	}
	if !strings.Contains(result, modal) {
		t.Error("expected result to contain modal content")
	}
	for _, line := range lines {
		if lipgloss.Width(line) != bgWidth {
			t.Errorf("expected line width %d, got %d", bgWidth, lipgloss.Width(line))
		}
	}
}

func TestColorToANSIBg(t *testing.T) {
	// colorToANSIBg uses lipgloss rendering, which may return empty when
	// no terminal is attached (e.g., during tests). This is correct behavior.
	t.Run("does not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("colorToANSIBg panicked: %v", r)
			}
		}()
		colorToANSIBg(lipgloss.Color("#252545"))
		colorToANSIBg(lipgloss.Color("#FF0000"))
		colorToANSIBg(lipgloss.Color(""))
	})

	t.Run("returns valid ANSI or empty", func(t *testing.T) {
		result := colorToANSIBg(lipgloss.Color("#252545"))
		// In a terminal, this returns an ANSI sequence; in tests, it may be empty.
		// NOTE: colorToANSIBg relies on lipgloss's internal rendering format
		// (rendering a space and slicing before the first " "). If lipgloss
		// changes its output format, this extraction may break silently.
		if result != "" && !strings.HasPrefix(result, "\x1b[") {
			t.Errorf("if non-empty, expected ANSI escape prefix, got %q", result)
		}
	})

	t.Run("extracted sequence re-renders correctly", func(t *testing.T) {
		// If colorToANSIBg produces a non-empty sequence, applying it and
		// then resetting should produce the same result as lipgloss rendering.
		result := colorToANSIBg(lipgloss.Color("#FF0000"))
		if result == "" {
			t.Skip("no ANSI output (expected in non-terminal test environments)")
		}
		// The sequence should end with 'm' (SGR terminator)
		if result[len(result)-1] != 'm' {
			t.Errorf("expected ANSI sequence ending with 'm', got %q", result)
		}
	})
}

func TestFillBackground(t *testing.T) {
	bg := lipgloss.Color("#252545")

	t.Run("preserves line count", func(t *testing.T) {
		content := "short\nlonger line here"
		result := fillBackground(content, bg)
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
	})

	t.Run("normalizes line widths", func(t *testing.T) {
		content := "short\nlonger line here"
		result := fillBackground(content, bg)
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		w0 := lipgloss.Width(lines[0])
		w1 := lipgloss.Width(lines[1])
		if w0 != w1 {
			t.Errorf("expected equal widths, got %d and %d", w0, w1)
		}
	})

	t.Run("preserves content text", func(t *testing.T) {
		content := "hello world"
		result := fillBackground(content, bg)
		plain := stripAnsi(result)
		if !strings.Contains(plain, "hello world") {
			t.Errorf("expected output to contain 'hello world', got %q", plain)
		}
	})

	t.Run("handles empty content", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("fillBackground panicked on empty content: %v", r)
			}
		}()
		fillBackground("", bg)
	})

	t.Run("handles styled content with resets", func(t *testing.T) {
		styled := "\x1b[31mred text\x1b[0m and more"
		result := fillBackground(styled, bg)
		plain := stripAnsi(result)
		if !strings.Contains(plain, "red text") {
			t.Error("expected styled text to be preserved")
		}
		if !strings.Contains(plain, "and more") {
			t.Error("expected text after reset to be preserved")
		}
	})
}

func TestHexToRGB(t *testing.T) {
	tests := []struct {
		name  string
		hex   string
		wantR int
		wantG int
		wantB int
	}{
		{
			name:  "standard hex color",
			hex:   "#b4befe",
			wantR: 180, wantG: 190, wantB: 254,
		},
		{
			name:  "black",
			hex:   "#000000",
			wantR: 0, wantG: 0, wantB: 0,
		},
		{
			name:  "white",
			hex:   "#ffffff",
			wantR: 255, wantG: 255, wantB: 255,
		},
		{
			name:  "without hash prefix",
			hex:   "45475a",
			wantR: 69, wantG: 71, wantB: 90,
		},
		{
			name:  "invalid hex falls back to Surface1",
			hex:   "xyz",
			wantR: 69, wantG: 71, wantB: 90,
		},
		{
			name:  "empty string falls back to Surface1",
			hex:   "",
			wantR: 69, wantG: 71, wantB: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, g, b := hexToRGB(tt.hex)
			if r != tt.wantR || g != tt.wantG || b != tt.wantB {
				t.Errorf("hexToRGB(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tt.hex, r, g, b, tt.wantR, tt.wantG, tt.wantB)
			}
		})
	}
}

func TestLookupDimColor(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "maps primary color",
			input:    "#b4befe",
			expected: "#585b7f",
		},
		{
			name:     "maps secondary color",
			input:    "#a6e3a1",
			expected: "#536f50",
		},
		{
			name:     "maps error color",
			input:    "#f38ba8",
			expected: "#794554",
		},
		{
			name:     "case insensitive lookup",
			input:    "#B4BEFE",
			expected: "#585b7f",
		},
		{
			name:     "unknown color falls back",
			input:    "#123456",
			expected: "#45475a",
		},
		{
			name:     "empty string falls back",
			input:    "",
			expected: "#45475a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lookupDimColor(tt.input, fallback)
			if result != tt.expected {
				t.Errorf("lookupDimColor(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseEscapeSeq(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantSeq string
		wantEnd int
	}{
		{
			name:    "SGR color sequence",
			input:   "\x1b[31mhello",
			wantSeq: "\x1b[31m",
			wantEnd: 5,
		},
		{
			name:    "SGR 24-bit color",
			input:   "\x1b[38;2;180;190;254mtext",
			wantSeq: "\x1b[38;2;180;190;254m",
			wantEnd: 19,
		},
		{
			name:    "reset sequence",
			input:   "\x1b[0m",
			wantSeq: "\x1b[0m",
			wantEnd: 4,
		},
		{
			name:    "cursor movement (non-SGR)",
			input:   "\x1b[2Ahello",
			wantSeq: "\x1b[2A",
			wantEnd: 4,
		},
		{
			name:    "not an escape sequence",
			input:   "hello",
			wantSeq: "",
			wantEnd: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runes := []rune(tt.input)
			seq, end := parseEscapeSeq(runes, 0)
			if seq != tt.wantSeq {
				t.Errorf("parseEscapeSeq seq = %q, want %q", seq, tt.wantSeq)
			}
			if end != tt.wantEnd {
				t.Errorf("parseEscapeSeq end = %d, want %d", end, tt.wantEnd)
			}
		})
	}
}

func TestDimSGRSequence(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	tests := []struct {
		name     string
		seq      string
		expected string
	}{
		{
			name:     "reset preserved",
			seq:      "\x1b[0m",
			expected: "\x1b[0m",
		},
		{
			name:     "empty reset preserved",
			seq:      "\x1b[m",
			expected: "\x1b[m",
		},
		{
			name: "24-bit foreground known color gets dimmed",
			seq:  "\x1b[38;2;180;190;254m", // #b4befe (PrimaryColor)
			// Should map to #585b7f = rgb(88, 91, 127)
			expected: "\x1b[38;2;88;91;127m",
		},
		{
			name: "24-bit foreground unknown color gets fallback",
			seq:  "\x1b[38;2;1;2;3m",
			// Unknown -> fallback #45475a = rgb(69, 71, 90)
			expected: "\x1b[38;2;69;71;90m",
		},
		{
			name: "basic foreground color gets fallback",
			seq:  "\x1b[31m",
			// Basic red -> fallback
			expected: "\x1b[38;2;69;71;90m",
		},
		{
			name: "bold attribute preserved with foreground",
			seq:  "\x1b[1;38;2;180;190;254m",
			// Bold preserved, foreground dimmed
			expected: "\x1b[1;38;2;88;91;127m",
		},
		{
			name:     "background color dropped",
			seq:      "\x1b[48;2;30;30;46m",
			expected: "\x1b[48;2;30;30;46m", // empty params -> returns original
		},
		{
			name:     "non-SGR sequence returned as-is",
			seq:      "\x1b[2A",
			expected: "\x1b[2A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dimSGRSequence(tt.seq, fallback)
			if result != tt.expected {
				t.Errorf("dimSGRSequence(%q) = %q, want %q", tt.seq, result, tt.expected)
			}
		})
	}
}

func TestDimAnsiColors(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	t.Run("plain text gets fallback color", func(t *testing.T) {
		result := dimAnsiColors("hello", fallback)
		plain := stripAnsi(result)
		if plain != "hello" {
			t.Errorf("expected plain text 'hello', got %q", plain)
		}
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		result := dimAnsiColors("", fallback)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("preserves text through color replacement", func(t *testing.T) {
		// Simulate styled text: red "error" followed by green "ok"
		styled := "\x1b[38;2;243;139;168merror\x1b[0m \x1b[38;2;166;227;161mok\x1b[0m"
		result := dimAnsiColors(styled, fallback)
		plain := stripAnsi(result)
		if !strings.Contains(plain, "error") {
			t.Error("expected 'error' text to be preserved")
		}
		if !strings.Contains(plain, "ok") {
			t.Error("expected 'ok' text to be preserved")
		}
	})

	t.Run("cursor movement sequences are skipped", func(t *testing.T) {
		input := "\x1b[2Ahello"
		result := dimAnsiColors(input, fallback)
		plain := stripAnsi(result)
		if plain != "hello" {
			t.Errorf("expected 'hello', got %q", plain)
		}
	})

	t.Run("mixed styled and plain text", func(t *testing.T) {
		// "Status: " (plain) + "OK" (green styled)
		input := "Status: \x1b[38;2;166;227;161mOK\x1b[0m"
		result := dimAnsiColors(input, fallback)
		plain := stripAnsi(result)
		if !strings.Contains(plain, "Status:") {
			t.Error("expected 'Status:' text to be preserved")
		}
		if !strings.Contains(plain, "OK") {
			t.Error("expected 'OK' text to be preserved")
		}
	})

	t.Run("border characters preserved", func(t *testing.T) {
		input := "\x1b[38;2;180;190;254m╭────────╮\x1b[0m"
		result := dimAnsiColors(input, fallback)
		plain := stripAnsi(result)
		if !strings.Contains(plain, "╭────────╮") {
			t.Errorf("expected border chars preserved, got %q", plain)
		}
	})
}

func TestDimContent_PreservesTextStructure(t *testing.T) {
	// Build a content string that mimics a dashboard with styled borders
	borderLine := "\x1b[38;2;180;190;254m╭──────────────────╮\x1b[0m"
	textLine := "\x1b[38;2;180;190;254m│\x1b[0m \x1b[38;2;205;214;244mDashboard\x1b[0m        \x1b[38;2;180;190;254m│\x1b[0m"
	bottomLine := "\x1b[38;2;180;190;254m╰──────────────────╯\x1b[0m"
	content := strings.Join([]string{borderLine, textLine, bottomLine}, "\n")

	result := dimContent(content, 40, 5, " ", lipgloss.Color("#45475a"))
	lines := strings.Split(result, "\n")

	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}

	// Verify that the text characters are preserved (borders and text)
	plain := stripAnsi(result)
	if !strings.Contains(plain, "╭") {
		t.Error("expected border character to be preserved")
	}
	if !strings.Contains(plain, "Dashboard") {
		t.Error("expected 'Dashboard' text to be preserved")
	}
	if !strings.Contains(plain, "╰") {
		t.Error("expected bottom border to be preserved")
	}
}

func TestDimContent_PaddingStillWorks(t *testing.T) {
	// Short content should still be padded to fill width
	content := "short"
	width := 20
	height := 3
	result := dimContent(content, width, height, " ", lipgloss.Color("#45475a"))
	lines := strings.Split(result, "\n")

	if len(lines) != height {
		t.Fatalf("expected %d lines, got %d", height, len(lines))
	}

	// Each line should have the correct visual width
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < width {
			t.Errorf("line %d: expected width >= %d, got %d", i, width, w)
		}
	}
}

func TestDimColorMap_Coverage(t *testing.T) {
	// Verify that all the main app colors are in the dim map
	appColors := map[string]lipgloss.Color{
		"PrimaryColor":   PrimaryColor,
		"SecondaryColor": SecondaryColor,
		"ErrorColor":     ErrorColor,
		"WarningColor":   WarningColor,
		"SubtleColor":    SubtleColor,
		"TextColor":      TextColor,
	}

	for name, color := range appColors {
		hexColor := strings.ToLower(string(color))
		if _, ok := dimColorMap[hexColor]; !ok {
			t.Errorf("dimColorMap missing entry for %s (%s)", name, hexColor)
		}
	}
}

func TestDimSGRSequence_256Color(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	// 256-color foreground: ESC[38;5;196m (bright red)
	seq := "\x1b[38;5;196m"
	result := dimSGRSequence(seq, fallback)

	// Should be converted to 24-bit fallback color
	dr, dg, db := hexToRGB(string(fallback))
	expected := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", dr, dg, db)
	if result != expected {
		t.Errorf("dimSGRSequence(%q) = %q, want %q", seq, result, expected)
	}
}

func TestDimSGRSequence_BasicBackground(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	// Basic background color (42 = green bg) should be dropped
	seq := "\x1b[42m"
	result := dimSGRSequence(seq, fallback)
	// With only a background param dropped, newParams is empty, so original is returned
	if result != seq {
		t.Errorf("dimSGRSequence(%q) = %q, want original %q", seq, result, seq)
	}
}

func TestDimSGRSequence_256Background(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	// 256-color background: ESC[48;5;21m should be dropped
	seq := "\x1b[48;5;21m"
	result := dimSGRSequence(seq, fallback)
	// With only background param dropped, newParams is empty, so original is returned
	if result != seq {
		t.Errorf("dimSGRSequence(%q) = %q, want original %q", seq, result, seq)
	}
}

func TestDimSGRSequence_FgAndBgCombined(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	// Combined: bold + foreground + background
	// ESC[1;38;2;180;190;254;48;2;30;30;46m
	seq := "\x1b[1;38;2;180;190;254;48;2;30;30;46m"
	result := dimSGRSequence(seq, fallback)

	// Bold should be preserved, fg dimmed to #585b7f (88,91,127), bg dropped
	expected := "\x1b[1;38;2;88;91;127m"
	if result != expected {
		t.Errorf("dimSGRSequence(%q) = %q, want %q", seq, result, expected)
	}
}

func TestDimSGRSequence_HighIntensityBackground(t *testing.T) {
	fallback := lipgloss.Color("#45475a")

	// High-intensity background (100-107) should be dropped
	seq := "\x1b[104m"
	result := dimSGRSequence(seq, fallback)
	if result != seq {
		t.Errorf("dimSGRSequence(%q) = %q, want original", seq, result)
	}
}
