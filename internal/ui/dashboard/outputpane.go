package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// logEntry represents a single log entry in the output pane.
type logEntry struct {
	Level     string
	Message   string
	Timestamp time.Time
}

// OutputPane displays operation output and logs with a scrollable viewport.
type OutputPane struct {
	viewport viewport.Model
	logs     []logEntry
	title    string
	width    int
	height   int
	ready    bool
}

// NewOutputPane creates a new OutputPane.
func NewOutputPane(title string) OutputPane {
	return OutputPane{
		title: title,
		logs:  make([]logEntry, 0),
	}
}

// borderWidth is the total horizontal border width (1 on each side).
const borderWidth = 2

// titleHeight accounts for the title line and border.
const titleHeight = 1

// innerWidth returns the width available for content inside the border.
// This is used by both SetSize and RenderWithBorder to ensure consistency.
func (o *OutputPane) innerWidth() int {
	return o.width - borderWidth
}

// innerHeight returns the height available for content inside the border.
func (o *OutputPane) innerHeight() int {
	return o.height - borderWidth - titleHeight
}

// SetSize sets the dimensions of the output pane.
// The viewport is sized to fit within the border using consistent calculations.
func (o *OutputPane) SetSize(width, height int) {
	o.width = width
	o.height = height

	innerW := o.innerWidth()
	innerH := o.innerHeight()

	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	if !o.ready {
		o.viewport = viewport.New(innerW, innerH)
		o.viewport.YPosition = 0
		o.ready = true
	} else {
		o.viewport.Width = innerW
		o.viewport.Height = innerH
	}

	o.updateContent()
}

// AddLog adds a log entry to the output pane.
func (o *OutputPane) AddLog(level, message string) {
	o.logs = append(o.logs, logEntry{
		Level:     level,
		Message:   message,
		Timestamp: time.Now(),
	})
	o.updateContent()
	// Scroll to bottom
	o.viewport.GotoBottom()
}

// Clear clears all log entries.
func (o *OutputPane) Clear() {
	o.logs = make([]logEntry, 0)
	o.updateContent()
}

// updateContent rebuilds the viewport content from logs.
func (o *OutputPane) updateContent() {
	if !o.ready {
		return
	}

	var lines []string
	for _, entry := range o.logs {
		line := o.formatLogEntry(entry)
		lines = append(lines, line)
	}

	o.viewport.SetContent(strings.Join(lines, "\n"))
}

// formatLogEntry formats a single log entry for display.
func (o *OutputPane) formatLogEntry(entry logEntry) string {
	var icon string
	var style lipgloss.Style

	switch entry.Level {
	case "success":
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	case "error":
		icon = "✗"
		style = lipgloss.NewStyle().Foreground(ui.ErrorColor)
	case "warn":
		icon = "⚠"
		style = lipgloss.NewStyle().Foreground(ui.WarningColor)
	default:
		icon = "•"
		style = lipgloss.NewStyle().Foreground(ui.SubtleColor)
	}

	timestamp := entry.Timestamp.Format("15:04:05")
	prefix := fmt.Sprintf("%s %s", style.Render(icon), ui.SubtleStyle.Render(timestamp))

	return fmt.Sprintf("%s %s", prefix, entry.Message)
}

// Update handles messages for the output pane.
func (o *OutputPane) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if o.ready {
		o.viewport, cmd = o.viewport.Update(msg)
	}
	return cmd
}

// View returns the viewport content without borders.
func (o *OutputPane) View() string {
	if !o.ready {
		return ""
	}
	return o.viewport.View()
}

// RenderWithBorder renders the output pane with a titled border.
// Uses the same inner width calculation as SetSize to prevent content truncation.
func (o *OutputPane) RenderWithBorder(borderColor lipgloss.Color) string {
	// IMPORTANT: Use the same width calculation as SetSize.
	// Previously this subtracted an extra 2, causing width mismatch with the viewport.
	return renderPaneWithTitle(o.View(), o.title, o.width, o.height, borderColor)
}

// renderPaneWithTitle renders content with a title bar and border.
func renderPaneWithTitle(content, title string, width, height int, borderColor lipgloss.Color) string {
	// Calculate inner dimensions
	innerW := width - borderWidth
	innerH := height - borderWidth - titleHeight

	if innerW < 1 {
		innerW = 1
	}
	if innerH < 1 {
		innerH = 1
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(borderColor).
		Bold(true).
		Padding(0, 1).
		Width(innerW)

	// Content style
	contentStyle := lipgloss.NewStyle().
		Width(innerW).
		Height(innerH)

	// Combine title and content
	inner := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		contentStyle.Render(content),
	)

	// Border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)

	return borderStyle.Render(inner)
}

// HasContent returns true if there are any log entries.
func (o *OutputPane) HasContent() bool {
	return len(o.logs) > 0
}

// LogCount returns the number of log entries.
func (o *OutputPane) LogCount() int {
	return len(o.logs)
}
