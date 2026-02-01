package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// ConfigsPanel displays the main config list (current sidebar functionality)
// This is a navigable panel with selection support
type ConfigsPanel struct {
	BasePanel
	state        State
	selectedIdx  int
	listOffset   int
	filteredIdxs []int
	selected     map[string]bool // Multi-select state
}

// NewConfigsPanel creates a new configs panel
func NewConfigsPanel(state State, selected map[string]bool) *ConfigsPanel {
	filteredIdxs := make([]int, len(state.Configs))
	for i := range state.Configs {
		filteredIdxs[i] = i
	}

	if selected == nil {
		selected = make(map[string]bool)
	}

	return &ConfigsPanel{
		BasePanel:    NewBasePanel(PanelConfigs, "Configs"),
		state:        state,
		filteredIdxs: filteredIdxs,
		selected:     selected,
	}
}

// Init implements Panel interface
func (p *ConfigsPanel) Init() tea.Cmd {
	return nil
}

// Update implements Panel interface
func (p *ConfigsPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused {
			return nil
		}

		currentPos := -1
		for i, idx := range p.filteredIdxs {
			if idx == p.selectedIdx {
				currentPos = i
				break
			}
		}

		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if currentPos > 0 {
				p.selectedIdx = p.filteredIdxs[currentPos-1]
				p.ensureVisible()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if currentPos < len(p.filteredIdxs)-1 {
				p.selectedIdx = p.filteredIdxs[currentPos+1]
				p.ensureVisible()
			}
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if p.listOffset > 0 {
				p.listOffset--
			}
		case tea.MouseButtonWheelDown:
			maxOffset := len(p.filteredIdxs) - p.ContentHeight()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if p.listOffset < maxOffset {
				p.listOffset++
			}
		case tea.MouseButtonLeft:
			if msg.Action == tea.MouseActionRelease {
				clickedLine := msg.Y - 2
				clickedIdx := p.listOffset + clickedLine
				if clickedIdx >= 0 && clickedIdx < len(p.filteredIdxs) {
					p.selectedIdx = p.filteredIdxs[clickedIdx]
				}
			}
		}
	}

	return nil
}

func (p *ConfigsPanel) ensureVisible() {
	currentPos := -1
	for i, idx := range p.filteredIdxs {
		if idx == p.selectedIdx {
			currentPos = i
			break
		}
	}

	visibleHeight := p.ContentHeight()
	if visibleHeight <= 0 || currentPos == -1 {
		return
	}
	if currentPos < p.listOffset {
		p.listOffset = currentPos
	} else if currentPos >= p.listOffset+visibleHeight {
		p.listOffset = currentPos - visibleHeight + 1
	}
}

// View implements Panel interface
func (p *ConfigsPanel) View() string {
	var lines []string

	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle.Width(p.ContentWidth())
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)

	// Build a map of drift results for quick lookup
	driftMap := make(map[string]*stow.DriftResult)
	if p.state.DriftSummary != nil {
		for i := range p.state.DriftSummary.Results {
			r := &p.state.DriftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}

	// Calculate visible range
	visibleHeight := p.ContentHeight()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	endIdx := p.listOffset + visibleHeight
	if endIdx > len(p.filteredIdxs) {
		endIdx = len(p.filteredIdxs)
	}

	for i := p.listOffset; i < endIdx; i++ {
		idx := p.filteredIdxs[i]
		cfg := p.state.Configs[idx]

		prefix := "  "
		if idx == p.selectedIdx && p.focused {
			prefix = "> "
		}

		checkbox := "[ ]"
		if p.selected[cfg.Name] {
			checkbox = okStyle.Render("[✓]")
		}

		// Get link status for this config
		linkStatus := p.state.LinkStatus[cfg.Name]
		drift := driftMap[cfg.Name]

		// Get enhanced status info
		statusInfo := p.getConfigStatusInfo(cfg, linkStatus, drift)

		// Calculate name width
		nameWidth := p.ContentWidth() - 10
		if nameWidth < 5 {
			nameWidth = 5
		}
		name := cfg.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		} else {
			name = fmt.Sprintf("%-*s", nameWidth, name)
		}

		content := fmt.Sprintf("%s%s %s %s",
			prefix,
			checkbox,
			name,
			statusInfo.icon,
		)

		content = fmt.Sprintf("%-*s", p.ContentWidth(), content)

		if idx == p.selectedIdx && p.focused {
			lines = append(lines, selectedStyle.Render(content))
		} else {
			lines = append(lines, normalStyle.Render(content))
		}
	}

	for len(lines) < visibleHeight {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// configStatusInfo holds detailed status information for a config
type configStatusInfo struct {
	icon       string
	statusText string
	statusTags []string
}

func (p *ConfigsPanel) getConfigStatusInfo(cfg config.ConfigItem, linkStatus *stow.ConfigLinkStatus, drift *stow.DriftResult) configStatusInfo {
	info := configStatusInfo{
		statusTags: []string{},
	}

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle

	if linkStatus != nil {
		conflictCount := 0
		for _, f := range linkStatus.Files {
			if !f.IsLinked && (strings.Contains(strings.ToLower(f.Issue), "conflict") ||
				strings.Contains(strings.ToLower(f.Issue), "exists") ||
				strings.Contains(strings.ToLower(f.Issue), "elsewhere")) {
				conflictCount++
			}
		}

		if conflictCount > 0 {
			info.icon = warnStyle.Render("⚠")
			info.statusTags = append(info.statusTags, fmt.Sprintf("conflicts (%d)", conflictCount))
		} else if linkStatus.IsFullyLinked() {
			info.icon = okStyle.Render("✓")
		} else if linkStatus.LinkedCount > 0 {
			info.icon = warnStyle.Render("◆")
		} else {
			info.icon = errStyle.Render("✗")
		}

		info.statusText = fmt.Sprintf("%d/%d", linkStatus.LinkedCount, linkStatus.TotalCount)
	} else {
		if drift != nil && drift.HasDrift {
			info.icon = warnStyle.Render("◆")
			info.statusText = fmt.Sprintf("%d new", len(drift.NewFiles))
		} else {
			info.icon = ui.SubtleStyle.Render("•")
			info.statusText = "unknown"
		}
	}

	if len(cfg.ExternalDeps) > 0 {
		missingExternal := false
		home := os.Getenv("HOME")
		for _, ext := range cfg.ExternalDeps {
			dest := ext.Destination
			if dest == "" {
				continue
			}
			fullDest := dest
			if !filepath.IsAbs(dest) {
				if home == "" {
					continue
				}
				fullDest = filepath.Join(home, dest)
			}
			if _, err := os.Stat(fullDest); os.IsNotExist(err) {
				missingExternal = true
				break
			}
		}
		if missingExternal {
			info.statusTags = append(info.statusTags, "external")
		}
	}

	if len(cfg.DependsOn) > 0 && p.state.LinkStatus != nil {
		missingDep := false
		for _, depName := range cfg.DependsOn {
			depStatus, ok := p.state.LinkStatus[depName]
			if !ok || !depStatus.IsFullyLinked() {
				missingDep = true
				break
			}
		}
		if missingDep {
			info.statusTags = append(info.statusTags, "deps")
		}
	}

	return info
}

// GetSelectedItem implements Panel interface
func (p *ConfigsPanel) GetSelectedItem() *SelectedItem {
	if len(p.state.Configs) == 0 || p.selectedIdx >= len(p.state.Configs) {
		return nil
	}
	cfg := p.state.Configs[p.selectedIdx]
	return &SelectedItem{
		ID:    cfg.Name,
		Name:  cfg.Name,
		Index: p.selectedIdx,
	}
}

// GetSelectedConfig returns the currently selected config
func (p *ConfigsPanel) GetSelectedConfig() *config.ConfigItem {
	if len(p.state.Configs) == 0 || p.selectedIdx >= len(p.state.Configs) {
		return nil
	}
	return &p.state.Configs[p.selectedIdx]
}

// GetSelectedIndex returns the selected index
func (p *ConfigsPanel) GetSelectedIndex() int {
	return p.selectedIdx
}

// SetSelectedIndex sets the selected index
func (p *ConfigsPanel) SetSelectedIndex(idx int) {
	if idx >= 0 && idx < len(p.state.Configs) {
		p.selectedIdx = idx
		p.ensureVisible()
	}
}

// ToggleSelection toggles selection state for current config
func (p *ConfigsPanel) ToggleSelection() {
	if len(p.state.Configs) == 0 || p.selectedIdx >= len(p.state.Configs) {
		return
	}
	cfgName := p.state.Configs[p.selectedIdx].Name
	if p.selected[cfgName] {
		delete(p.selected, cfgName)
	} else {
		p.selected[cfgName] = true
	}
}

// SelectAll selects all filtered configs
func (p *ConfigsPanel) SelectAll() {
	for _, idx := range p.filteredIdxs {
		p.selected[p.state.Configs[idx].Name] = true
	}
}

// DeselectAll deselects all configs
func (p *ConfigsPanel) DeselectAll() {
	for _, idx := range p.filteredIdxs {
		delete(p.selected, p.state.Configs[idx].Name)
	}
}

// GetSelected returns the selected config names
func (p *ConfigsPanel) GetSelected() map[string]bool {
	return p.selected
}

// GetSelectedNames returns a slice of selected config names
func (p *ConfigsPanel) GetSelectedNames() []string {
	names := make([]string, 0, len(p.selected))
	for name := range p.selected {
		names = append(names, name)
	}
	return names
}

// SetFilter applies a filter to the config list
func (p *ConfigsPanel) SetFilter(filterText string) {
	filtered := []int{}
	if filterText == "" {
		for i := range p.state.Configs {
			filtered = append(filtered, i)
		}
	} else {
		for i, cfg := range p.state.Configs {
			if strings.Contains(strings.ToLower(cfg.Name), strings.ToLower(filterText)) {
				filtered = append(filtered, i)
			}
		}
	}
	p.filteredIdxs = filtered
	p.listOffset = 0
	if len(filtered) > 0 {
		p.selectedIdx = filtered[0]
	}
}

// GetFilteredCount returns the number of filtered configs
func (p *ConfigsPanel) GetFilteredCount() int {
	return len(p.filteredIdxs)
}

// GetTotalCount returns the total number of configs
func (p *ConfigsPanel) GetTotalCount() int {
	return len(p.state.Configs)
}

// UpdateState updates the panel's state reference
func (p *ConfigsPanel) UpdateState(state State) {
	p.state = state
	// Rebuild filter indices
	p.filteredIdxs = make([]int, len(state.Configs))
	for i := range state.Configs {
		p.filteredIdxs[i] = i
	}
}
