package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/ui"
)

// OverridesPanel displays machine configuration list with status icons
// This is a navigable panel - Enter opens edit form (modal)
type OverridesPanel struct {
	BasePanel
	cfg           *config.Config
	machineStatus []machine.MachineConfigStatus

	selectedIdx int
	listOffset  int
}

// NewOverridesPanel creates a new overrides panel
func NewOverridesPanel(cfg *config.Config) *OverridesPanel {
	var status []machine.MachineConfigStatus
	if cfg != nil {
		status = machine.CheckMachineConfigStatus(cfg)
	}

	return &OverridesPanel{
		BasePanel:     NewBasePanel(PanelOverrides, "Overrides"),
		cfg:           cfg,
		machineStatus: status,
	}
}

// Init implements Panel interface
func (p *OverridesPanel) Init() tea.Cmd {
	return nil
}

// Update implements Panel interface
func (p *OverridesPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.focused {
		return nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			p.moveDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			p.moveUp()
		}
	}

	return nil
}

func (p *OverridesPanel) moveDown() {
	if p.cfg == nil {
		return
	}
	maxIdx := len(p.cfg.MachineConfig) - 1
	if p.selectedIdx < maxIdx {
		p.selectedIdx++
		p.ensureVisible()
	}
}

func (p *OverridesPanel) moveUp() {
	if p.selectedIdx > 0 {
		p.selectedIdx--
		p.ensureVisible()
	}
}

func (p *OverridesPanel) ensureVisible() {
	visibleHeight := p.ContentHeight()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	if p.selectedIdx < p.listOffset {
		p.listOffset = p.selectedIdx
	} else if p.selectedIdx >= p.listOffset+visibleHeight {
		p.listOffset = p.selectedIdx - visibleHeight + 1
	}
}

// View implements Panel interface
func (p *OverridesPanel) View() string {
	if p.width < 5 || p.height < 3 {
		return ""
	}

	if p.cfg == nil || len(p.cfg.MachineConfig) == 0 {
		return ui.SubtleStyle.Render("No overrides")
	}

	var lines []string

	// Build status map
	statusMap := make(map[string]string)
	for _, s := range p.machineStatus {
		statusMap[s.ID] = s.Status
	}

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)

	visibleHeight := p.ContentHeight()
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	endIdx := p.listOffset + visibleHeight
	if endIdx > len(p.cfg.MachineConfig) {
		endIdx = len(p.cfg.MachineConfig)
	}

	for i := p.listOffset; i < endIdx; i++ {
		mc := p.cfg.MachineConfig[i]

		// Status icon
		var icon string
		status := statusMap[mc.ID]
		switch status {
		case "configured":
			icon = okStyle.Render("✓")
		case "missing":
			icon = warnStyle.Render("○")
		case "error":
			icon = errStyle.Render("✗")
		default:
			icon = ui.SubtleStyle.Render("?")
		}

		// Truncate description to fit
		desc := mc.Description
		maxLen := p.ContentWidth() - 4
		if maxLen < 5 {
			maxLen = 5
		}
		if len(desc) > maxLen {
			desc = desc[:maxLen-1] + "…"
		}

		line := fmt.Sprintf("%s %s", icon, desc)

		if i == p.selectedIdx && p.focused {
			line = ui.SelectedItemStyle.Width(p.ContentWidth()).Render(line)
		}

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// GetSelectedItem implements Panel interface
func (p *OverridesPanel) GetSelectedItem() *SelectedItem {
	if p.cfg == nil || p.selectedIdx >= len(p.cfg.MachineConfig) {
		return nil
	}
	mc := p.cfg.MachineConfig[p.selectedIdx]
	return &SelectedItem{
		ID:    mc.ID,
		Name:  mc.Description,
		Index: p.selectedIdx,
	}
}

// GetSelectedConfig returns the currently selected machine config
func (p *OverridesPanel) GetSelectedConfig() *config.MachinePrompt {
	if p.cfg == nil || p.selectedIdx >= len(p.cfg.MachineConfig) {
		return nil
	}
	return &p.cfg.MachineConfig[p.selectedIdx]
}

// GetMachineStatus returns the status for the currently selected config
func (p *OverridesPanel) GetMachineStatus() string {
	if p.cfg == nil || p.selectedIdx >= len(p.cfg.MachineConfig) {
		return ""
	}
	mc := p.cfg.MachineConfig[p.selectedIdx]
	for _, s := range p.machineStatus {
		if s.ID == mc.ID {
			return s.Status
		}
	}
	return ""
}

// RefreshStatus updates the machine config status
func (p *OverridesPanel) RefreshStatus() {
	if p.cfg != nil {
		p.machineStatus = machine.CheckMachineConfigStatus(p.cfg)
	}
}

// HasOverrides returns true if there are any machine configs
func (p *OverridesPanel) HasOverrides() bool {
	return p.cfg != nil && len(p.cfg.MachineConfig) > 0
}
