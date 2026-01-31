package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// LogEntry represents a single log line with a level
type LogEntry struct {
	Level   string // "info", "success", "warning", "error"
	Message string
}

// OutputPane displays logs and operation output in a scrollable viewport
type OutputPane struct {
	viewport viewport.Model
	logs     []LogEntry
	width    int
	height   int
	title    string
}

// NewOutputPane creates a new output pane
func NewOutputPane() OutputPane {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()
	return OutputPane{
		viewport: vp,
		logs:     []LogEntry{},
		title:    "Output",
	}
}

// SetSize updates the pane dimensions
func (o *OutputPane) SetSize(width, height int) {
	o.width = width
	o.height = height
	// Account for border (2) and padding
	contentWidth := width - 4
	contentHeight := height - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	o.viewport.Width = contentWidth
	o.viewport.Height = contentHeight
	o.updateContent()
}

// SetTitle sets the pane title
func (o *OutputPane) SetTitle(title string) {
	o.title = title
}

// AddLog adds a log entry and scrolls to bottom
func (o *OutputPane) AddLog(level, message string) {
	o.logs = append(o.logs, LogEntry{Level: level, Message: message})
	o.updateContent()
	o.viewport.GotoBottom()
}

// Clear removes all logs
func (o *OutputPane) Clear() {
	o.logs = []LogEntry{}
	o.updateContent()
}

// updateContent rebuilds the viewport content from logs
func (o *OutputPane) updateContent() {
	var lines []string
	for _, log := range o.logs {
		line := o.formatLog(log)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	o.viewport.SetContent(content)
}

// formatLog formats a log entry with appropriate styling
func (o *OutputPane) formatLog(log LogEntry) string {
	var icon string
	var style lipgloss.Style

	switch log.Level {
	case "success":
		icon = "✓"
		style = lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	case "warning":
		icon = "⚠"
		style = lipgloss.NewStyle().Foreground(ui.WarningColor)
	case "error":
		icon = "✗"
		style = lipgloss.NewStyle().Foreground(ui.ErrorColor)
	default: // "info"
		icon = "•"
		style = lipgloss.NewStyle().Foreground(ui.SubtleColor)
	}

	return style.Render(fmt.Sprintf("%s %s", icon, log.Message))
}

// Update handles viewport scrolling including mouse events
func (o *OutputPane) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Forward mouse events to viewport for scrolling
		o.viewport, cmd = o.viewport.Update(msg)
		return cmd
	}

	o.viewport, cmd = o.viewport.Update(msg)
	return cmd
}

// View renders the output pane with inline header
func (o *OutputPane) View() string {
	if o.width < 5 || o.height < 3 {
		return ""
	}

	// Content area
	content := o.viewport.View()

	// If no logs, show placeholder
	if len(o.logs) == 0 {
		placeholderStyle := lipgloss.NewStyle().
			Foreground(ui.SubtleColor).
			Italic(true)
		content = placeholderStyle.Render("No output yet...")
	}

	return content
}

// RenderWithBorder renders the pane with a border and inline title
func (o *OutputPane) RenderWithBorder(focused bool) string {
	borderColor := ui.SubtleColor
	if focused {
		borderColor = ui.PrimaryColor
	}

	// Build the pane with inline title in border
	// Use o.width directly to match SetSize calculations
	return renderPaneWithTitle(o.View(), o.title, o.width, o.height, borderColor)
}

// renderPaneWithTitle creates a bordered pane with an inline title
func renderPaneWithTitle(content, title string, width, height int, borderColor lipgloss.Color) string {
	if width < 5 || height < 3 {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Foreground(borderColor)

	// Build top border with inline title: ╭─ Title ─────────╮
	titleText := titleStyle.Render(title)
	titleLen := lipgloss.Width(title)

	// Calculate dashes needed
	// Border chars: ╭ (1) + ─ (1) + space (1) + title + space (1) + ─... + ╮ (1)
	// Fixed chars = 5: ╭ + ─ + space before title + space after title + ╮
	leftDash := borderStyle.Render("─ ")
	rightPadding := width - 5 - titleLen
	if rightPadding < 0 {
		rightPadding = 0
	}
	rightDashes := borderStyle.Render(strings.Repeat("─", rightPadding))

	topBorder := borderStyle.Render("╭") + leftDash + titleText + " " + rightDashes + borderStyle.Render("╮")

	// Bottom border: ╰────────────────╯
	bottomDashes := strings.Repeat("─", width-2)
	bottomBorder := borderStyle.Render("╰" + bottomDashes + "╯")

	// Side borders with content
	lines := strings.Split(content, "\n")
	contentHeight := height - 2 // Subtract top and bottom borders

	// Pad or truncate lines to fit
	var middleLines []string
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		// Ensure line fits width (accounting for borders and padding)
		lineWidth := lipgloss.Width(line)
		innerWidth := width - 4 // -2 for borders, -2 for padding
		if lineWidth > innerWidth {
			// Truncate - this is simplistic, proper truncation would need rune handling
			line = line[:innerWidth]
		}
		padding := innerWidth - lipgloss.Width(line)
		if padding < 0 {
			padding = 0
		}
		paddedLine := " " + line + strings.Repeat(" ", padding) + " "
		middleLines = append(middleLines, borderStyle.Render("│")+paddedLine+borderStyle.Render("│"))
	}

	return topBorder + "\n" + strings.Join(middleLines, "\n") + "\n" + bottomBorder
}
