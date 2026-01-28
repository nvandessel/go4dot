package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

func TestNewOutputPane(t *testing.T) {
	pane := NewOutputPane("Test Output")

	if pane.title != "Test Output" {
		t.Errorf("expected title 'Test Output', got '%s'", pane.title)
	}
	if len(pane.logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(pane.logs))
	}
	if pane.ready {
		t.Error("expected pane to not be ready before SetSize")
	}
}

func TestOutputPane_SetSize(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	if !pane.ready {
		t.Error("expected pane to be ready after SetSize")
	}
	if pane.width != 80 {
		t.Errorf("expected width 80, got %d", pane.width)
	}
	if pane.height != 24 {
		t.Errorf("expected height 24, got %d", pane.height)
	}

	// Test inner dimensions
	expectedInnerW := 80 - borderWidth
	expectedInnerH := 24 - borderWidth - titleHeight

	if pane.viewport.Width != expectedInnerW {
		t.Errorf("expected viewport width %d, got %d", expectedInnerW, pane.viewport.Width)
	}
	if pane.viewport.Height != expectedInnerH {
		t.Errorf("expected viewport height %d, got %d", expectedInnerH, pane.viewport.Height)
	}
}

func TestOutputPane_SetSize_MultipleUpdates(t *testing.T) {
	pane := NewOutputPane("Output")

	// First size
	pane.SetSize(80, 24)
	if !pane.ready {
		t.Error("expected pane to be ready")
	}

	// Resize
	pane.SetSize(100, 30)
	if pane.width != 100 {
		t.Errorf("expected width 100, got %d", pane.width)
	}
	if pane.height != 30 {
		t.Errorf("expected height 30, got %d", pane.height)
	}
}

func TestOutputPane_SetSize_MinimumDimensions(t *testing.T) {
	pane := NewOutputPane("Output")

	// Very small size
	pane.SetSize(4, 4)

	// Should clamp to minimum of 1
	if pane.viewport.Width < 1 {
		t.Errorf("expected viewport width >= 1, got %d", pane.viewport.Width)
	}
	if pane.viewport.Height < 1 {
		t.Errorf("expected viewport height >= 1, got %d", pane.viewport.Height)
	}
}

func TestOutputPane_AddLog(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	pane.AddLog("info", "Test message")

	if len(pane.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(pane.logs))
	}

	if pane.logs[0].Level != "info" {
		t.Errorf("expected level 'info', got '%s'", pane.logs[0].Level)
	}
	if pane.logs[0].Message != "Test message" {
		t.Errorf("expected message 'Test message', got '%s'", pane.logs[0].Message)
	}
}

func TestOutputPane_AddLog_MultipleLevels(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	tests := []struct {
		level   string
		message string
	}{
		{"info", "Info message"},
		{"success", "Success message"},
		{"warn", "Warning message"},
		{"error", "Error message"},
	}

	for _, tt := range tests {
		pane.AddLog(tt.level, tt.message)
	}

	if len(pane.logs) != len(tests) {
		t.Fatalf("expected %d logs, got %d", len(tests), len(pane.logs))
	}

	for i, tt := range tests {
		if pane.logs[i].Level != tt.level {
			t.Errorf("log %d: expected level '%s', got '%s'", i, tt.level, pane.logs[i].Level)
		}
		if pane.logs[i].Message != tt.message {
			t.Errorf("log %d: expected message '%s', got '%s'", i, tt.message, pane.logs[i].Message)
		}
	}
}

func TestOutputPane_Clear(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	pane.AddLog("info", "Message 1")
	pane.AddLog("info", "Message 2")

	if len(pane.logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(pane.logs))
	}

	pane.Clear()

	if len(pane.logs) != 0 {
		t.Errorf("expected 0 logs after clear, got %d", len(pane.logs))
	}
}

func TestOutputPane_View_Empty(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	view := pane.View()
	// Empty viewport should return something (possibly empty or whitespace)
	_ = view // Just ensure no panic
}

func TestOutputPane_View_WithLogs(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	pane.AddLog("info", "Test message")

	view := pane.View()
	if !strings.Contains(view, "Test message") {
		t.Errorf("expected view to contain 'Test message', got '%s'", view)
	}
}

func TestOutputPane_RenderWithBorder(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	pane.AddLog("info", "Test message")

	rendered := pane.RenderWithBorder(ui.PrimaryColor)

	// Should contain the title
	if !strings.Contains(rendered, "Output") {
		t.Error("expected rendered output to contain title 'Output'")
	}

	// Should contain the log message
	if !strings.Contains(rendered, "Test message") {
		t.Errorf("expected rendered output to contain 'Test message', got: %s", rendered)
	}
}

func TestOutputPane_RenderWithBorder_WidthConsistency(t *testing.T) {
	// This test verifies the fix for the width calculation mismatch
	// between SetSize and RenderWithBorder
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	pane.AddLog("info", "A message that should not be truncated unexpectedly")

	rendered := pane.RenderWithBorder(ui.PrimaryColor)

	// The key test: viewport content should not be truncated due to width mismatch
	// We verify by checking that innerWidth is computed consistently
	expectedInnerW := pane.innerWidth()
	if pane.viewport.Width != expectedInnerW {
		t.Errorf("viewport width (%d) does not match innerWidth (%d)", pane.viewport.Width, expectedInnerW)
	}

	// Rendered content should contain the full message
	if !strings.Contains(rendered, "truncated unexpectedly") {
		t.Errorf("message appears to be truncated in rendered output")
	}
}

func TestOutputPane_HasContent(t *testing.T) {
	tests := []struct {
		name     string
		logs     int
		expected bool
	}{
		{"no logs", 0, false},
		{"one log", 1, true},
		{"multiple logs", 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pane := NewOutputPane("Output")
			pane.SetSize(80, 24)

			for i := 0; i < tt.logs; i++ {
				pane.AddLog("info", "message")
			}

			if pane.HasContent() != tt.expected {
				t.Errorf("expected HasContent() to be %v, got %v", tt.expected, pane.HasContent())
			}
		})
	}
}

func TestOutputPane_LogCount(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	if pane.LogCount() != 0 {
		t.Errorf("expected LogCount 0, got %d", pane.LogCount())
	}

	pane.AddLog("info", "1")
	pane.AddLog("info", "2")
	pane.AddLog("info", "3")

	if pane.LogCount() != 3 {
		t.Errorf("expected LogCount 3, got %d", pane.LogCount())
	}
}

func TestOutputPane_Update(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	// Test with a window size message
	cmd := pane.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Should not crash and may return a command
	_ = cmd
}

func TestOutputPane_InnerWidth(t *testing.T) {
	tests := []struct {
		width    int
		expected int
	}{
		{80, 78},
		{100, 98},
		{10, 8},
		{2, 0}, // Edge case
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			pane := NewOutputPane("Output")
			pane.width = tt.width
			if pane.innerWidth() != tt.expected {
				t.Errorf("for width %d, expected innerWidth %d, got %d", tt.width, tt.expected, pane.innerWidth())
			}
		})
	}
}

func TestOutputPane_InnerHeight(t *testing.T) {
	tests := []struct {
		height   int
		expected int
	}{
		{24, 21}, // 24 - 2 (border) - 1 (title)
		{30, 27},
		{10, 7},
		{3, 0}, // Edge case
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			pane := NewOutputPane("Output")
			pane.height = tt.height
			if pane.innerHeight() != tt.expected {
				t.Errorf("for height %d, expected innerHeight %d, got %d", tt.height, tt.expected, pane.innerHeight())
			}
		})
	}
}

func TestRenderPaneWithTitle(t *testing.T) {
	content := "Test content"
	title := "Title"

	rendered := renderPaneWithTitle(content, title, 40, 10, ui.PrimaryColor)

	if !strings.Contains(rendered, title) {
		t.Error("expected rendered pane to contain title")
	}
	if !strings.Contains(rendered, content) {
		t.Error("expected rendered pane to contain content")
	}
}

func TestRenderPaneWithTitle_MinimumDimensions(t *testing.T) {
	content := "Test"
	title := "T"

	// Very small dimensions - should not panic
	rendered := renderPaneWithTitle(content, title, 4, 4, lipgloss.Color("#FFFFFF"))
	_ = rendered
}

func TestOutputPane_FormatLogEntry(t *testing.T) {
	pane := NewOutputPane("Output")
	pane.SetSize(80, 24)

	tests := []struct {
		level       string
		message     string
		expectIcon  string
		expectColor bool
	}{
		{"success", "Success msg", "✓", true},
		{"error", "Error msg", "✗", true},
		{"warn", "Warn msg", "⚠", true},
		{"info", "Info msg", "•", true},
		{"unknown", "Unknown msg", "•", true}, // defaults to info style
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			entry := logEntry{
				Level:   tt.level,
				Message: tt.message,
			}

			formatted := pane.formatLogEntry(entry)

			if !strings.Contains(formatted, tt.message) {
				t.Errorf("expected formatted entry to contain message '%s', got '%s'", tt.message, formatted)
			}
			// The icon may be styled with ANSI codes, but should be present
			if !strings.Contains(formatted, tt.expectIcon) {
				t.Errorf("expected formatted entry to contain icon '%s', got '%s'", tt.expectIcon, formatted)
			}
		})
	}
}
