package dashboard

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// SummaryPanel displays system stats (config count, sync status)
// This is a non-navigable panel that shows at-a-glance information
type SummaryPanel struct {
	BasePanel
	state State
}

// NewSummaryPanel creates a new summary panel
func NewSummaryPanel(state State) *SummaryPanel {
	return &SummaryPanel{
		BasePanel: NewBasePanel(PanelSummary, "1 Summary"),
		state:     state,
	}
}

// Init implements Panel interface
func (p *SummaryPanel) Init() tea.Cmd {
	return nil
}

// Update implements Panel interface
func (p *SummaryPanel) Update(msg tea.Msg) tea.Cmd {
	// Summary panel doesn't handle any messages
	return nil
}

// View implements Panel interface
func (p *SummaryPanel) View() string {
	if p.width < 5 || p.height < 3 {
		return ""
	}

	var lines []string

	// Config count
	configCount := len(p.state.Configs)
	countStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	lines = append(lines, countStyle.Render(fmt.Sprintf("%d", configCount))+" "+labelStyle.Render("configs"))

	// Sync status
	var syncStatus string
	if !p.state.HasBaseline {
		syncStatus = lipgloss.NewStyle().Foreground(ui.WarningColor).Render("Not synced")
	} else if p.state.DriftSummary != nil && p.state.DriftSummary.HasDrift() {
		syncStatus = lipgloss.NewStyle().Foreground(ui.WarningColor).Render(fmt.Sprintf("%d drift", p.state.DriftSummary.DriftedConfigs))
	} else {
		syncStatus = lipgloss.NewStyle().Foreground(ui.SecondaryColor).Render("Synced")
	}
	lines = append(lines, syncStatus)

	// Linked count
	linkedCount := 0
	totalLinks := 0
	for _, ls := range p.state.LinkStatus {
		linkedCount += ls.LinkedCount
		totalLinks += ls.TotalCount
	}
	if totalLinks > 0 {
		linkStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)
		lines = append(lines, linkStyle.Render(fmt.Sprintf("%d/%d links", linkedCount, totalLinks)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// GetSelectedItem implements Panel interface - summary is not navigable
func (p *SummaryPanel) GetSelectedItem() *SelectedItem {
	return nil
}

// UpdateState updates the panel's state reference
func (p *SummaryPanel) UpdateState(state State) {
	p.state = state
}
