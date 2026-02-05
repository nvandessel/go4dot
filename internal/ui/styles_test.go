package ui

import (
	"testing"
)

func TestColorConstants(t *testing.T) {
	// Verify all color constants are defined
	colors := map[string]interface{}{
		"PrimaryColor":   PrimaryColor,
		"SecondaryColor": SecondaryColor,
		"ErrorColor":     ErrorColor,
		"WarningColor":   WarningColor,
		"SubtleColor":    SubtleColor,
		"TextColor":      TextColor,
	}

	for name, color := range colors {
		if color == nil {
			t.Errorf("%s should be defined", name)
		}
	}
}

func TestStylesRender(t *testing.T) {
	// Test that styles can render content without panicking
	tests := []struct {
		name  string
		style interface{ Render(...string) string }
	}{
		{"TitleStyle", TitleStyle},
		{"TextStyle", TextStyle},
		{"SubtleStyle", SubtleStyle},
		{"ErrorStyle", ErrorStyle},
		{"SuccessStyle", SuccessStyle},
		{"WarningStyle", WarningStyle},
		{"BoxStyle", BoxStyle},
		{"ItemStyle", ItemStyle},
		{"SelectedItemStyle", SelectedItemStyle},
		{"HeaderStyle", HeaderStyle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic and should return non-empty for non-empty input
			result := tt.style.Render("test content")
			if result == "" {
				t.Errorf("%s.Render() returned empty string", tt.name)
			}
		})
	}
}

func TestStylesWithEmptyContent(t *testing.T) {
	// Styles should handle empty content gracefully
	styles := []struct {
		name  string
		style interface{ Render(...string) string }
	}{
		{"TitleStyle", TitleStyle},
		{"ErrorStyle", ErrorStyle},
		{"SuccessStyle", SuccessStyle},
	}

	for _, s := range styles {
		t.Run(s.name+"_empty", func(t *testing.T) {
			// Should not panic with empty content
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s.Render(\"\") panicked: %v", s.name, r)
				}
			}()
			s.style.Render("")
		})
	}
}
