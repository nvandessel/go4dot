package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// LogEntry represents a log entry with level and message
type LogEntry struct {
	Level   string
	Message string
}

// OutputPanel displays logs and operation output in a scrollable viewport
// This is a scrollable panel when focused
type OutputPanel struct {
	BasePanel
	viewport viewport.Model
	logs     []LogEntry
	ready    bool
}

// NewOutputPanel creates a new output panel
func NewOutputPanel() *OutputPanel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	return &OutputPanel{
		BasePanel: NewBasePanel(PanelOutput, "Output"),
		viewport:  vp,
		logs:      []LogEntry{},
	}
}

// Init implements Panel interface
func (p *OutputPanel) Init() tea.Cmd {
	return nil
}

// SetSize implements Panel interface
func (p *OutputPanel) SetSize(width, height int) {
	p.BasePanel.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()

	p.viewport.Width = contentWidth
	p.viewport.Height = contentHeight
	p.ready = true
	p.updateContent()
}

// Update implements Panel interface
func (p *OutputPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.ready {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		p.viewport, cmd = p.viewport.Update(msg)
		return cmd
	case tea.KeyMsg:
		if p.focused {
			p.viewport, cmd = p.viewport.Update(msg)
			return cmd
		}
	}

	return nil
}

// View implements Panel interface
func (p *OutputPanel) View() string {
	if !p.ready {
		return ""
	}

	if len(p.logs) == 0 {
		placeholderStyle := lipgloss.NewStyle().
			Foreground(ui.SubtleColor).
			Italic(true)
		return placeholderStyle.Render("No output yet...")
	}

	return p.viewport.View()
}

// GetSelectedItem implements Panel interface - output doesn't have selection
func (p *OutputPanel) GetSelectedItem() *SelectedItem {
	return nil
}

// AddLog adds a log entry and scrolls to bottom
func (p *OutputPanel) AddLog(level, message string) {
	p.logs = append(p.logs, LogEntry{Level: level, Message: message})
	p.updateContent()
	p.viewport.GotoBottom()
}

// Clear removes all logs
func (p *OutputPanel) Clear() {
	p.logs = []LogEntry{}
	p.updateContent()
}

// updateContent rebuilds the viewport content from logs
func (p *OutputPanel) updateContent() {
	var lines []string
	for _, log := range p.logs {
		line := p.formatLog(log)
		lines = append(lines, line)
	}

	content := strings.Join(lines, "\n")
	p.viewport.SetContent(content)
}

// formatLog formats a log entry with appropriate styling
func (p *OutputPanel) formatLog(log LogEntry) string {
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

// GetLogCount returns the number of log entries
func (p *OutputPanel) GetLogCount() int {
	return len(p.logs)
}

// GetLogs returns a copy of all log entries
func (p *OutputPanel) GetLogs() []LogEntry {
	logs := make([]LogEntry, len(p.logs))
	copy(logs, p.logs)
	return logs
}

// ScrollToTop scrolls the viewport to the top
func (p *OutputPanel) ScrollToTop() {
	p.viewport.GotoTop()
}

// ScrollToBottom scrolls the viewport to the bottom
func (p *OutputPanel) ScrollToBottom() {
	p.viewport.GotoBottom()
}
