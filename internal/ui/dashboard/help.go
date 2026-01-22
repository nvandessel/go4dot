package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Help is the model for the help component.
type Help struct {
	width  int
	height int
}

// NewHelp creates a new help component.
func NewHelp() Help {
	return Help{}
}

// View renders the help overlay.
func (h Help) View() string {
	var b strings.Builder

	boxWidth := 60
	if h.width > 0 && h.width < boxWidth+4 {
		boxWidth = h.width - 4
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(boxWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Foreground(ui.SecondaryColor).
		Bold(true).
		MarginTop(1).
		MarginLeft(2)

	keyStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(14).
		Align(lipgloss.Right)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		MarginLeft(2)

	subtleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Width(boxWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	b.WriteString(titleStyle.Render("go4dot Dashboard - Keyboard Shortcuts"))
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("↑/k"), descStyle.Render("Move selection up")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("↓/j"), descStyle.Render("Move selection down")))

	b.WriteString(headerStyle.Render("Actions"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("enter"), descStyle.Render("Sync selected config")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("s"), descStyle.Render("Sync all configs")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("shift+s"), descStyle.Render("Sync selected configs")))

	b.WriteString(headerStyle.Render("Selection & Filter"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("space"), descStyle.Render("Toggle selection")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("shift+a"), descStyle.Render("Select/deselect all visible")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("/"), descStyle.Render("Enter filter mode")))

	b.WriteString(headerStyle.Render("Other"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("d"), descStyle.Render("Run doctor check")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("m"), descStyle.Render("Configure overrides")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("tab"), descStyle.Render("More commands menu")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("?"), descStyle.Render("Toggle help screen")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("q / esc"), descStyle.Render("Quit dashboard")))

	b.WriteString(subtleStyle.Render("Press ?, q, or esc to close"))

	return ui.BoxStyle.
		Width(boxWidth).
		Render(b.String())
}
