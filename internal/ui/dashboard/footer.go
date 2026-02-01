package dashboard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Footer is the model for the footer component.
type Footer struct {
	width       int
	focusedPanel PanelID
}

// NewFooter creates a new footer component.
func NewFooter() Footer {
	return Footer{
		focusedPanel: PanelConfigs,
	}
}

// SetFocusedPanel updates which panel is focused for context-sensitive hints
func (f *Footer) SetFocusedPanel(panel PanelID) {
	f.focusedPanel = panel
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

	// Base actions always shown
	allActions := []action{
		{"?", "Help", 0},
		{"q", "Quit", 0},
		{"tab", "Panel", 1},
	}

	// Context-sensitive actions based on focused panel
	switch f.focusedPanel {
	case PanelConfigs:
		allActions = append(allActions,
			action{"enter", "Sync", 1},
			action{"space", "Select", 2},
			action{"/", "Filter", 2},
			action{"s", "Sync All", 3},
		)
	case PanelHealth:
		allActions = append(allActions,
			action{"enter", "Refresh", 1},
			action{"↑↓", "Navigate", 2},
		)
	case PanelOverrides:
		allActions = append(allActions,
			action{"enter", "Configure", 1},
			action{"↑↓", "Navigate", 2},
		)
	case PanelExternal:
		allActions = append(allActions,
			action{"enter", "Clone/Update", 1},
			action{"↑↓", "Navigate", 2},
		)
	case PanelDetails, PanelOutput:
		allActions = append(allActions,
			action{"↑↓", "Scroll", 2},
		)
	default:
		allActions = append(allActions,
			action{"s", "Sync", 2},
			action{"i", "Install", 3},
		)
	}

	// Global shortcuts at lower priority
	allActions = append(allActions,
		action{"1-7", "Jump", 4},
		action{"ctrl+hjkl", "Move", 5},
	)

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
