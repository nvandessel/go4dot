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

// ASCII-safe status icons for consistent rendering across terminals
const (
	iconOK      = "[ok]"
	iconWarning = "[!]"
	iconError   = "[x]"
	iconSkipped = "[-]"
)

// Lines reserved for non-list content in the panel:
// 1 line for the summary, 1 line for the blank separator
const healthSummaryLines = 2

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

// getListVisibleHeight returns the number of check items that can be displayed
// in the panel's content area after accounting for the summary line and separator.
func (p *HealthPanel) getListVisibleHeight() int {
	h := p.ContentHeight() - healthSummaryLines
	if h < 1 {
		return 1
	}
	return h
}

// ensureVisible adjusts listOffset so the selected item is within the visible
// area. It accounts for scroll indicator lines that renderCheckItems reserves
// when the list overflows above or below the viewport.
func (p *HealthPanel) ensureVisible() {
	totalChecks := len(p.result.Checks)
	visibleHeight := p.getListVisibleHeight()

	// First pass: coarse adjustment using the raw visible height
	if p.selectedIdx < p.listOffset {
		p.listOffset = p.selectedIdx
	} else if p.selectedIdx >= p.listOffset+visibleHeight {
		p.listOffset = p.selectedIdx - visibleHeight + 1
	}

	// Second pass: account for scroll indicator lines.
	// renderCheckItems reserves one line each for the scroll-up and scroll-down
	// indicators, reducing the number of item slots available.
	effectiveSlots := visibleHeight
	if p.listOffset > 0 {
		effectiveSlots-- // scroll-up indicator takes a line
	}
	if p.listOffset+effectiveSlots < totalChecks {
		effectiveSlots-- // scroll-down indicator takes a line
	}
	if effectiveSlots < 1 {
		effectiveSlots = 1
	}

	// Re-adjust offset so selectedIdx fits within the effective item slots
	if p.selectedIdx < p.listOffset {
		p.listOffset = p.selectedIdx
	} else if p.selectedIdx >= p.listOffset+effectiveSlots {
		p.listOffset = p.selectedIdx - effectiveSlots + 1
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

	// Summary line with proper spacing
	lines = append(lines, p.renderSummary())

	// Blank separator between summary and check list
	lines = append(lines, "")

	// Check items with scroll indicators
	lines = append(lines, p.renderCheckItems()...)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderSummary builds the summary counts line with proper spacing between items.
func (p *HealthPanel) renderSummary() string {
	ok, warnings, errors, _ := p.result.CountByStatus()
	var parts []string

	if errors > 0 {
		parts = append(parts, ui.ErrorStyle.Render(fmt.Sprintf("%d err", errors)))
	}
	if warnings > 0 {
		parts = append(parts, ui.WarningStyle.Render(fmt.Sprintf("%d warn", warnings)))
	}
	if ok > 0 {
		okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
		parts = append(parts, okStyle.Render(fmt.Sprintf("%d ok", ok)))
	}

	if len(parts) == 0 {
		return ui.SubtleStyle.Render("No checks")
	}

	return strings.Join(parts, "  ")
}

// renderCheckItems builds the visible slice of check items, including scroll
// indicators when the list overflows above or below the visible area.
func (p *HealthPanel) renderCheckItems() []string {
	totalChecks := len(p.result.Checks)
	visibleHeight := p.getListVisibleHeight()

	// Determine if we need scroll indicators and adjust available height
	hasScrollUp := p.listOffset > 0
	hasScrollDown := p.listOffset+visibleHeight < totalChecks

	// Reserve lines for scroll indicators
	itemSlots := visibleHeight
	if hasScrollUp {
		itemSlots--
	}
	if hasScrollDown {
		itemSlots--
	}
	if itemSlots < 1 {
		itemSlots = 1
	}

	// Recalculate end index with adjusted item slots
	endIdx := p.listOffset + itemSlots
	if endIdx > totalChecks {
		endIdx = totalChecks
	}

	var lines []string

	// Scroll-up indicator
	if hasScrollUp {
		above := p.listOffset
		lines = append(lines, ui.SubtleStyle.Render(fmt.Sprintf("^^ %d more", above)))
	}

	// Render visible check items
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	skipStyle := ui.SubtleStyle

	for i := p.listOffset; i < endIdx; i++ {
		check := p.result.Checks[i]

		var icon string
		switch check.Status {
		case doctor.StatusOK:
			icon = okStyle.Render(iconOK)
		case doctor.StatusWarning:
			icon = warnStyle.Render(iconWarning)
		case doctor.StatusError:
			icon = errStyle.Render(iconError)
		case doctor.StatusSkipped:
			icon = skipStyle.Render(iconSkipped)
		}

		// Truncate name to fit (icon + space + name)
		name := check.Name
		maxLen := p.ContentWidth() - 6 // icon width (4) + space (1) + margin (1)
		if maxLen < 5 {
			maxLen = 5
		}
		if len(name) > maxLen {
			name = name[:maxLen-3] + "..."
		}

		line := fmt.Sprintf("%s %s", icon, name)

		if i == p.selectedIdx && p.focused {
			line = ui.SelectedItemStyle.Width(p.ContentWidth()).Render(line)
		}

		lines = append(lines, line)
	}

	// Scroll-down indicator
	if hasScrollDown {
		below := totalChecks - endIdx
		lines = append(lines, ui.SubtleStyle.Render(fmt.Sprintf("vv %d more", below)))
	}

	return lines
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
