package dashboard

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// SelectedItem represents the currently selected item in a navigable panel
type SelectedItem struct {
	ID    string // Unique identifier
	Name  string // Display name
	Index int    // Index in the list
}

// Panel is the interface that all dashboard panels implement
type Panel interface {
	// Init initializes the panel and returns any initial commands
	Init() tea.Cmd

	// Update handles messages and returns commands
	Update(msg tea.Msg) tea.Cmd

	// View renders the panel content (without border)
	View() string

	// SetSize updates the panel dimensions
	SetSize(width, height int)

	// IsFocused returns whether this panel is currently focused
	IsFocused() bool

	// SetFocused sets the focus state of the panel
	SetFocused(focused bool)

	// GetSelectedItem returns the currently selected item (for navigable panels)
	// Returns nil if the panel has no selection or is not navigable
	GetSelectedItem() *SelectedItem

	// GetTitle returns the panel's title for display
	GetTitle() string

	// GetID returns the panel's identifier
	GetID() PanelID
}

// BasePanel provides common functionality for all panels
type BasePanel struct {
	id      PanelID
	title   string
	width   int
	height  int
	focused bool
}

// NewBasePanel creates a base panel with the given ID and title
func NewBasePanel(id PanelID, title string) BasePanel {
	return BasePanel{
		id:    id,
		title: title,
	}
}

// GetID returns the panel identifier
func (b *BasePanel) GetID() PanelID {
	return b.id
}

// GetTitle returns the panel title
func (b *BasePanel) GetTitle() string {
	return b.title
}

// SetTitle updates the panel title
func (b *BasePanel) SetTitle(title string) {
	b.title = title
}

// SetSize updates the panel dimensions
func (b *BasePanel) SetSize(width, height int) {
	b.width = width
	b.height = height
}

// GetWidth returns the panel width
func (b *BasePanel) GetWidth() int {
	return b.width
}

// GetHeight returns the panel height
func (b *BasePanel) GetHeight() int {
	return b.height
}

// IsFocused returns whether this panel is focused
func (b *BasePanel) IsFocused() bool {
	return b.focused
}

// SetFocused sets the focus state
func (b *BasePanel) SetFocused(focused bool) {
	b.focused = focused
}

// ContentWidth returns the usable content width (accounting for borders and padding)
func (b *BasePanel) ContentWidth() int {
	w := b.width - 4 // 2 for borders, 2 for padding
	if w < 1 {
		return 1
	}
	return w
}

// ContentHeight returns the usable content height (accounting for borders)
func (b *BasePanel) ContentHeight() int {
	h := b.height - 2 // 2 for top/bottom borders
	if h < 1 {
		return 1
	}
	return h
}

// RenderPanelFrame renders content inside a panel frame with title and focus styling
func RenderPanelFrame(content, title string, width, height int, focused bool) string {
	if width < 5 || height < 3 {
		return ""
	}

	borderColor := ui.SubtleColor
	if focused {
		borderColor = ui.PrimaryColor
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Foreground(borderColor)

	// Build top border with inline title: ╭─ Title ─────────╮
	titleText := titleStyle.Render(title)
	titleLen := lipgloss.Width(title)

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

	var middleLines []string
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		// Ensure line fits width (accounting for borders and padding)
		lineWidth := lipgloss.Width(line)
		innerWidth := width - 4 // -2 for borders, -2 for padding
		if innerWidth < 0 {
			innerWidth = 0
		}
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

// RenderPanelFrameCompact renders a more compact frame for mini panels
func RenderPanelFrameCompact(content, title string, width, height int, focused bool) string {
	if width < 3 || height < 2 {
		return ""
	}

	borderColor := ui.SubtleColor
	if focused {
		borderColor = ui.PrimaryColor
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(borderColor).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Foreground(borderColor)

	// Shorter title style for mini panels
	displayTitle := title
	maxTitleLen := width - 4
	if maxTitleLen < 3 {
		maxTitleLen = 3
	}
	if len(displayTitle) > maxTitleLen {
		displayTitle = displayTitle[:maxTitleLen-1] + "…"
	}

	titleText := titleStyle.Render(displayTitle)
	titleLen := lipgloss.Width(displayTitle)

	rightPadding := width - 4 - titleLen
	if rightPadding < 0 {
		rightPadding = 0
	}
	rightDashes := borderStyle.Render(strings.Repeat("─", rightPadding))

	topBorder := borderStyle.Render("╭─") + titleText + " " + rightDashes + borderStyle.Render("╮")

	bottomDashes := strings.Repeat("─", width-2)
	bottomBorder := borderStyle.Render("╰" + bottomDashes + "╯")

	lines := strings.Split(content, "\n")
	contentHeight := height - 2

	var middleLines []string
	for i := 0; i < contentHeight; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}
		lineWidth := lipgloss.Width(line)
		innerWidth := width - 2
		if innerWidth < 0 {
			innerWidth = 0
		}
		if lineWidth > innerWidth {
			line = line[:innerWidth]
		}
		padding := innerWidth - lipgloss.Width(line)
		if padding < 0 {
			padding = 0
		}
		middleLines = append(middleLines, borderStyle.Render("│")+line+strings.Repeat(" ", padding)+borderStyle.Render("│"))
	}

	return topBorder + "\n" + strings.Join(middleLines, "\n") + "\n" + bottomBorder
}
