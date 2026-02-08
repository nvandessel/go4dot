package ui

import (
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

	modalHeight := len(strings.Split(modal, "\n"))
	expectedY := (bgHeight - modalHeight) / 2
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
