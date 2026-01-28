package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/ui"
)

func TestOutputPane(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() *OutputPane
		assert func(t *testing.T, o *OutputPane)
	}{
		{
			name: "New has default title",
			setup: func() *OutputPane {
				o := NewOutputPane()
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				if o.title != "Output" {
					t.Errorf("expected title 'Output', got %q", o.title)
				}
			},
		},
		{
			name: "New has empty logs",
			setup: func() *OutputPane {
				o := NewOutputPane()
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				if len(o.logs) != 0 {
					t.Errorf("expected empty logs, got %d", len(o.logs))
				}
			},
		},
		{
			name: "SetSize updates dimensions",
			setup: func() *OutputPane {
				o := NewOutputPane()
				o.SetSize(80, 20)
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				if o.width != 80 {
					t.Errorf("expected width 80, got %d", o.width)
				}
				if o.height != 20 {
					t.Errorf("expected height 20, got %d", o.height)
				}
			},
		},
		{
			name: "SetTitle updates title",
			setup: func() *OutputPane {
				o := NewOutputPane()
				o.SetTitle("Custom Title")
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				if o.title != "Custom Title" {
					t.Errorf("expected title 'Custom Title', got %q", o.title)
				}
			},
		},
		{
			name: "Clear removes all logs",
			setup: func() *OutputPane {
				o := NewOutputPane()
				o.SetSize(80, 20)
				o.AddLog("info", "Test message")
				o.Clear()
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				if len(o.logs) != 0 {
					t.Errorf("expected 0 logs after clear, got %d", len(o.logs))
				}
			},
		},
		{
			name: "View empty shows placeholder",
			setup: func() *OutputPane {
				o := NewOutputPane()
				o.SetSize(80, 20)
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				view := o.View()
				if !strings.Contains(view, "No output yet") {
					t.Error("expected placeholder text for empty pane")
				}
			},
		},
		{
			name: "View with logs is non-empty",
			setup: func() *OutputPane {
				o := NewOutputPane()
				o.SetSize(80, 20)
				o.AddLog("success", "Operation completed")
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				view := o.View()
				if view == "" {
					t.Error("expected non-empty view with logs")
				}
			},
		},
		{
			name: "RenderWithBorder has corners",
			setup: func() *OutputPane {
				o := NewOutputPane()
				o.SetSize(40, 10)
				o.SetTitle("Test")
				return &o
			},
			assert: func(t *testing.T, o *OutputPane) {
				rendered := o.RenderWithBorder(false)
				if !strings.Contains(rendered, "╭") {
					t.Error("expected top-left corner in border")
				}
				if !strings.Contains(rendered, "╯") {
					t.Error("expected bottom-right corner in border")
				}
				if !strings.Contains(rendered, "Test") {
					t.Error("expected title in border")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := tt.setup()
			tt.assert(t, o)
		})
	}
}

func TestOutputPane_AddLog(t *testing.T) {
	tests := []struct {
		name          string
		logs          []struct{ level, message string }
		expectedCount int
	}{
		{
			name: "single log",
			logs: []struct{ level, message string }{
				{"info", "Test message"},
			},
			expectedCount: 1,
		},
		{
			name: "multiple levels",
			logs: []struct{ level, message string }{
				{"info", "Info message"},
				{"success", "Success message"},
				{"warning", "Warning message"},
				{"error", "Error message"},
			},
			expectedCount: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewOutputPane()
			o.SetSize(80, 20)

			for _, log := range tt.logs {
				o.AddLog(log.level, log.message)
			}

			if len(o.logs) != tt.expectedCount {
				t.Errorf("expected %d logs, got %d", tt.expectedCount, len(o.logs))
			}

			// Verify first log details if present
			if len(tt.logs) > 0 && len(o.logs) > 0 {
				if o.logs[0].Level != tt.logs[0].level {
					t.Errorf("expected level %q, got %q", tt.logs[0].level, o.logs[0].Level)
				}
				if o.logs[0].Message != tt.logs[0].message {
					t.Errorf("expected message %q, got %q", tt.logs[0].message, o.logs[0].Message)
				}
			}
		})
	}
}

func TestRenderPaneWithTitle(t *testing.T) {
	tests := []struct {
		name    string
		content string
		title   string
		width   int
		height  int
	}{
		{"normal pane", "Hello", "Title", 20, 5},
		{"too small width", "Hello", "Title", 3, 5},
		{"too small height", "Hello", "Title", 20, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			result := renderPaneWithTitle(tt.content, tt.title, tt.width, tt.height, ui.PrimaryColor)
			// For valid dimensions, expect non-empty result
			if tt.width >= 5 && tt.height >= 3 && result == "" {
				t.Error("expected non-empty result for valid dimensions")
			}
		})
	}
}
