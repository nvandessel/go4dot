package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestClampToConstraints(t *testing.T) {
	tests := []struct {
		name           string
		termWidth      int
		termHeight     int
		maxWidthPct    float64
		maxHeightPct   float64
		expectedWidth  int
		expectedHeight int
	}{
		{
			name:           "default constraints clamp to 75% width and 65% height",
			termWidth:      100,
			termHeight:     40,
			maxWidthPct:    0.75,
			maxHeightPct:   0.65,
			expectedWidth:  75,
			expectedHeight: 26,
		},
		{
			name:           "zero percentages mean no constraint",
			termWidth:      100,
			termHeight:     40,
			maxWidthPct:    0,
			maxHeightPct:   0,
			expectedWidth:  100,
			expectedHeight: 40,
		},
		{
			name:           "negative percentages mean no constraint",
			termWidth:      100,
			termHeight:     40,
			maxWidthPct:    -0.5,
			maxHeightPct:   -0.5,
			expectedWidth:  100,
			expectedHeight: 40,
		},
		{
			name:           "percentage at 1.0 means no constraint",
			termWidth:      100,
			termHeight:     40,
			maxWidthPct:    1.0,
			maxHeightPct:   1.0,
			expectedWidth:  100,
			expectedHeight: 40,
		},
		{
			name:           "small terminal with constraints",
			termWidth:      20,
			termHeight:     10,
			maxWidthPct:    0.50,
			maxHeightPct:   0.50,
			expectedWidth:  10,
			expectedHeight: 5,
		},
		{
			name:           "confirm dialog 40% constraints",
			termWidth:      100,
			termHeight:     40,
			maxWidthPct:    0.40,
			maxHeightPct:   0.40,
			expectedWidth:  40,
			expectedHeight: 16,
		},
		{
			name:           "help overlay 60x70 constraints",
			termWidth:      120,
			termHeight:     50,
			maxWidthPct:    0.60,
			maxHeightPct:   0.70,
			expectedWidth:  72,
			expectedHeight: 35,
		},
		{
			name:           "conflict dialog 50x50 constraints",
			termWidth:      100,
			termHeight:     40,
			maxWidthPct:    0.50,
			maxHeightPct:   0.50,
			expectedWidth:  50,
			expectedHeight: 20,
		},
		{
			name:           "very small terminal clamps to minimum of 1",
			termWidth:      2,
			termHeight:     2,
			maxWidthPct:    0.10,
			maxHeightPct:   0.10,
			expectedWidth:  1,
			expectedHeight: 1,
		},
		{
			name:           "terminal already smaller than constraint",
			termWidth:      30,
			termHeight:     15,
			maxWidthPct:    0.90,
			maxHeightPct:   0.90,
			expectedWidth:  27,
			expectedHeight: 13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := OverlayStyle{
				MaxWidthPct:  tt.maxWidthPct,
				MaxHeightPct: tt.maxHeightPct,
			}
			w, h := ClampToConstraints(tt.termWidth, tt.termHeight, style)
			if w != tt.expectedWidth {
				t.Errorf("width: got %d, want %d", w, tt.expectedWidth)
			}
			if h != tt.expectedHeight {
				t.Errorf("height: got %d, want %d", h, tt.expectedHeight)
			}
		})
	}
}

func TestOverlayStyleVariants(t *testing.T) {
	t.Run("DefaultOverlayStyle has constraints", func(t *testing.T) {
		s := DefaultOverlayStyle()
		if s.MaxWidthPct != 0.75 {
			t.Errorf("expected MaxWidthPct 0.75, got %f", s.MaxWidthPct)
		}
		if s.MaxHeightPct != 0.65 {
			t.Errorf("expected MaxHeightPct 0.65, got %f", s.MaxHeightPct)
		}
	})

	t.Run("HelpOverlayStyle has 60x70 constraints", func(t *testing.T) {
		s := HelpOverlayStyle()
		if s.MaxWidthPct != 0.60 {
			t.Errorf("expected MaxWidthPct 0.60, got %f", s.MaxWidthPct)
		}
		if s.MaxHeightPct != 0.70 {
			t.Errorf("expected MaxHeightPct 0.70, got %f", s.MaxHeightPct)
		}
		if s.BorderColor != PrimaryColor {
			t.Error("expected primary border color for help style")
		}
	})

	t.Run("ConfirmOverlayStyle has 40x40 constraints", func(t *testing.T) {
		s := ConfirmOverlayStyle()
		if s.MaxWidthPct != 0.40 {
			t.Errorf("expected MaxWidthPct 0.40, got %f", s.MaxWidthPct)
		}
		if s.MaxHeightPct != 0.40 {
			t.Errorf("expected MaxHeightPct 0.40, got %f", s.MaxHeightPct)
		}
		if s.BorderColor != PrimaryColor {
			t.Error("expected primary border color for confirm style")
		}
	})

	t.Run("ConflictOverlayStyle has 50x50 constraints and warning border", func(t *testing.T) {
		s := ConflictOverlayStyle()
		if s.MaxWidthPct != 0.50 {
			t.Errorf("expected MaxWidthPct 0.50, got %f", s.MaxWidthPct)
		}
		if s.MaxHeightPct != 0.50 {
			t.Errorf("expected MaxHeightPct 0.50, got %f", s.MaxHeightPct)
		}
		if s.BorderColor != WarningColor {
			t.Error("expected warning border color for conflict style")
		}
	})

	t.Run("WarningOverlayStyle inherits default constraints", func(t *testing.T) {
		s := WarningOverlayStyle()
		d := DefaultOverlayStyle()
		if s.MaxWidthPct != d.MaxWidthPct {
			t.Errorf("expected MaxWidthPct %f, got %f", d.MaxWidthPct, s.MaxWidthPct)
		}
		if s.MaxHeightPct != d.MaxHeightPct {
			t.Errorf("expected MaxHeightPct %f, got %f", d.MaxHeightPct, s.MaxHeightPct)
		}
		if s.BorderColor != WarningColor {
			t.Error("expected warning border color")
		}
	})
}

func TestClampToConstraints_IntegrationWithChrome(t *testing.T) {
	// Verify that clamped dimensions leave room for border + padding.
	// Default style: border=1, paddingH=2, paddingV=1 => chrome is 2*(1+2)=6 wide, 2*(1+1)=4 tall
	style := DefaultOverlayStyle()
	termW, termH := 100, 40

	clampedW, clampedH := ClampToConstraints(termW, termH, style)

	// Verify the clamped size
	if clampedW != 75 {
		t.Errorf("clamped width: got %d, want 75", clampedW)
	}
	if clampedH != 26 {
		t.Errorf("clamped height: got %d, want 26", clampedH)
	}

	// Simulate what overlayContentSize does: subtract chrome
	border := 0
	if style.BorderStyle != (lipgloss.Border{}) {
		border = 1
	}
	contentW := clampedW - 2*(border+style.PaddingH)
	contentH := clampedH - 2*(border+style.PaddingV)

	// Content should be significantly smaller than terminal
	if contentW >= termW {
		t.Errorf("content width %d should be smaller than terminal width %d", contentW, termW)
	}
	if contentH >= termH {
		t.Errorf("content height %d should be smaller than terminal height %d", contentH, termH)
	}

	// Content should be positive
	if contentW < 1 {
		t.Errorf("content width should be positive, got %d", contentW)
	}
	if contentH < 1 {
		t.Errorf("content height should be positive, got %d", contentH)
	}

	// For 100x40 terminal with 75% x 65% constraints:
	// clamped = 75 x 26
	// chrome = 6 wide, 4 tall
	// content = 69 x 22
	expectedContentW := 69
	expectedContentH := 22
	if contentW != expectedContentW {
		t.Errorf("content width: got %d, want %d", contentW, expectedContentW)
	}
	if contentH != expectedContentH {
		t.Errorf("content height: got %d, want %d", contentH, expectedContentH)
	}
}
