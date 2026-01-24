package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Footer is the model for the footer component.
type Footer struct {
	width int
}

// NewFooter creates a new footer component.
func NewFooter() Footer {
	return Footer{}
}

// View renders the footer.
func (f Footer) View() string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	type action struct {
		key      string
		label    string
		priority int // Lower is higher priority
	}

	allActions := []action{
		{"?", "Help", 0},
		{"q", "Quit", 0},
		{"s", "Sync All", 1},
		{"/", "Filter", 1},
		{"space", "Select", 2},
		{"i", "Install", 3},
		{"u", "Update", 3},
		{"d", "Doctor", 4},
		{"m", "Overrides", 4},
		{"tab", "More", 5},
	}

	var visibleActions []string
	currentWidth := 0
	margin := 3

	for _, a := range allActions {
		rendered := keyStyle.Render("["+a.key+"]") + " " + descStyle.Render(a.label)
		width := lipgloss.Width(rendered)

		if currentWidth+width+margin > f.width && len(visibleActions) > 0 {
			if a.priority > 0 {
				continue
			}
			if currentWidth+width > f.width {
				break
			}
		}

		visibleActions = append(visibleActions, rendered)
		currentWidth += width + margin
	}

	return strings.Join(visibleActions, "   ")
}
