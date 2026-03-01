package dashboard

import (
	"testing"

	"github.com/nvandessel/go4dot/internal/ui"
)

func TestOverlayContentSize(t *testing.T) {
	tests := []struct {
		name           string
		width          int
		height         int
		style          ui.OverlayStyle
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:   "default style constrains content to 75% width, 65% height minus chrome",
			width:  100,
			height: 40,
			style:  ui.DefaultOverlayStyle(),
			// 75% of 100 = 75, minus 2*(1+2) = 6 => 69
			// 65% of 40 = 26, minus 2*(1+1) = 4 => 22
			expectedWidth:  69,
			expectedHeight: 22,
		},
		{
			name:   "help style uses 60% width, 70% height",
			width:  100,
			height: 40,
			style:  ui.HelpOverlayStyle(),
			// 60% of 100 = 60, minus 6 => 54
			// 70% of 40 = 28, minus 4 => 24
			expectedWidth:  54,
			expectedHeight: 24,
		},
		{
			name:   "confirm style uses 40% width, 40% height",
			width:  100,
			height: 40,
			style:  ui.ConfirmOverlayStyle(),
			// 40% of 100 = 40, minus 6 => 34
			// 40% of 40 = 16, minus 4 => 12
			expectedWidth:  34,
			expectedHeight: 12,
		},
		{
			name:   "conflict style uses 50% width, 50% height",
			width:  100,
			height: 40,
			style:  ui.ConflictOverlayStyle(),
			// 50% of 100 = 50, minus 6 => 44
			// 50% of 40 = 20, minus 4 => 16
			expectedWidth:  44,
			expectedHeight: 16,
		},
		{
			name:   "small terminal clamps to minimum 1",
			width:  8,
			height: 4,
			style:  ui.DefaultOverlayStyle(),
			// 75% of 8 = 6, minus 6 => 0 clamped to 1
			// 65% of 4 = 2, minus 4 => -2 clamped to 1
			expectedWidth:  1,
			expectedHeight: 1,
		},
		{
			name:  "zero pct means no constraint",
			width: 100, height: 40,
			style: ui.OverlayStyle{
				MaxWidthPct:  0,
				MaxHeightPct: 0,
				PaddingH:     2,
				PaddingV:     1,
			},
			// No border (zero value Border), no constraint:
			// width = 100 - 2*(0+2) = 96
			// height = 40 - 2*(0+1) = 38
			expectedWidth:  96,
			expectedHeight: 38,
		},
		{
			name:   "large terminal with default style",
			width:  200,
			height: 60,
			style:  ui.DefaultOverlayStyle(),
			// 75% of 200 = 150, minus 6 => 144
			// 65% of 60 = 39, minus 4 => 35
			expectedWidth:  144,
			expectedHeight: 35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, h := overlayContentSize(tt.width, tt.height, tt.style)
			if w != tt.expectedWidth {
				t.Errorf("content width: got %d, want %d", w, tt.expectedWidth)
			}
			if h != tt.expectedHeight {
				t.Errorf("content height: got %d, want %d", h, tt.expectedHeight)
			}
		})
	}
}

func TestOverlayContentSize_ContentSmallerThanTerminal(t *testing.T) {
	// Verify that for any styled overlay, the content size is always
	// strictly smaller than the terminal dimensions.
	styles := []struct {
		name  string
		style ui.OverlayStyle
	}{
		{"default", ui.DefaultOverlayStyle()},
		{"help", ui.HelpOverlayStyle()},
		{"confirm", ui.ConfirmOverlayStyle()},
		{"conflict", ui.ConflictOverlayStyle()},
		{"warning", ui.WarningOverlayStyle()},
	}

	for _, ss := range styles {
		t.Run(ss.name, func(t *testing.T) {
			w, h := overlayContentSize(100, 40, ss.style)
			if w >= 100 {
				t.Errorf("content width %d should be less than terminal width 100", w)
			}
			if h >= 40 {
				t.Errorf("content height %d should be less than terminal height 40", h)
			}
			if w < 1 {
				t.Errorf("content width should be at least 1, got %d", w)
			}
			if h < 1 {
				t.Errorf("content height should be at least 1, got %d", h)
			}
		})
	}
}
