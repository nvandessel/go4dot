package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui"
)

// externalStatusMsg is sent when external status is loaded
type externalStatusMsg struct {
	status []deps.ExternalStatus
	err    error
}

// ExternalPanel displays external dependencies list with status
// This is a navigable panel - Enter triggers clone/update
type ExternalPanel struct {
	BasePanel
	cfg          *config.Config
	dotfilesPath string
	platform     *platform.Platform

	status     []deps.ExternalStatus
	lastError  error
	spinner    spinner.Model
	loading    bool
	selectedIdx int
	listOffset  int
}

// NewExternalPanel creates a new external dependencies panel
func NewExternalPanel(cfg *config.Config, dotfilesPath string, plat *platform.Platform) *ExternalPanel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	return &ExternalPanel{
		BasePanel:    NewBasePanel(PanelExternal, "4 External"),
		cfg:          cfg,
		dotfilesPath: dotfilesPath,
		platform:     plat,
		spinner:      s,
		loading:      true,
	}
}

// Init implements Panel interface - starts loading status
func (p *ExternalPanel) Init() tea.Cmd {
	return tea.Batch(
		p.spinner.Tick,
		p.loadStatus,
	)
}

func (p *ExternalPanel) loadStatus() tea.Msg {
	if p.cfg == nil {
		return externalStatusMsg{status: nil, err: nil}
	}
	status := deps.CheckExternalStatus(p.cfg, p.platform, p.dotfilesPath)
	return externalStatusMsg{status: status}
}

// Update implements Panel interface
func (p *ExternalPanel) Update(msg tea.Msg) tea.Cmd {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !p.focused || p.loading {
			return nil
		}
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			p.moveDown()
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			p.moveUp()
		}

	case spinner.TickMsg:
		if p.loading {
			var cmd tea.Cmd
			p.spinner, cmd = p.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case externalStatusMsg:
		p.loading = false
		if msg.err != nil {
			p.lastError = msg.err
			p.status = nil
		} else {
			p.lastError = nil
			p.status = msg.status
			// Clamp selection if results shrunk
			if len(p.status) > 0 {
				if p.selectedIdx >= len(p.status) {
					p.selectedIdx = len(p.status) - 1
				}
				p.ensureVisible()
			} else {
				p.selectedIdx = 0
				p.listOffset = 0
			}
		}
	}

	return tea.Batch(cmds...)
}

func (p *ExternalPanel) moveDown() {
	maxIdx := len(p.status) - 1
	if p.selectedIdx < maxIdx {
		p.selectedIdx++
		p.ensureVisible()
	}
}

func (p *ExternalPanel) moveUp() {
	if p.selectedIdx > 0 {
		p.selectedIdx--
		p.ensureVisible()
	}
}

func (p *ExternalPanel) ensureVisible() {
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
func (p *ExternalPanel) View() string {
	if p.width < 5 || p.height < 3 {
		return ""
	}

	if p.loading {
		return p.spinner.View() + " Loading..."
	}

	if p.lastError != nil {
		return ui.ErrorStyle.Render("Error")
	}

	if len(p.status) == 0 {
		return ui.SubtleStyle.Render("No externals")
	}

	var lines []string

	// Summary counts
	installed, missing, skipped := 0, 0, 0
	for _, s := range p.status {
		switch s.Status {
		case "installed":
			installed++
		case "missing":
			missing++
		case "skipped":
			skipped++
		}
	}

	summaryParts := []string{}
	if missing > 0 {
		summaryParts = append(summaryParts, ui.WarningStyle.Render(fmt.Sprintf("%d○", missing)))
	}
	if installed > 0 {
		summaryParts = append(summaryParts, lipgloss.NewStyle().Foreground(ui.SecondaryColor).Render(fmt.Sprintf("%d✓", installed)))
	}
	if len(summaryParts) > 0 {
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Left, summaryParts...))
	}

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	skipStyle := ui.SubtleStyle

	visibleHeight := p.ContentHeight() - 1 // Account for summary
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	endIdx := p.listOffset + visibleHeight
	if endIdx > len(p.status) {
		endIdx = len(p.status)
	}

	for i := p.listOffset; i < endIdx; i++ {
		s := p.status[i]

		var icon string
		switch s.Status {
		case "installed":
			icon = okStyle.Render("✓")
		case "missing":
			icon = warnStyle.Render("○")
		case "skipped":
			icon = skipStyle.Render("⊘")
		default:
			icon = skipStyle.Render("?")
		}

		// Get name from dep
		name := s.Dep.Name
		if name == "" {
			name = s.Dep.ID
		}

		// Truncate name to fit
		maxLen := p.ContentWidth() - 4
		if maxLen < 5 {
			maxLen = 5
		}
		if len(name) > maxLen {
			name = name[:maxLen-1] + "…"
		}

		line := fmt.Sprintf("%s %s", icon, name)

		if i == p.selectedIdx && p.focused {
			line = ui.SelectedItemStyle.Width(p.ContentWidth()).Render(line)
		}

		lines = append(lines, line)
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// GetSelectedItem implements Panel interface
func (p *ExternalPanel) GetSelectedItem() *SelectedItem {
	if len(p.status) == 0 || p.selectedIdx >= len(p.status) {
		return nil
	}
	s := p.status[p.selectedIdx]
	name := s.Dep.Name
	if name == "" {
		name = s.Dep.ID
	}
	return &SelectedItem{
		ID:    s.Dep.ID,
		Name:  name,
		Index: p.selectedIdx,
	}
}

// GetSelectedExternal returns the currently selected external dep status
func (p *ExternalPanel) GetSelectedExternal() *deps.ExternalStatus {
	if len(p.status) == 0 || p.selectedIdx >= len(p.status) {
		return nil
	}
	return &p.status[p.selectedIdx]
}

// GetStatus returns all external dependency statuses
func (p *ExternalPanel) GetStatus() []deps.ExternalStatus {
	return p.status
}

// IsLoading returns whether the panel is still loading
func (p *ExternalPanel) IsLoading() bool {
	return p.loading
}

// Refresh reloads the external status while preserving the current selection
func (p *ExternalPanel) Refresh() tea.Cmd {
	p.loading = true
	// Don't reset selectedIdx or listOffset - preserve user's position
	return tea.Batch(
		p.spinner.Tick,
		p.loadStatus,
	)
}

// HasExternals returns true if there are any external dependencies
func (p *ExternalPanel) HasExternals() bool {
	return len(p.status) > 0
}
