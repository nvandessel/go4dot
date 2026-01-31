package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// ConfirmResult is sent when a confirmation dialog is resolved
type ConfirmResult struct {
	ID        string
	Confirmed bool
}

// Confirm is a simple yes/no confirmation dialog overlay
type Confirm struct {
	id          string
	title       string
	description string
	affirmative string
	negative    string
	width       int
	height      int
	selected    int // 0 = yes, 1 = no
}

// NewConfirm creates a new confirmation dialog
func NewConfirm(id, title, description string) *Confirm {
	return &Confirm{
		id:          id,
		title:       title,
		description: description,
		affirmative: "Yes",
		negative:    "No",
		selected:    1, // Default to "No" for safety
	}
}

// WithLabels sets custom button labels
func (c *Confirm) WithLabels(affirmative, negative string) *Confirm {
	c.affirmative = affirmative
	c.negative = negative
	return c
}

// SetSize updates the dialog dimensions
func (c *Confirm) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Init initializes the confirmation dialog
func (c *Confirm) Init() tea.Cmd {
	return nil
}

// Update handles messages for the confirmation dialog
func (c *Confirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			c.selected = 0
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			c.selected = 1
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			c.selected = (c.selected + 1) % 2
		case key.Matches(msg, key.NewBinding(key.WithKeys("y"))):
			return c, func() tea.Msg {
				return ConfirmResult{ID: c.id, Confirmed: true}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("n", "esc"))):
			return c, func() tea.Msg {
				return ConfirmResult{ID: c.id, Confirmed: false}
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			return c, func() tea.Msg {
				return ConfirmResult{ID: c.id, Confirmed: c.selected == 0}
			}
		}
	}
	return c, nil
}

// View renders the confirmation dialog
func (c *Confirm) View() string {
	dialogWidth := 50
	if c.width > 0 && c.width < dialogWidth+20 {
		dialogWidth = c.width - 20
		if dialogWidth < 30 {
			dialogWidth = 30
		}
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	selectedBtnStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.PrimaryColor).
		Padding(0, 3).
		Bold(true)

	normalBtnStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Padding(0, 3)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Padding(1, 2).
		Width(dialogWidth)

	// Build buttons
	var yesBtn, noBtn string
	if c.selected == 0 {
		yesBtn = selectedBtnStyle.Render(c.affirmative)
		noBtn = normalBtnStyle.Render(c.negative)
	} else {
		yesBtn = normalBtnStyle.Render(c.affirmative)
		noBtn = selectedBtnStyle.Render(c.negative)
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "  ", noBtn)
	buttonsRow := lipgloss.NewStyle().Width(dialogWidth - 4).Align(lipgloss.Center).Render(buttons)

	// Build dialog content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(c.title),
		"",
		descStyle.Render(c.description),
		"",
		buttonsRow,
	)

	dialog := borderStyle.Render(content)

	// Center in available space
	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#333333")),
	)
}

// ID returns the confirmation dialog identifier
func (c *Confirm) ID() string {
	return c.id
}
