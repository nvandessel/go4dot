package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// SummaryPanel displays system stats (config count, sync status, platform, deps, source)
// This is a non-navigable panel that shows at-a-glance information
type SummaryPanel struct {
	BasePanel
	state         State
	selectedCount int
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

	labelStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)
	valueStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)

	var lines []string
	lines = append(lines, p.renderConfigLine(valueStyle, labelStyle))
	lines = append(lines, p.renderSyncLine(labelStyle))
	lines = append(lines, p.renderPlatformLine(valueStyle, labelStyle))
	lines = append(lines, p.renderDepsLine(labelStyle))
	lines = append(lines, p.renderSourceLine(labelStyle))

	// Filter empty lines
	var result []string
	for _, line := range lines {
		if line != "" {
			result = append(result, line)
		}
	}

	// Limit to available content height
	maxLines := p.ContentHeight()
	if maxLines < 1 {
		maxLines = 1
	}
	if len(result) > maxLines {
		result = result[:maxLines]
	}

	return lipgloss.JoinVertical(lipgloss.Left, result...)
}

// renderConfigLine shows config count or selection status
func (p *SummaryPanel) renderConfigLine(valueStyle, labelStyle lipgloss.Style) string {
	configCount := len(p.state.Configs)
	if p.selectedCount > 0 {
		return valueStyle.Render(fmt.Sprintf("%d", p.selectedCount)) +
			labelStyle.Render(fmt.Sprintf(" of %d selected", configCount))
	}
	return valueStyle.Render(fmt.Sprintf("%d", configCount)) + " " + labelStyle.Render("configs")
}

// renderSyncLine shows per-config sync status counts
func (p *SummaryPanel) renderSyncLine(labelStyle lipgloss.Style) string {
	if len(p.state.LinkStatus) == 0 && p.state.DriftSummary == nil {
		if !p.state.HasBaseline {
			return lipgloss.NewStyle().Foreground(ui.WarningColor).Render("Not synced")
		}
		return labelStyle.Render("No link data")
	}

	syncedCount, driftedCount, notInstalledCount := p.computeSyncCounts()

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)

	// If everything is synced with no issues, show a simple message
	if driftedCount == 0 && notInstalledCount == 0 {
		return okStyle.Render("All synced")
	}

	var parts []string
	if syncedCount > 0 {
		parts = append(parts, okStyle.Render(fmt.Sprintf("%d synced", syncedCount)))
	}
	if driftedCount > 0 {
		parts = append(parts, warnStyle.Render(fmt.Sprintf("%d drifted", driftedCount)))
	}
	if notInstalledCount > 0 {
		parts = append(parts, errStyle.Render(fmt.Sprintf("%d unlinked", notInstalledCount)))
	}
	return strings.Join(parts, labelStyle.Render(", "))
}

// computeSyncCounts categorizes configs into synced, drifted, and not installed
func (p *SummaryPanel) computeSyncCounts() (synced, drifted, notInstalled int) {
	driftMap := make(map[string]bool)
	if p.state.DriftSummary != nil {
		for _, r := range p.state.DriftSummary.Results {
			if r.HasDrift {
				driftMap[r.ConfigName] = true
			}
		}
	}

	for _, cfg := range p.state.Configs {
		ls, hasLink := p.state.LinkStatus[cfg.Name]
		hasDrift := driftMap[cfg.Name]

		switch {
		case hasDrift:
			drifted++
		case hasLink && ls.IsFullyLinked():
			synced++
		case hasLink && ls.LinkedCount > 0:
			drifted++
		default:
			notInstalled++
		}
	}
	return
}

// renderPlatformLine shows OS/distro and package manager
func (p *SummaryPanel) renderPlatformLine(valueStyle, labelStyle lipgloss.Style) string {
	if p.state.Platform == nil {
		return ""
	}
	plat := p.state.Platform
	var osInfo string
	if plat.Distro != "" {
		osInfo = fmt.Sprintf("%s/%s", plat.OS, plat.Distro)
	} else {
		osInfo = plat.OS
	}
	result := valueStyle.Render(osInfo)
	if plat.PackageManager != "" && plat.PackageManager != "unknown" && plat.PackageManager != "none" {
		result += labelStyle.Render(" (") + labelStyle.Render(plat.PackageManager) + labelStyle.Render(")")
	}
	return result
}

// renderDepsLine shows dependency count
func (p *SummaryPanel) renderDepsLine(labelStyle lipgloss.Style) string {
	if p.state.Config == nil {
		return ""
	}
	allDeps := p.state.Config.GetAllDependencies()
	if len(allDeps) == 0 {
		return ""
	}
	return labelStyle.Render(fmt.Sprintf("%d dependencies", len(allDeps)))
}

// renderSourceLine shows the dotfiles path, truncated if needed
func (p *SummaryPanel) renderSourceLine(labelStyle lipgloss.Style) string {
	if p.state.DotfilesPath == "" {
		return ""
	}
	path := p.state.DotfilesPath
	maxLen := p.ContentWidth()
	if maxLen < 5 {
		maxLen = 5
	}
	if len(path) > maxLen {
		path = "..." + path[len(path)-maxLen+3:]
	}
	return labelStyle.Render(path)
}

// GetSelectedItem implements Panel interface - summary is not navigable
func (p *SummaryPanel) GetSelectedItem() *SelectedItem {
	return nil
}

// UpdateState updates the panel's state reference
func (p *SummaryPanel) UpdateState(state State) {
	p.state = state
}

// SetSelectedCount updates the number of selected configs
func (p *SummaryPanel) SetSelectedCount(count int) {
	p.selectedCount = count
}
