package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Footer is the model for the footer component.
type Footer struct {
	width        int
	focusedPanel PanelID
	platform     *platform.Platform
	updateMsg    string
}

// NewFooter creates a new footer component.
func NewFooter() Footer {
	return Footer{
		focusedPanel: PanelConfigs,
	}
}

// SetPlatform sets the platform info to display
func (f *Footer) SetPlatform(p *platform.Platform) {
	f.platform = p
}

// SetUpdateMsg sets the update message to display
func (f *Footer) SetUpdateMsg(msg string) {
	f.updateMsg = msg
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
		action{"0-6", "Jump", 4},
		action{"ctrl+hjkl", "Move", 5},
	)

	// Build header info for right side
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.PrimaryColor).
		Bold(true).
		Padding(0, 1)

	platformStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		MarginLeft(1)

	headerInfo := titleStyle.Render("GO4DOT DASHBOARD")

	if f.platform != nil {
		platformInfo := f.platform.OS
		if f.platform.PackageManager != "" {
			platformInfo = fmt.Sprintf("%s (%s)", f.platform.OS, f.platform.PackageManager)
		}
		headerInfo += platformStyle.Render(platformInfo)
	}

	if f.updateMsg != "" {
		updateStyle := lipgloss.NewStyle().
			Foreground(ui.SecondaryColor).
			Bold(true).
			MarginLeft(1)
		headerInfo += updateStyle.Render(f.updateMsg)
	}

	headerWidth := lipgloss.Width(headerInfo)

	// Reserve space for header info on the right
	availableWidth := f.width - headerWidth - 4 // padding between shortcuts and header

	var visibleActions []string
	currentWidth := 0
	margin := 3

	for _, a := range allActions {
		rendered := keyStyle.Render("["+a.key+"]") + " " + descStyle.Render(a.label)
		width := lipgloss.Width(rendered)

		if currentWidth+width+margin > availableWidth && len(visibleActions) > 0 {
			if a.priority > 0 {
				continue
			}
			if currentWidth+width > availableWidth {
				break
			}
		}

		visibleActions = append(visibleActions, rendered)
		currentWidth += width + margin
	}

	shortcuts := strings.Join(visibleActions, "   ")
	shortcutsWidth := lipgloss.Width(shortcuts)

	// Add padding between shortcuts and header info
	padding := f.width - shortcutsWidth - headerWidth
	if padding < 1 {
		padding = 1
	}

	return shortcuts + strings.Repeat(" ", padding) + headerInfo
}
