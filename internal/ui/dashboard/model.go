package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Action represents a user action from the dashboard
type Action int

const (
	ActionNone Action = iota
	ActionSync
	ActionSyncConfig
	ActionDoctor
	ActionInstall
	ActionMachineConfig
	ActionExternal
	ActionUninstall
	ActionUpdate
	ActionList
	ActionInit
	ActionQuit
	ActionRefresh
)

// Result is returned when the dashboard exits
type Result struct {
	Action     Action
	ConfigName string // For ActionSyncConfig
}

// MachineStatus represents the status of a machine config for the dashboard
type MachineStatus struct {
	ID          string
	Description string
	Status      string // "configured", "missing", "error"
}

// Model is the Bubbletea model for the dashboard
type Model struct {
	width         int
	height        int
	platform      *platform.Platform
	driftSummary  *stow.DriftSummary
	linkStatus    map[string]*stow.ConfigLinkStatus // Per-config link status
	machineStatus []MachineStatus
	configs       []config.ConfigItem
	dotfilesPath  string
	updateMsg     string
	selectedIdx   int
	expandedIdx   int // Index of expanded config (-1 = none)
	scrollOffset  int // Scroll offset within expanded file list
	result        *Result
	quitting      bool
	hasBaseline   bool // True if we have stored symlink counts (synced before)

	// Enhanced features
	searchMode    bool   // True when in search mode
	searchQuery   string // Current search filter
	filteredIdxs  []int  // Indices of configs matching search
	multiSelect   bool   // True when in multi-select mode
	selected      map[int]bool // Set of selected config indices
	showHelp      bool   // Show help overlay
	showDetails   bool   // Show details panel for selected config
	themeName     string // Current theme name
}

// keyMap defines the key bindings
type keyMap struct {
	Sync       key.Binding
	Doctor     key.Binding
	Install    key.Binding
	Machine    key.Binding
	Update     key.Binding
	Menu       key.Binding
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Expand     key.Binding
	Help       key.Binding
	Search     key.Binding
	MultiSelect key.Binding
	SelectItem key.Binding
	BulkStow   key.Binding
	BulkUnstow key.Binding
	Details    key.Binding
	Refresh    key.Binding
	Theme      key.Binding
}

var keys = keyMap{
	Sync: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sync all"),
	),
	Doctor: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "doctor"),
	),
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Machine: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "overrides"),
	),
	Update: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update"),
	),
	Menu: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "more"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc"),
		key.WithHelp("q/esc", "quit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "sync config"),
	),
	Expand: key.NewBinding(
		key.WithKeys("e", "right"),
		key.WithHelp("e/→", "expand"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	MultiSelect: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "multi-select"),
	),
	SelectItem: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	BulkStow: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "bulk stow"),
	),
	BulkUnstow: key.NewBinding(
		key.WithKeys("U"),
		key.WithHelp("U", "bulk unstow"),
	),
	Details: key.NewBinding(
		key.WithKeys("D"),
		key.WithHelp("D", "details"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r", "ctrl+r"),
		key.WithHelp("r", "refresh"),
	),
	Theme: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "theme"),
	),
}

// New creates a new dashboard model
func New(p *platform.Platform, driftSummary *stow.DriftSummary, linkStatus map[string]*stow.ConfigLinkStatus, machineStatus []MachineStatus, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool) Model {
	// Initialize filtered indices to all configs
	filteredIdxs := make([]int, len(configs))
	for i := range configs {
		filteredIdxs[i] = i
	}

	return Model{
		platform:      p,
		driftSummary:  driftSummary,
		linkStatus:    linkStatus,
		machineStatus: machineStatus,
		configs:       configs,
		dotfilesPath:  dotfilesPath,
		updateMsg:     updateMsg,
		selectedIdx:   0,
		expandedIdx:   -1,
		hasBaseline:   hasBaseline,
		searchMode:    false,
		searchQuery:   "",
		filteredIdxs:  filteredIdxs,
		multiSelect:   false,
		selected:      make(map[int]bool),
		showHelp:      false,
		showDetails:   false,
		themeName:     "default",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle search mode input
		if m.searchMode {
			switch msg.String() {
			case "esc", "ctrl+c":
				m.searchMode = false
				m.searchQuery = ""
				m.updateFilter()
			case "enter":
				m.searchMode = false
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
					m.updateFilter()
				}
			default:
				if len(msg.String()) == 1 {
					m.searchQuery += msg.String()
					m.updateFilter()
				}
			}
			return m, nil
		}

		// Handle help screen
		if m.showHelp {
			switch msg.String() {
			case "?", "esc", "q":
				m.showHelp = false
			}
			return m, nil
		}

		// Handle details panel
		if m.showDetails {
			switch msg.String() {
			case "D", "esc", "q":
				m.showDetails = false
			}
			return m, nil
		}

		// Normal navigation and commands
		switch {
		case key.Matches(msg, keys.Quit):
			m.result = &Result{Action: ActionQuit}
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Sync):
			m.result = &Result{Action: ActionSync}
			return m, tea.Quit

		case key.Matches(msg, keys.Doctor):
			m.result = &Result{Action: ActionDoctor}
			return m, tea.Quit

		case key.Matches(msg, keys.Install):
			m.result = &Result{Action: ActionInstall}
			return m, tea.Quit

		case key.Matches(msg, keys.Machine):
			m.result = &Result{Action: ActionMachineConfig}
			return m, tea.Quit

		case key.Matches(msg, keys.Update):
			m.result = &Result{Action: ActionUpdate}
			return m, tea.Quit

		case key.Matches(msg, keys.Menu):
			m.result = &Result{Action: ActionList}
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if !m.searchMode && !m.showHelp {
				m.navigateUp()
			}

		case key.Matches(msg, keys.Down):
			if !m.searchMode && !m.showHelp {
				m.navigateDown()
			}

		case key.Matches(msg, keys.Expand):
			if !m.searchMode && !m.showHelp && !m.showDetails {
				// Toggle expansion for selected config
				actualIdx := m.getActualSelectedIdx()
				if m.expandedIdx == actualIdx {
					m.expandedIdx = -1
					m.scrollOffset = 0
				} else {
					m.expandedIdx = actualIdx
					m.scrollOffset = 0
				}
			}

		case key.Matches(msg, keys.Enter):
			if !m.searchMode && !m.showHelp && !m.showDetails {
				if m.multiSelect && len(m.selected) > 0 {
					// In multi-select, sync all selected configs
					m.result = &Result{Action: ActionSync}
					return m, tea.Quit
				} else if len(m.configs) > 0 && m.selectedIdx < len(m.filteredIdxs) {
					// Sync single selected config
					actualIdx := m.filteredIdxs[m.selectedIdx]
					m.result = &Result{
						Action:     ActionSyncConfig,
						ConfigName: m.configs[actualIdx].Name,
					}
					return m, tea.Quit
				}
			}

		case key.Matches(msg, keys.Search):
			if !m.searchMode {
				m.searchMode = true
				m.searchQuery = ""
				m.updateFilter()
			}

		case key.Matches(msg, keys.MultiSelect):
			m.multiSelect = !m.multiSelect
			if !m.multiSelect {
				m.selected = make(map[int]bool)
			}

		case key.Matches(msg, keys.SelectItem):
			if m.multiSelect && len(m.configs) > 0 && m.selectedIdx < len(m.configs) {
				actualIdx := m.getActualIndex(m.selectedIdx)
				if m.selected[actualIdx] {
					delete(m.selected, actualIdx)
				} else {
					m.selected[actualIdx] = true
				}
			}

		case key.Matches(msg, keys.BulkStow):
			if m.multiSelect && len(m.selected) > 0 {
				// TODO: Implement bulk stow
				m.result = &Result{Action: ActionSync}
				return m, tea.Quit
			}

		case key.Matches(msg, keys.BulkUnstow):
			if m.multiSelect && len(m.selected) > 0 {
				// TODO: Implement bulk unstow
				m.result = &Result{Action: ActionUninstall}
				return m, tea.Quit
			}

		case key.Matches(msg, keys.Details):
			m.showDetails = !m.showDetails

		case key.Matches(msg, keys.Refresh):
			// Refresh action - reload config and state
			m.result = &Result{Action: ActionRefresh}
			return m, nil

		case key.Matches(msg, keys.Theme):
			// Cycle through themes
			m.cycleTheme()

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// Helper methods for navigation and filtering

func (m *Model) navigateUp() {
	if m.selectedIdx > 0 {
		m.selectedIdx--
		m.expandedIdx = -1
		m.scrollOffset = 0
	}
}

func (m *Model) navigateDown() {
	maxIdx := len(m.filteredIdxs) - 1
	if m.selectedIdx < maxIdx {
		m.selectedIdx++
		m.expandedIdx = -1
		m.scrollOffset = 0
	}
}

func (m *Model) getActualSelectedIdx() int {
	if m.selectedIdx >= 0 && m.selectedIdx < len(m.filteredIdxs) {
		return m.filteredIdxs[m.selectedIdx]
	}
	return 0
}

func (m *Model) getActualIndex(displayIdx int) int {
	if displayIdx >= 0 && displayIdx < len(m.filteredIdxs) {
		return m.filteredIdxs[displayIdx]
	}
	return 0
}

func (m *Model) updateFilter() {
	if m.searchQuery == "" {
		// No filter - show all configs
		m.filteredIdxs = make([]int, len(m.configs))
		for i := range m.configs {
			m.filteredIdxs[i] = i
		}
	} else {
		// Filter configs by name
		m.filteredIdxs = []int{}
		query := strings.ToLower(m.searchQuery)
		for i, cfg := range m.configs {
			if strings.Contains(strings.ToLower(cfg.Name), query) {
				m.filteredIdxs = append(m.filteredIdxs, i)
			}
		}
	}

	// Reset selection if out of bounds
	if m.selectedIdx >= len(m.filteredIdxs) {
		m.selectedIdx = 0
	}
}

func (m *Model) cycleTheme() {
	themes := []string{"default", "dark", "light", "ocean", "forest"}
	currentIdx := 0
	for i, t := range themes {
		if t == m.themeName {
			currentIdx = i
			break
		}
	}
	nextIdx := (currentIdx + 1) % len(themes)
	m.themeName = themes[nextIdx]
	m.applyTheme()
}

func (m *Model) applyTheme() {
	switch m.themeName {
	case "dark":
		ui.PrimaryColor = lipgloss.Color("#8B5CF6")
		ui.SecondaryColor = lipgloss.Color("#10B981")
		ui.WarningColor = lipgloss.Color("#F59E0B")
		ui.ErrorColor = lipgloss.Color("#EF4444")
		ui.SubtleColor = lipgloss.Color("#6B7280")
	case "light":
		ui.PrimaryColor = lipgloss.Color("#7C3AED")
		ui.SecondaryColor = lipgloss.Color("#059669")
		ui.WarningColor = lipgloss.Color("#D97706")
		ui.ErrorColor = lipgloss.Color("#DC2626")
		ui.SubtleColor = lipgloss.Color("#9CA3AF")
	case "ocean":
		ui.PrimaryColor = lipgloss.Color("#0EA5E9")
		ui.SecondaryColor = lipgloss.Color("#06B6D4")
		ui.WarningColor = lipgloss.Color("#F59E0B")
		ui.ErrorColor = lipgloss.Color("#EF4444")
		ui.SubtleColor = lipgloss.Color("#64748B")
	case "forest":
		ui.PrimaryColor = lipgloss.Color("#22C55E")
		ui.SecondaryColor = lipgloss.Color("#84CC16")
		ui.WarningColor = lipgloss.Color("#EAB308")
		ui.ErrorColor = lipgloss.Color("#F87171")
		ui.SubtleColor = lipgloss.Color("#71717A")
	default: // "default"
		ui.PrimaryColor = lipgloss.Color("#7D56F4")
		ui.SecondaryColor = lipgloss.Color("#04B575")
		ui.WarningColor = lipgloss.Color("#FFCC00")
		ui.ErrorColor = lipgloss.Color("#FF0000")
		ui.SubtleColor = lipgloss.Color("#626262")
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	// Show help overlay if requested
	if m.showHelp {
		return m.renderHelpScreen()
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// Search mode indicator
	if m.searchMode {
		searchPrompt := m.renderSearchPrompt()
		b.WriteString(searchPrompt)
		b.WriteString("\n\n")
	}

	// Multi-select mode indicator
	if m.multiSelect {
		multiSelectInfo := m.renderMultiSelectInfo()
		b.WriteString(multiSelectInfo)
		b.WriteString("\n\n")
	}

	// Status summary
	status := m.renderStatus()
	b.WriteString(status)
	b.WriteString("\n\n")

	// Machine status (if any)
	if len(m.machineStatus) > 0 {
		machineStatus := m.renderMachineStatus()
		b.WriteString(machineStatus)
		b.WriteString("\n\n")
	}

	// Show details panel if requested
	if m.showDetails && m.selectedIdx < len(m.filteredIdxs) {
		details := m.renderDetailsPanel()
		b.WriteString(details)
		b.WriteString("\n\n")
	}

	// Config list
	configList := m.renderConfigList()
	b.WriteString(configList)
	b.WriteString("\n\n")

	// Action bar
	actions := m.renderActions()
	b.WriteString(actions)

	return b.String()
}

func (m Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	title := titleStyle.Render("go4dot Dashboard")

	platformInfo := ""
	if m.platform != nil {
		platformInfo = fmt.Sprintf(" %s (%s)", m.platform.OS, m.platform.PackageManager)
	}

	subtitle := subtitleStyle.Render(platformInfo)

	updateInfo := ""
	if m.updateMsg != "" {
		updateInfo = subtitleStyle.Render(" " + m.updateMsg)
	}

	return title + subtitle + updateInfo
}

func (m Model) renderStatus() string {
	// If we haven't synced before (no baseline), show that clearly
	if !m.hasBaseline {
		return lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render("  Not synced yet - press [s] to create symlinks")
	}

	// We have baseline - check for drift
	if m.driftSummary != nil && m.driftSummary.HasDrift() {
		return lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render(fmt.Sprintf("  %d config(s) need syncing", m.driftSummary.DriftedConfigs))
	}

	// Baseline exists and no drift
	return lipgloss.NewStyle().
		Foreground(ui.SecondaryColor).
		Bold(true).
		Render("  All synced")
}

func (m Model) renderMachineStatus() string {
	var parts []string

	headerStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor).Bold(true)
	parts = append(parts, headerStyle.Render("  Overrides:"))

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	missStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)

	for _, status := range m.machineStatus {
		icon := ""
		switch status.Status {
		case "configured":
			icon = okStyle.Render("+")
		case "missing":
			icon = missStyle.Render("x")
		case "error":
			icon = errStyle.Render("!")
		}

		parts = append(parts, fmt.Sprintf("%s %s", icon, status.ID))
	}

	return strings.Join(parts, "  ")
}

func (m Model) renderConfigList() string {
	var lines []string

	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle
	subtleStyle := ui.SubtleStyle

	// Build a map of drift results for quick lookup
	driftMap := make(map[string]*stow.DriftResult)
	if m.driftSummary != nil {
		for i := range m.driftSummary.Results {
			r := &m.driftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}

	// Show filtered results
	if len(m.filteredIdxs) == 0 {
		noResultsStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor).Italic(true)
		return noResultsStyle.Render("  No configs match your search")
	}

	for displayIdx, actualIdx := range m.filteredIdxs {
		cfg := m.configs[actualIdx]
		var line string
		prefix := "  "

		// Show selection checkbox in multi-select mode
		if m.multiSelect {
			if m.selected[actualIdx] {
				prefix = "[✓] "
			} else {
				prefix = "[ ] "
			}
		}

		if displayIdx == m.selectedIdx {
			if !m.multiSelect {
				prefix = "> "
			} else if m.selected[actualIdx] {
				prefix = ">[✓] "
			} else {
				prefix = ">[ ] "
			}
		}

		nameStyle := normalStyle
		if displayIdx == m.selectedIdx {
			nameStyle = selectedStyle
		}

		// Get link status for this config
		linkStatus := m.linkStatus[cfg.Name]
		drift := driftMap[cfg.Name]

		// Determine status icon and text based on link status
		var statusIcon, statusText string

		if linkStatus != nil {
			linked := linkStatus.LinkedCount
			total := linkStatus.TotalCount

			if linkStatus.IsFullyLinked() {
				// Fully linked
				statusIcon = okStyle.Render("✓")
				statusText = fmt.Sprintf("%d/%d", linked, total)
			} else if linkStatus.IsPartiallyLinked() {
				// Partially linked
				statusIcon = warnStyle.Render("⚠")
				newFiles := 0
				if drift != nil {
					newFiles = len(drift.NewFiles)
				}
				if newFiles > 0 {
					statusText = fmt.Sprintf("%d/%d (%d new)", linked, total, newFiles)
				} else {
					statusText = fmt.Sprintf("%d/%d", linked, total)
				}
			} else {
				// Not linked
				statusIcon = errStyle.Render("✗")
				statusText = fmt.Sprintf("0/%d", total)
			}
		} else if drift != nil {
			// Fallback to drift-based display if no link status
			if drift.HasDrift {
				statusIcon = warnStyle.Render("◆")
				statusText = fmt.Sprintf("%d new", len(drift.NewFiles))
			} else {
				statusIcon = okStyle.Render("●")
				statusText = fmt.Sprintf("%d files", drift.CurrentCount)
			}
		} else {
			statusIcon = subtleStyle.Render("○")
			statusText = "unknown"
		}

		// Pad name to align status
		maxNameLen := 18
		nameLen := len(cfg.Name)
		if nameLen > maxNameLen {
			nameLen = maxNameLen
		}
		dots := subtleStyle.Render(strings.Repeat(".", maxNameLen-nameLen+2))

		line = fmt.Sprintf("%s%s %s %s %s",
			prefix,
			nameStyle.Render(cfg.Name),
			dots,
			statusIcon,
			subtleStyle.Render(statusText),
		)

		lines = append(lines, line)

		// Show expanded file list if this config is expanded
		if displayIdx == m.selectedIdx && m.expandedIdx == actualIdx && linkStatus != nil && len(linkStatus.Files) > 0 {
			maxFiles := 8
			files := linkStatus.Files
			start := m.scrollOffset
			if start >= len(files) {
				start = 0
			}
			end := start + maxFiles
			if end > len(files) {
				end = len(files)
			}

			for j := start; j < end; j++ {
				f := files[j]
				var fileIcon string
				if f.IsLinked {
					fileIcon = okStyle.Render("✓")
				} else {
					fileIcon = errStyle.Render("✗")
				}
				fileLine := fmt.Sprintf("      %s %s", fileIcon, subtleStyle.Render(f.RelPath))
				if !f.IsLinked && f.Issue != "" {
					fileLine += subtleStyle.Render(" (" + f.Issue + ")")
				}
				lines = append(lines, fileLine)
			}

			// Show scroll indicator if needed
			if len(files) > maxFiles {
				remaining := len(files) - end
				if remaining > 0 {
					lines = append(lines, subtleStyle.Render(fmt.Sprintf("      ... %d more", remaining)))
				}
			}
		} else if displayIdx == m.selectedIdx && linkStatus != nil {
			// Show summary hint when selected but not expanded
			if !linkStatus.IsFullyLinked() && len(linkStatus.GetMissingFiles()) > 0 {
				missing := linkStatus.GetMissingFiles()
				hint := missing[0].RelPath
				if len(missing) > 1 {
					hint += fmt.Sprintf(" (+%d more)", len(missing)-1)
				}
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("      press [e] to expand • %s", hint)))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderActions() string {
	style := lipgloss.NewStyle().Foreground(ui.SubtleColor)
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)

	actions := []string{
		keyStyle.Render("[s]") + style.Render(" Sync All"),
		keyStyle.Render("[i]") + style.Render(" Install"),
		keyStyle.Render("[d]") + style.Render(" Doctor"),
		keyStyle.Render("[m]") + style.Render(" Overrides"),
		keyStyle.Render("[u]") + style.Render(" Update"),
		keyStyle.Render("[tab]") + style.Render(" More"),
		keyStyle.Render("[enter]") + style.Render(" Sync Selected"),
		keyStyle.Render("[q]") + style.Render(" Quit"),
	}

	return strings.Join(actions, "   ")
}

func (m Model) renderSearchPrompt() string {
	promptStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	subtleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	cursor := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Render("█")

	filterInfo := ""
	if len(m.filteredIdxs) < len(m.configs) {
		filterInfo = subtleStyle.Render(fmt.Sprintf(" (%d/%d)", len(m.filteredIdxs), len(m.configs)))
	}

	return fmt.Sprintf("  %s %s%s%s",
		promptStyle.Render("Search:"),
		m.searchQuery,
		cursor,
		filterInfo,
	)
}

func (m Model) renderMultiSelectInfo() string {
	modeStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	subtleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	selectedCount := len(m.selected)
	info := fmt.Sprintf("  %s %s selected",
		modeStyle.Render("Multi-Select Mode:"),
		subtleStyle.Render(fmt.Sprintf("%d", selectedCount)),
	)

	if selectedCount > 0 {
		info += subtleStyle.Render(" • Press [S] to stow or [U] to unstow")
	} else {
		info += subtleStyle.Render(" • Press [space] to select • [v] to exit")
	}

	return info
}

func (m Model) renderDetailsPanel() string {
	if m.selectedIdx >= len(m.filteredIdxs) {
		return ""
	}

	actualIdx := m.filteredIdxs[m.selectedIdx]
	cfg := m.configs[actualIdx]
	linkStatus := m.linkStatus[cfg.Name]

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	labelStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	valueStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor)

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)

	var details strings.Builder
	details.WriteString(titleStyle.Render(fmt.Sprintf("  Details: %s", cfg.Name)))
	details.WriteString("\n\n")

	// Config info
	details.WriteString(labelStyle.Render("  Path: "))
	details.WriteString(valueStyle.Render(filepath.Join(m.dotfilesPath, cfg.Name)))
	details.WriteString("\n")

	if linkStatus != nil {
		details.WriteString(labelStyle.Render("  Files: "))
		details.WriteString(valueStyle.Render(fmt.Sprintf("%d total", linkStatus.TotalCount)))
		details.WriteString("\n")

		details.WriteString(labelStyle.Render("  Linked: "))
		if linkStatus.IsFullyLinked() {
			details.WriteString(okStyle.Render(fmt.Sprintf("✓ %d/%d", linkStatus.LinkedCount, linkStatus.TotalCount)))
		} else {
			details.WriteString(warnStyle.Render(fmt.Sprintf("⚠ %d/%d", linkStatus.LinkedCount, linkStatus.TotalCount)))
		}
		details.WriteString("\n")

		// Show conflicts if any
		missing := linkStatus.GetMissingFiles()
		if len(missing) > 0 {
			details.WriteString("\n")
			details.WriteString(labelStyle.Render("  Conflicts:\n"))
			for i, f := range missing {
				if i >= 5 {
					details.WriteString(labelStyle.Render(fmt.Sprintf("    ... and %d more\n", len(missing)-5)))
					break
				}
				icon := errStyle.Render("✗")
				details.WriteString(fmt.Sprintf("    %s %s", icon, valueStyle.Render(f.RelPath)))
				if f.Issue != "" {
					details.WriteString(labelStyle.Render(fmt.Sprintf(" (%s)", f.Issue)))
				}
				details.WriteString("\n")
			}
		}
	}

	// Dependencies
	if len(cfg.Dependencies.Apt) > 0 || len(cfg.Dependencies.Dnf) > 0 || len(cfg.Dependencies.Brew) > 0 {
		details.WriteString("\n")
		details.WriteString(labelStyle.Render("  Dependencies:\n"))
		allDeps := append(cfg.Dependencies.Apt, cfg.Dependencies.Dnf...)
		allDeps = append(allDeps, cfg.Dependencies.Brew...)
		for i, dep := range allDeps {
			if i >= 5 {
				details.WriteString(labelStyle.Render(fmt.Sprintf("    ... and %d more\n", len(allDeps)-5)))
				break
			}
			details.WriteString(valueStyle.Render(fmt.Sprintf("    • %s\n", dep)))
		}
	}

	details.WriteString("\n")
	details.WriteString(labelStyle.Render("  Press [D] or [esc] to close"))

	return boxStyle.Render(details.String())
}

func (m Model) renderHelpScreen() string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Underline(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor)

	var help strings.Builder

	help.WriteString(titleStyle.Render("Keyboard Shortcuts"))
	help.WriteString("\n\n")

	// Navigation
	help.WriteString(titleStyle.Render("Navigation"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("↑/k"), descStyle.Render("Move up")))
	help.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("↓/j"), descStyle.Render("Move down")))
	help.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render("e/→"), descStyle.Render("Expand/collapse config")))
	help.WriteString("\n")

	// Actions
	help.WriteString(titleStyle.Render("Actions"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("enter"), descStyle.Render("Sync selected config")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("s"), descStyle.Render("Sync all configs")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("i"), descStyle.Render("Install dependencies")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("d"), descStyle.Render("Run doctor check")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("m"), descStyle.Render("Configure overrides")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("u"), descStyle.Render("Update configs")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("r"), descStyle.Render("Refresh status")))
	help.WriteString("\n")

	// Enhanced Features
	help.WriteString(titleStyle.Render("Enhanced Features"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("/"), descStyle.Render("Search/filter configs")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("v"), descStyle.Render("Toggle multi-select mode")))
	help.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("space"), descStyle.Render("Select item (in multi-select)")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("S"), descStyle.Render("Bulk stow selected")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("U"), descStyle.Render("Bulk unstow selected")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("D"), descStyle.Render("Show details panel")))
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("t"), descStyle.Render("Cycle theme")))
	help.WriteString("\n")

	// Other
	help.WriteString(titleStyle.Render("Other"))
	help.WriteString("\n")
	help.WriteString(fmt.Sprintf("  %s        %s\n", keyStyle.Render("?"), descStyle.Render("Toggle this help screen")))
	help.WriteString(fmt.Sprintf("  %s      %s\n", keyStyle.Render("tab"), descStyle.Render("More commands menu")))
	help.WriteString(fmt.Sprintf("  %s    %s\n", keyStyle.Render("q/esc"), descStyle.Render("Quit/back")))

	help.WriteString("\n")
	help.WriteString(lipgloss.NewStyle().Foreground(ui.SubtleColor).Render("Press [?], [esc], or [q] to close"))

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		boxStyle.Render(help.String()),
	)
}

// GetResult returns the action result after the model exits
func (m Model) GetResult() *Result {
	return m.result
}

// Run starts the dashboard and returns the selected action
func Run(p *platform.Platform, driftSummary *stow.DriftSummary, linkStatus map[string]*stow.ConfigLinkStatus, machineStatus []MachineStatus, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool) (*Result, error) {
	m := New(p, driftSummary, linkStatus, machineStatus, configs, dotfilesPath, updateMsg, hasBaseline)

	finalModel, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(Model).GetResult(), nil
}

// SetupModel is the Bubbletea model for the setup screen (no config)
type SetupModel struct {
	width       int
	height      int
	platform    *platform.Platform
	updateMsg   string
	result      *Result
	quitting    bool
	selectedIdx int
}

// NewSetup creates a new setup model
func NewSetup(p *platform.Platform, updateMsg string) SetupModel {
	return SetupModel{
		platform:  p,
		updateMsg: updateMsg,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.result = &Result{Action: ActionQuit}
			m.quitting = true
			return m, tea.Quit

		case "i", "enter":
			m.result = &Result{Action: ActionInit}
			return m, tea.Quit

		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case "down", "j":
			if m.selectedIdx < 1 {
				m.selectedIdx++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m SetupModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header - same style as dashboard
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	title := titleStyle.Render("go4dot")
	platformInfo := ""
	if m.platform != nil {
		platformInfo = fmt.Sprintf(" %s (%s)", m.platform.OS, m.platform.PackageManager)
	}
	subtitle := subtitleStyle.Render(platformInfo)
	updateInfo := ""
	if m.updateMsg != "" {
		updateInfo = subtitleStyle.Render(" " + m.updateMsg)
	}

	b.WriteString(title + subtitle + updateInfo)
	b.WriteString("\n\n")

	// Status
	statusStyle := lipgloss.NewStyle().
		Foreground(ui.WarningColor).
		Bold(true)
	b.WriteString(statusStyle.Render("  No configuration found"))
	b.WriteString("\n\n")

	// Message
	b.WriteString(ui.TextStyle.Render("  No .go4dot.yaml found in current directory."))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("  Initialize go4dot to start managing your dotfiles."))
	b.WriteString("\n\n")

	// Options
	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle

	options := []struct {
		label string
		desc  string
	}{
		{"Initialize go4dot", "Set up a new .go4dot.yaml config"},
		{"Quit", "Exit go4dot"},
	}

	for i, opt := range options {
		prefix := "  "
		style := normalStyle
		if i == m.selectedIdx {
			prefix = "> "
			style = selectedStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(opt.label)))
		if i == m.selectedIdx {
			b.WriteString(subtitleStyle.Render(fmt.Sprintf("    %s", opt.desc)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Action bar - same style as dashboard
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	actions := []string{
		keyStyle.Render("[i]") + subtitleStyle.Render(" Initialize"),
		keyStyle.Render("[q]") + subtitleStyle.Render(" Quit"),
	}
	b.WriteString(strings.Join(actions, "   "))

	return b.String()
}

// GetResult returns the action result after the model exits
func (m SetupModel) GetResult() *Result {
	return m.result
}

// RunSetup starts the setup screen and returns the selected action
func RunSetup(p *platform.Platform, updateMsg string) (*Result, error) {
	m := NewSetup(p, updateMsg)

	finalModel, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(SetupModel).GetResult(), nil
}
