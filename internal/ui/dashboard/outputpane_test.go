package dashboard

import (
	"strings"
	"testing"
)

func TestOutputPane_New(t *testing.T) {
	pane := NewOutputPane()

	if pane.title != "Output" {
		t.Errorf("expected title 'Output', got %q", pane.title)
	}
	if len(pane.logs) != 0 {
		t.Errorf("expected empty logs, got %d", len(pane.logs))
	}
}

func TestOutputPane_SetSize(t *testing.T) {
	pane := NewOutputPane()

	pane.SetSize(80, 20)

	if pane.width != 80 {
		t.Errorf("expected width 80, got %d", pane.width)
	}
	if pane.height != 20 {
		t.Errorf("expected height 20, got %d", pane.height)
	}
}

func TestOutputPane_AddLog(t *testing.T) {
	pane := NewOutputPane()
	pane.SetSize(80, 20)

	pane.AddLog("info", "Test message")

	if len(pane.logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(pane.logs))
	}
	if pane.logs[0].Level != "info" {
		t.Errorf("expected level 'info', got %q", pane.logs[0].Level)
	}
	if pane.logs[0].Message != "Test message" {
		t.Errorf("expected message 'Test message', got %q", pane.logs[0].Message)
	}
}

func TestOutputPane_AddLog_MultipleLevels(t *testing.T) {
	pane := NewOutputPane()
	pane.SetSize(80, 20)

	pane.AddLog("info", "Info message")
	pane.AddLog("success", "Success message")
	pane.AddLog("warning", "Warning message")
	pane.AddLog("error", "Error message")

	if len(pane.logs) != 4 {
		t.Errorf("expected 4 logs, got %d", len(pane.logs))
	}
}

func TestOutputPane_Clear(t *testing.T) {
	pane := NewOutputPane()
	pane.SetSize(80, 20)

	pane.AddLog("info", "Test message")
	pane.Clear()

	if len(pane.logs) != 0 {
		t.Errorf("expected 0 logs after clear, got %d", len(pane.logs))
	}
}

func TestOutputPane_SetTitle(t *testing.T) {
	pane := NewOutputPane()

	pane.SetTitle("Custom Title")

	if pane.title != "Custom Title" {
		t.Errorf("expected title 'Custom Title', got %q", pane.title)
	}
}

func TestOutputPane_View_Empty(t *testing.T) {
	pane := NewOutputPane()
	pane.SetSize(80, 20)

	view := pane.View()

	if !strings.Contains(view, "No output yet") {
		t.Error("expected placeholder text for empty pane")
	}
}

func TestOutputPane_View_WithLogs(t *testing.T) {
	pane := NewOutputPane()
	pane.SetSize(80, 20)

	pane.AddLog("success", "Operation completed")

	view := pane.View()

	// The view should contain the log content (rendered through viewport)
	if view == "" {
		t.Error("expected non-empty view with logs")
	}
}

func TestOutputPane_RenderWithBorder(t *testing.T) {
	pane := NewOutputPane()
	pane.SetSize(40, 10)
	pane.SetTitle("Test")

	rendered := pane.RenderWithBorder(false)

	// Should contain border characters
	if !strings.Contains(rendered, "╭") {
		t.Error("expected top-left corner in border")
	}
	if !strings.Contains(rendered, "╯") {
		t.Error("expected bottom-right corner in border")
	}
	if !strings.Contains(rendered, "Test") {
		t.Error("expected title in border")
	}
}

func TestRenderPaneWithTitle(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		title    string
		width    int
		height   int
		wantErr  bool
	}{
		{
			name:    "normal pane",
			content: "Hello",
			title:   "Title",
			width:   20,
			height:  5,
		},
		{
			name:    "too small width",
			content: "Hello",
			title:   "Title",
			width:   3,
			height:  5,
		},
		{
			name:    "too small height",
			content: "Hello",
			title:   "Title",
			width:   20,
			height:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderPaneWithTitle(tt.content, tt.title, tt.width, tt.height, "#FFFFFF")
			// Just verify it doesn't panic
			_ = result
		})
	}
}
