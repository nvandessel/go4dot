package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/doctor"
	"github.com/nvandessel/go4dot/internal/ui"
)

// DetailsContext indicates what panel the details should show info for
type DetailsContext int

const (
	DetailsContextConfigs DetailsContext = iota
	DetailsContextHealth
	DetailsContextOverrides
	DetailsContextExternal
)

// DetailsPanel displays expanded info for the focused panel's selected item
// This is a scrollable panel when focused
type DetailsPanel struct {
	BasePanel
	state    State
	viewport viewport.Model
	ready    bool

	// Context determines what to display
	context DetailsContext

	// Data from various panels
	configsPanel   *ConfigsPanel
	healthPanel    *HealthPanel
	overridesPanel *OverridesPanel
	externalPanel  *ExternalPanel
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel(state State) *DetailsPanel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	return &DetailsPanel{
		BasePanel: NewBasePanel(PanelDetails, "Details"),
		state:     state,
		viewport:  vp,
		context:   DetailsContextConfigs,
	}
}

// Init implements Panel interface
func (p *DetailsPanel) Init() tea.Cmd {
	return nil
}

// SetSize implements Panel interface
func (p *DetailsPanel) SetSize(width, height int) {
	p.BasePanel.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()

	p.viewport.Width = contentWidth
	p.viewport.Height = contentHeight
	p.ready = true
	p.updateContent()
}

// Update implements Panel interface
func (p *DetailsPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.ready {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if p.focused {
			p.viewport, cmd = p.viewport.Update(msg)
			return cmd
		}
	case tea.KeyMsg:
		if p.focused {
			p.viewport, cmd = p.viewport.Update(msg)
			return cmd
		}
	}

	return nil
}

// View implements Panel interface
func (p *DetailsPanel) View() string {
	if !p.ready {
		return ""
	}
	return p.viewport.View()
}

// GetSelectedItem implements Panel interface - details doesn't have selection
func (p *DetailsPanel) GetSelectedItem() *SelectedItem {
	return nil
}

// SetContext sets what panel's content to display details for
func (p *DetailsPanel) SetContext(ctx DetailsContext) {
	p.context = ctx
	p.updateContent()
}

// SetPanels sets references to other panels for context-aware display
func (p *DetailsPanel) SetPanels(configs *ConfigsPanel, health *HealthPanel, overrides *OverridesPanel, external *ExternalPanel) {
	p.configsPanel = configs
	p.healthPanel = health
	p.overridesPanel = overrides
	p.externalPanel = external
}

// RefreshContent updates the content based on current context
func (p *DetailsPanel) RefreshContent() {
	p.updateContent()
}

func (p *DetailsPanel) updateContent() {
	var content string

	switch p.context {
	case DetailsContextHealth:
		content = p.renderHealthDetails()
	case DetailsContextOverrides:
		content = p.renderOverridesDetails()
	case DetailsContextExternal:
		content = p.renderExternalDetails()
	default:
		content = p.renderConfigDetails()
	}

	p.viewport.SetContent(content)
}

func (p *DetailsPanel) renderConfigDetails() string {
	if p.configsPanel == nil {
		return ui.SubtleStyle.Render("No config selected")
	}

	cfg := p.configsPanel.GetSelectedConfig()
	if cfg == nil {
		return ui.SubtleStyle.Render("No config selected")
	}

	linkStatus := p.state.LinkStatus[cfg.Name]

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle
	subtleStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Bold(true).
		Background(ui.PrimaryColor).
		Padding(0, 1)

	title := titleStyle.Render(strings.ToUpper(cfg.Name))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, title))
	lines = append(lines, "")

	if cfg.Description != "" {
		descStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Italic(true).Width(p.ContentWidth())
		lines = append(lines, descStyle.Render(cfg.Description))
		lines = append(lines, "")
	}

	if linkStatus != nil {
		lines = append(lines, headerStyle.Render("FILESYSTEM MAPPINGS"))

		for _, f := range linkStatus.Files {
			icon := okStyle.Render("✓")
			if !f.IsLinked {
				if strings.Contains(strings.ToLower(f.Issue), "conflict") ||
					strings.Contains(strings.ToLower(f.Issue), "exists") ||
					strings.Contains(strings.ToLower(f.Issue), "elsewhere") {
					icon = warnStyle.Render("⚠")
				} else {
					icon = errStyle.Render("✗")
				}
			}

			source := f.RelPath
			target := filepath.Join("~", f.RelPath)

			mapping := fmt.Sprintf("%s %s %s %s", icon, source, subtleStyle.Render("→"), target)
			lines = append(lines, mapping)

			if !f.IsLinked && f.Issue != "" {
				lines = append(lines, subtleStyle.Render("    └─ "+f.Issue))
			}
		}
		lines = append(lines, "")
	}

	if len(cfg.DependsOn) > 0 {
		lines = append(lines, headerStyle.Render("MODULE DEPENDENCIES"))
		for _, depName := range cfg.DependsOn {
			status := subtleStyle.Render("(unknown)")
			if p.state.LinkStatus != nil {
				if depStatus, ok := p.state.LinkStatus[depName]; ok {
					if depStatus.IsFullyLinked() {
						status = okStyle.Render("(✓ linked)")
					} else {
						status = warnStyle.Render("(✗ missing)")
					}
				}
			}
			lines = append(lines, fmt.Sprintf("• %s %s", depName, status))
		}
		lines = append(lines, "")
	}

	if len(cfg.ExternalDeps) > 0 {
		lines = append(lines, headerStyle.Render("EXTERNAL REPOSITORIES"))
		for _, extDep := range cfg.ExternalDeps {
			lines = append(lines, fmt.Sprintf("• %s", extDep.URL))
			lines = append(lines, subtleStyle.Render("  └─ "+extDep.Destination))
		}
		lines = append(lines, "")
	}

	if linkStatus != nil {
		statsLine := fmt.Sprintf("Total: %d files", linkStatus.TotalCount)
		statsStyle := lipgloss.NewStyle().
			Foreground(ui.SubtleColor).
			Align(lipgloss.Right).
			Width(p.ContentWidth())
		lines = append(lines, statsStyle.Render(statsLine))
	}

	return strings.Join(lines, "\n")
}

func (p *DetailsPanel) renderHealthDetails() string {
	if p.healthPanel == nil || p.healthPanel.IsLoading() {
		return ui.SubtleStyle.Render("Loading health checks...")
	}

	check := p.healthPanel.GetSelectedCheck()
	if check == nil {
		return ui.SubtleStyle.Render("No check selected")
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	skipStyle := ui.SubtleStyle
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	fixStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Italic(true)
	headerStyle := ui.HeaderStyle

	// Status icon
	var icon, statusText string
	switch check.Status {
	case doctor.StatusOK:
		icon = okStyle.Render("✓")
		statusText = okStyle.Render("Passed")
	case doctor.StatusWarning:
		icon = warnStyle.Render("⚠")
		statusText = warnStyle.Render("Warning")
	case doctor.StatusError:
		icon = errStyle.Render("✗")
		statusText = errStyle.Render("Error")
	case doctor.StatusSkipped:
		icon = skipStyle.Render("○")
		statusText = skipStyle.Render("Skipped")
	}

	lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(check.Name)))
	lines = append(lines, statusText)
	lines = append(lines, "")
	lines = append(lines, headerStyle.Render("DESCRIPTION"))
	lines = append(lines, descStyle.Render(check.Description))
	lines = append(lines, "")

	if check.Message != "" {
		lines = append(lines, headerStyle.Render("MESSAGE"))
		msgStyle := descStyle
		switch check.Status {
		case doctor.StatusError:
			msgStyle = errStyle
		case doctor.StatusWarning:
			msgStyle = warnStyle
		}
		lines = append(lines, msgStyle.Render(check.Message))
		lines = append(lines, "")
	}

	if check.Fix != "" {
		lines = append(lines, headerStyle.Render("FIX SUGGESTION"))
		lines = append(lines, fixStyle.Render(check.Fix))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (p *DetailsPanel) renderOverridesDetails() string {
	if p.overridesPanel == nil {
		return ui.SubtleStyle.Render("No machine config selected")
	}

	mc := p.overridesPanel.GetSelectedConfig()
	if mc == nil {
		return ui.SubtleStyle.Render("No machine config selected")
	}

	status := p.overridesPanel.GetMachineStatus()

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle

	// Status icon and text
	var icon, statusText string
	switch status {
	case "configured":
		icon = okStyle.Render("✓")
		statusText = okStyle.Render("Configured")
	case "missing":
		icon = warnStyle.Render("○")
		statusText = warnStyle.Render("Not configured")
	case "error":
		icon = errStyle.Render("✗")
		statusText = errStyle.Render("Error")
	default:
		icon = descStyle.Render("?")
		statusText = descStyle.Render("Unknown")
	}

	lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(mc.Description)))
	lines = append(lines, statusText)
	lines = append(lines, "")

	lines = append(lines, headerStyle.Render("DESTINATION"))
	lines = append(lines, descStyle.Render(mc.Destination))
	lines = append(lines, "")

	if len(mc.Prompts) > 0 {
		lines = append(lines, headerStyle.Render("FIELDS"))
		for _, prompt := range mc.Prompts {
			reqMark := ""
			if prompt.Required {
				reqMark = " *"
			}
			lines = append(lines, fmt.Sprintf("• %s%s", prompt.Prompt, reqMark))
			if prompt.Type != "" {
				lines = append(lines, descStyle.Render(fmt.Sprintf("  Type: %s", prompt.Type)))
			}
			if prompt.Default != "" {
				lines = append(lines, descStyle.Render(fmt.Sprintf("  Default: %s", prompt.Default)))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, descStyle.Render("Press Enter to configure"))

	return strings.Join(lines, "\n")
}

func (p *DetailsPanel) renderExternalDetails() string {
	if p.externalPanel == nil || p.externalPanel.IsLoading() {
		return ui.SubtleStyle.Render("Loading external dependencies...")
	}

	ext := p.externalPanel.GetSelectedExternal()
	if ext == nil {
		return ui.SubtleStyle.Render("No external dependency selected")
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	skipStyle := ui.SubtleStyle
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle

	name := ext.Dep.Name
	if name == "" {
		name = ext.Dep.ID
	}

	// Status icon and text
	var icon, statusText string
	switch ext.Status {
	case "installed":
		icon = okStyle.Render("✓")
		statusText = okStyle.Render("Installed")
	case "missing":
		icon = warnStyle.Render("○")
		statusText = warnStyle.Render("Not cloned")
	case "skipped":
		icon = skipStyle.Render("⊘")
		statusText = skipStyle.Render("Skipped (platform mismatch)")
	default:
		icon = skipStyle.Render("?")
		statusText = skipStyle.Render("Unknown")
	}

	lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(name)))
	lines = append(lines, statusText)
	lines = append(lines, "")

	lines = append(lines, headerStyle.Render("URL"))
	lines = append(lines, descStyle.Render(ext.Dep.URL))
	lines = append(lines, "")

	lines = append(lines, headerStyle.Render("DESTINATION"))
	lines = append(lines, descStyle.Render(ext.Dep.Destination))
	lines = append(lines, "")

	switch ext.Status {
	case "missing":
		lines = append(lines, descStyle.Render("Press Enter to clone"))
	case "installed":
		lines = append(lines, descStyle.Render("Press Enter to update"))
	}

	return strings.Join(lines, "\n")
}

// UpdateState updates the panel's state reference
func (p *DetailsPanel) UpdateState(state State) {
	p.state = state
	p.updateContent()
}
