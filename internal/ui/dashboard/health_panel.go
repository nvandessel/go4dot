package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/doctor"
	"github.com/nvandessel/go4dot/internal/ui"
)

// healthResultMsg is sent when doctor checks complete
type healthResultMsg struct {
	result *doctor.CheckResult
	err    error
}

// HealthPanel displays condensed doctor results (errors/warnings/ok counts)
// This is a navigable panel that shows individual checks
type HealthPanel struct {
	BasePanel
	cfg          *config.Config
	dotfilesPath string

	result      *doctor.CheckResult
	lastError   error
	spinner     spinner.Model
	loading     bool
	selectedIdx int
	listOffset  int
}

// NewHealthPanel creates a new health panel
func NewHealthPanel(cfg *config.Config, dotfilesPath string) *HealthPanel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	return &HealthPanel{
		BasePanel:    NewBasePanel(PanelHealth, "2 Health"),
		cfg:          cfg,
		dotfilesPath: dotfilesPath,
		spinner:      s,
		loading:      true,
	}
}

// Init implements Panel interface - starts health check
func (p *HealthPanel) Init() tea.Cmd {
	return tea.Batch(
		p.spinner.Tick,
		p.runChecks,
	)
}

func (p *HealthPanel) runChecks() tea.Msg {
	if p.cfg == nil {
		return healthResultMsg{result: nil, err: fmt.Errorf("no config")}
	}
	opts := doctor.CheckOptions{
		DotfilesPath: p.dotfilesPath,
	}
	result, err := doctor.RunChecks(p.cfg, opts)
	return healthResultMsg{result: result, err: err}
}

// Update implements Panel interface
func (p *HealthPanel) Update(msg tea.Msg) tea.Cmd {
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

	case healthResultMsg:
		p.loading = false
		if msg.err != nil {
			p.lastError = msg.err
			p.result = nil
		} else {
			p.lastError = nil
			p.result = msg.result
			// Clamp selection if results shrunk
			if p.result != nil && len(p.result.Checks) > 0 {
				if p.selectedIdx >= len(p.result.Checks) {
					p.selectedIdx = len(p.result.Checks) - 1
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

func (p *HealthPanel) moveDown() {
	if p.result == nil {
		return
	}
	maxIdx := len(p.result.Checks) - 1
	if p.selectedIdx < maxIdx {
		p.selectedIdx++
		p.ensureVisible()
	}
}

func (p *HealthPanel) moveUp() {
	if p.selectedIdx > 0 {
		p.selectedIdx--
		p.ensureVisible()
	}
}

func (p *HealthPanel) ensureVisible() {
	visibleHeight := p.ContentHeight() - 2 // Account for summary line
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
func (p *HealthPanel) View() string {
	if p.width < 5 || p.height < 3 {
		return ""
	}

	if p.loading {
		return p.spinner.View() + " Checking..."
	}

	if p.lastError != nil {
		return ui.ErrorStyle.Render("Error: " + p.lastError.Error())
	}

	if p.result == nil {
		return ui.SubtleStyle.Render("No results")
	}

	var lines []string

	// Summary counts
	ok, warnings, errors, _ := p.result.CountByStatus()
	summaryParts := []string{}
	if errors > 0 {
		summaryParts = append(summaryParts, ui.ErrorStyle.Render(fmt.Sprintf("%d✗", errors)))
	}
	if warnings > 0 {
		summaryParts = append(summaryParts, ui.WarningStyle.Render(fmt.Sprintf("%d⚠", warnings)))
	}
	if ok > 0 {
		summaryParts = append(summaryParts, lipgloss.NewStyle().Foreground(ui.SecondaryColor).Render(fmt.Sprintf("%d✓", ok)))
	}
	lines = append(lines, strings.Join(summaryParts, " "))

	// Check list (compact)
	visibleHeight := p.ContentHeight() - 2
	if visibleHeight < 1 {
		visibleHeight = 1
	}

	endIdx := p.listOffset + visibleHeight
	if endIdx > len(p.result.Checks) {
		endIdx = len(p.result.Checks)
	}

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	skipStyle := ui.SubtleStyle

	for i := p.listOffset; i < endIdx; i++ {
		check := p.result.Checks[i]

		var icon string
		switch check.Status {
		case doctor.StatusOK:
			icon = okStyle.Render("✓")
		case doctor.StatusWarning:
			icon = warnStyle.Render("⚠")
		case doctor.StatusError:
			icon = errStyle.Render("✗")
		case doctor.StatusSkipped:
			icon = skipStyle.Render("○")
		}

		// Truncate name to fit
		name := check.Name
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
func (p *HealthPanel) GetSelectedItem() *SelectedItem {
	if p.result == nil || p.selectedIdx >= len(p.result.Checks) {
		return nil
	}
	check := p.result.Checks[p.selectedIdx]
	return &SelectedItem{
		ID:    check.Name,
		Name:  check.Name,
		Index: p.selectedIdx,
	}
}

// GetSelectedCheck returns the currently selected check for details display
func (p *HealthPanel) GetSelectedCheck() *doctor.Check {
	if p.result == nil || p.selectedIdx >= len(p.result.Checks) {
		return nil
	}
	return &p.result.Checks[p.selectedIdx]
}

// GetResult returns the full health check result
func (p *HealthPanel) GetResult() *doctor.CheckResult {
	return p.result
}

// IsLoading returns whether the panel is still loading
func (p *HealthPanel) IsLoading() bool {
	return p.loading
}

// Refresh re-runs the health checks while preserving the current selection
func (p *HealthPanel) Refresh() tea.Cmd {
	p.loading = true
	// Don't reset selectedIdx or listOffset - preserve user's position
	return tea.Batch(
		p.spinner.Tick,
		p.runChecks,
	)
}
