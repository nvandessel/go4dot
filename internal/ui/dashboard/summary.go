package dashboard

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Summary is the model for the summary component.
type Summary struct {
	state State
}

// NewSummary creates a new summary component.
func NewSummary(s State) Summary {
	return Summary{state: s}
}

// View renders the summary.
func (s Summary) View() string {
	var status string

	// If we haven't synced before (no baseline), show that clearly
	if !s.state.HasBaseline {
		status = lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render("  Not synced yet - press [s] to create symlinks")
	} else if s.state.DriftSummary != nil && s.state.DriftSummary.HasDrift() {
		// We have baseline - check for drift
		status = lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render(fmt.Sprintf("  %d config(s) need syncing", s.state.DriftSummary.DriftedConfigs))
	} else {
		// Baseline exists and no drift
		status = lipgloss.NewStyle().
			Foreground(ui.SecondaryColor).
			Bold(true).
			Render("  All synced")
	}

	// TODO: Add back selected configs count
	// if len(m.selectedConfigs) > 0 {
	// 	selectionInfo := lipgloss.NewStyle().
	// 		Foreground(ui.PrimaryColor).
	// 		Bold(true).
	// 		Render(fmt.Sprintf(" • %d selected", len(m.selectedConfigs)))
	// 	return status + selectionInfo
	// }

	return status
}
