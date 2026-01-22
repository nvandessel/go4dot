package dashboard

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Model is the model for the header component.
type Header struct {
	state State
}

// New creates a new header component.
func NewHeader(s State) Header {
	return Header{state: s}
}

// View renders the header.
func (h Header) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.PrimaryColor).
		Bold(true).
		Padding(0, 2)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		MarginLeft(1)

	title := titleStyle.Render("GO4DOT DASHBOARD")

	platformInfo := ""
	if h.state.Platform != nil {
		platformInfo = fmt.Sprintf("%s (%s)", h.state.Platform.OS, h.state.Platform.PackageManager)
	}

	subtitle := subtitleStyle.Render(platformInfo)

	updateInfo := ""
	if h.state.UpdateMsg != "" {
		updateInfo = lipgloss.NewStyle().
			Foreground(ui.SecondaryColor).
			Bold(true).
			MarginLeft(2).
			Render(h.state.UpdateMsg)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle, updateInfo)
}
