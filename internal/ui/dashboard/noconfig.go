package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// NoConfig is the model for the no config component.
type NoConfig struct {
	selectedIdx int
}

// NewNoConfig creates a new no config component.
func NewNoConfig() NoConfig {
	return NoConfig{}
}

// View renders the no config view.
func (m NoConfig) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	b.WriteString(titleStyle.Render("go4dot"))
	b.WriteString("\n\n")

	statusStyle := lipgloss.NewStyle().
		Foreground(ui.WarningColor).
		Bold(true)
	b.WriteString(statusStyle.Render("  No configuration found"))
	b.WriteString("\n\n")

	b.WriteString(ui.TextStyle.Render("  No .go4dot.yaml found in current directory."))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("  Initialize go4dot to start managing your dotfiles."))
	b.WriteString("\n\n")

	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle

	options := []struct {
		label string
		desc  string
	}{
		{"Initialize go4dot", "Set up a new .go4dot.yaml config"},
		{"Quit", "Exit go4dot"},
	}

	for i, opt := range options {
		prefix := "  "
		style := normalStyle
		if i == m.selectedIdx {
			prefix = "> "
			style = selectedStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(opt.label)))
		if i == m.selectedIdx {
			b.WriteString(subtitleStyle.Render(fmt.Sprintf("    %s", opt.desc)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	actions := []string{
		keyStyle.Render("[i]") + subtitleStyle.Render(" Initialize"),
		keyStyle.Render("[q]") + subtitleStyle.Render(" Quit"),
	}
	b.WriteString(strings.Join(actions, "   "))

	return b.String()
}
