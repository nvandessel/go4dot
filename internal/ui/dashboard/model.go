package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

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
	ActionBulkSync
	ActionRefresh
)

// Result is returned when the dashboard exits
type Result struct {
	Action      Action
	ConfigName  string   // For ActionSyncConfig
	ConfigNames []string // For ActionBulkSync
}

// MachineStatus represents the status of a machine config for the dashboard
type MachineStatus struct {
	ID          string
	Description string
	Status      string // "configured", "missing", "error"
}

// Model is the Bubbletea model for the dashboard
type Model struct {
	width           int
	height          int
	platform        *platform.Platform
	driftSummary    *stow.DriftSummary
	linkStatus      map[string]*stow.ConfigLinkStatus // Per-config link status
	machineStatus   []MachineStatus
	configs         []config.ConfigItem
	dotfilesPath    string
	updateMsg       string
	selectedIdx     int
	expandedIdx     int // Index of expanded config (-1 = none)
	scrollOffset    int // Scroll offset within expanded file list
	filterMode      bool
	filterText      string
	filteredIdxs    []int
	selectedConfigs map[string]bool // Config name -> selected state
	result          *Result
	quitting        bool
	hasBaseline     bool // True if we have stored symlink counts (synced before)
	showHelp        bool
}

// keyMap defines the key bindings
type keyMap struct {
	Sync    key.Binding
	Doctor  key.Binding
	Install key.Binding
	Machine key.Binding
	Update  key.Binding
	Menu    key.Binding
	Quit    key.Binding
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Expand  key.Binding
	Filter  key.Binding
	Help    key.Binding
	Select  key.Binding
	All     key.Binding
	Bulk    key.Binding
	Refresh key.Binding
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
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
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
		key.WithHelp("e", "expand/collapse"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	All: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("shift+a", "select all"),
	),
	Bulk: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("shift+s", "sync selected"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
}

// New creates a new dashboard model
func New(p *platform.Platform, driftSummary *stow.DriftSummary, linkStatus map[string]*stow.ConfigLinkStatus, machineStatus []MachineStatus, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool) Model {
	m := Model{
		platform:        p,
		driftSummary:    driftSummary,
		linkStatus:      linkStatus,
		machineStatus:   machineStatus,
		configs:         configs,
		dotfilesPath:    dotfilesPath,
		updateMsg:       updateMsg,
		selectedIdx:     0,
		expandedIdx:     -1,
		hasBaseline:     hasBaseline,
		filterMode:      false,
		filterText:      "",
		filteredIdxs:    []int{},
		selectedConfigs: make(map[string]bool),
		showHelp:        false,
	}
	m.updateFilter()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Help overlay takes precedence
		if m.showHelp {
			switch msg.String() {
			case "?", "q", "esc":
				m.showHelp = false
			}
			return m, nil
		}

		// Special handling when in filter mode
		if m.filterMode {
			switch msg.String() {
			case "enter", "esc":
				m.filterMode = false
				return m, nil

			case "ctrl+c":
				m.filterMode = false
				m.filterText = ""
				m.updateFilter()
				return m, nil

			case "backspace":
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.updateFilter()
				}
				return m, nil

			default:
				// Capture text input (a-z, 0-9, -, _)
				if len(msg.String()) == 1 {
					r := []rune(msg.String())[0]
					if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '_' {
						m.filterText += msg.String()
						m.updateFilter()
					}
				}
				return m, nil
			}
		}

		switch {
		case key.Matches(msg, keys.Quit):
			m.result = &Result{Action: ActionQuit}
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Filter):
			m.filterMode = true
			return m, nil

		case key.Matches(msg, keys.Help):
			m.showHelp = true
			return m, nil

		case key.Matches(msg, keys.Refresh):
			m.result = &Result{Action: ActionRefresh}
			return m, tea.Quit

		case key.Matches(msg, keys.Sync):
			m.result = &Result{Action: ActionSync}
			return m, tea.Quit

		case key.Matches(msg, keys.Bulk):
			if len(m.selectedConfigs) > 0 {
				names := make([]string, 0, len(m.selectedConfigs))
				for name := range m.selectedConfigs {
					names = append(names, name)
				}
				m.result = &Result{
					Action:      ActionBulkSync,
					ConfigNames: names,
				}
				return m, tea.Quit
			}
			return m, nil

		case key.Matches(msg, keys.Select):
			if len(m.configs) > 0 && m.selectedIdx < len(m.configs) {
				name := m.configs[m.selectedIdx].Name
				m.selectedConfigs[name] = !m.selectedConfigs[name]
				if !m.selectedConfigs[name] {
					delete(m.selectedConfigs, name)
				}
			}
			return m, nil

		case key.Matches(msg, keys.All):
			// If all visible are selected, deselect them. Otherwise select all visible.
			allSelected := true
			if len(m.filteredIdxs) == 0 {
				allSelected = false
			}
			for _, idx := range m.filteredIdxs {
				if !m.selectedConfigs[m.configs[idx].Name] {
					allSelected = false
					break
				}
			}

			if allSelected {
				for _, idx := range m.filteredIdxs {
					delete(m.selectedConfigs, m.configs[idx].Name)
				}
			} else {
				for _, idx := range m.filteredIdxs {
					m.selectedConfigs[m.configs[idx].Name] = true
				}
			}
			return m, nil

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
			// For now, we'll just return a special action to show the menu
			// In a more complex app, we might switch models here
			m.result = &Result{Action: ActionList} // Using ActionList as a placeholder for "More"
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if len(m.filteredIdxs) > 0 {
				// Find current position in filtered list
				currentPos := -1
				for i, idx := range m.filteredIdxs {
					if idx == m.selectedIdx {
						currentPos = i
						break
					}
				}

				if currentPos > 0 {
					m.selectedIdx = m.filteredIdxs[currentPos-1]
					// Reset expansion when changing selection
					m.expandedIdx = -1
					m.scrollOffset = 0
				}
			}

		case key.Matches(msg, keys.Down):
			if len(m.filteredIdxs) > 0 {
				// Find current position in filtered list
				currentPos := -1
				for i, idx := range m.filteredIdxs {
					if idx == m.selectedIdx {
						currentPos = i
						break
					}
				}

				if currentPos != -1 && currentPos < len(m.filteredIdxs)-1 {
					m.selectedIdx = m.filteredIdxs[currentPos+1]
					// Reset expansion when changing selection
					m.expandedIdx = -1
					m.scrollOffset = 0
				}
			}

		case key.Matches(msg, keys.Expand):
			// Toggle expansion for selected config
			if m.expandedIdx == m.selectedIdx {
				m.expandedIdx = -1
				m.scrollOffset = 0
			} else {
				m.expandedIdx = m.selectedIdx
				m.scrollOffset = 0
			}

		case key.Matches(msg, keys.Enter):
			if len(m.configs) > 0 && m.selectedIdx < len(m.configs) {
				m.result = &Result{
					Action:     ActionSyncConfig,
					ConfigName: m.configs[m.selectedIdx].Name,
				}
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// updateFilter recalculates which configs match the current filter text
func (m *Model) updateFilter() {
	m.filteredIdxs = []int{}

	// No filter text = show all configs
	if m.filterText == "" {
		for i := range m.configs {
			m.filteredIdxs = append(m.filteredIdxs, i)
		}
		return
	}

	// Filter configs by case-insensitive substring match
	filterLower := strings.ToLower(m.filterText)
	for i, cfg := range m.configs {
		if strings.Contains(strings.ToLower(cfg.Name), filterLower) {
			m.filteredIdxs = append(m.filteredIdxs, i)
		}
	}

	// Reset selection to first matching config if current selection is not in filtered list
	found := false
	for _, idx := range m.filteredIdxs {
		if idx == m.selectedIdx {
			found = true
			break
		}
	}

	if !found && len(m.filteredIdxs) > 0 {
		m.selectedIdx = m.filteredIdxs[0]
	}
}

// renderHelp renders the help overlay with all keyboard shortcuts
func (m Model) renderHelp() string {
	var b strings.Builder

	boxWidth := 60
	if m.width > 0 && m.width < boxWidth+4 {
		boxWidth = m.width - 4
	}

	// Styles
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(boxWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Foreground(ui.SecondaryColor).
		Bold(true).
		MarginTop(1).
		MarginLeft(2)

	keyStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(14).
		Align(lipgloss.Right)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		MarginLeft(2)

	subtleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Width(boxWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	// Build help content
	b.WriteString(titleStyle.Render("go4dot Dashboard - Keyboard Shortcuts"))
	b.WriteString("\n")

	// Navigation section
	b.WriteString(headerStyle.Render("Navigation"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("↑/k"), descStyle.Render("Move selection up")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("↓/j"), descStyle.Render("Move selection down")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("e / →"), descStyle.Render("Expand/collapse details")))

	// Actions section
	b.WriteString(headerStyle.Render("Actions"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("enter"), descStyle.Render("Sync selected config")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("s"), descStyle.Render("Sync all configs")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("shift+s"), descStyle.Render("Sync selected configs")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("i"), descStyle.Render("Run install wizard")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("u"), descStyle.Render("Update dotfiles/deps")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("r"), descStyle.Render("Refresh dashboard")))

	// Selection section
	b.WriteString(headerStyle.Render("Selection"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("space"), descStyle.Render("Toggle selection")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("shift+a"), descStyle.Render("Select/deselect all")))

	// Filter section
	b.WriteString(headerStyle.Render("Filter"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("/"), descStyle.Render("Enter filter mode")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("esc"), descStyle.Render("Exit filter mode")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("ctrl+c"), descStyle.Render("Clear filter and exit")))

	// Status Indicators section
	b.WriteString(headerStyle.Render("Status Indicators"))
	b.WriteString("\n")
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle
	b.WriteString(fmt.Sprintf("  %s  %s\n", okStyle.Render("✓"), descStyle.Render("Fully linked / Configured")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", warnStyle.Render("◆"), descStyle.Render("Partially linked")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", warnStyle.Render("⚠"), descStyle.Render("Conflicts detected")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", errStyle.Render("✗"), descStyle.Render("Not linked / Missing")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", subtleStyle.Render("•"), descStyle.Render("Status tags (deps, external)")))

	// Other section
	b.WriteString(headerStyle.Render("Other"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("d"), descStyle.Render("Run doctor check")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("m"), descStyle.Render("Configure overrides")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("tab"), descStyle.Render("More commands menu")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("?"), descStyle.Render("Toggle help screen")))
	b.WriteString(fmt.Sprintf("%s%s\n", keyStyle.Render("q / esc"), descStyle.Render("Quit dashboard")))

	b.WriteString(subtleStyle.Render("Press ?, q, or esc to close"))

	// Wrap in a bordered box
	return ui.BoxStyle.
		Width(boxWidth).
		Render(b.String())
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// Status summary
	status := m.renderStatus()
	b.WriteString(status)
	b.WriteString("\n\n")

	// Filter bar
	if m.filterMode || m.filterText != "" {
		b.WriteString(m.renderFilterBar())
		b.WriteString("\n\n")
	}

	// Machine status (if any)
	if len(m.machineStatus) > 0 {
		machineStatus := m.renderMachineStatus()
		b.WriteString(machineStatus)
		b.WriteString("\n\n")
	}

	// Config list
	configList := m.renderConfigList()
	b.WriteString(configList)

	// Main content
	content := b.String()

	// Action bar (pinned to bottom)
	actions := m.renderActions()

	// Calculate how much space we have for content
	headerHeight := lipgloss.Height(header) + 2
	statusHeight := lipgloss.Height(status) + 2
	filterHeight := 0
	if m.filterMode || m.filterText != "" {
		filterHeight = lipgloss.Height(m.renderFilterBar()) + 2
	}
	machineHeight := 0
	if len(m.machineStatus) > 0 {
		machineHeight = lipgloss.Height(m.renderMachineStatus()) + 2
	}
	actionsHeight := lipgloss.Height(actions)

	// Total height used by non-config-list elements
	fixedHeight := headerHeight + statusHeight + filterHeight + machineHeight + actionsHeight

	// Fill remaining space with newlines to push actions to bottom
	// but only if we have enough height
	if m.height > fixedHeight {
		configListHeight := lipgloss.Height(configList)
		padding := m.height - fixedHeight - configListHeight
		if padding > 0 {
			content += strings.Repeat("\n", padding)
		}
	}

	finalView := content + "\n" + actions

	// Show help overlay if active
	if m.showHelp {
		help := m.renderHelp()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, help, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(ui.SubtleColor))
	}

	return finalView
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
	var status string

	// If we haven't synced before (no baseline), show that clearly
	if !m.hasBaseline {
		status = lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render("  Not synced yet - press [s] to create symlinks")
	} else if m.driftSummary != nil && m.driftSummary.HasDrift() {
		// We have baseline - check for drift
		status = lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render(fmt.Sprintf("  %d config(s) need syncing", m.driftSummary.DriftedConfigs))
	} else {
		// Baseline exists and no drift
		status = lipgloss.NewStyle().
			Foreground(ui.SecondaryColor).
			Bold(true).
			Render("  All synced")
	}

	if len(m.selectedConfigs) > 0 {
		selectionInfo := lipgloss.NewStyle().
			Foreground(ui.PrimaryColor).
			Bold(true).
			Render(fmt.Sprintf(" • %d selected", len(m.selectedConfigs)))
		return status + selectionInfo
	}

	return status
}

// renderFilterBar renders the filter input/display bar
func (m Model) renderFilterBar() string {
	style := lipgloss.NewStyle().Foreground(ui.PrimaryColor)
	subtleStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	var filterIndicator string
	if m.filterMode {
		// Active mode: show cursor
		filterIndicator = style.Render("Filter:") + " " + m.filterText + style.Render("█")
	} else {
		// Inactive but filtered: show subtle
		filterIndicator = subtleStyle.Render("Filter:") + " " + m.filterText
	}

	// Show match count when filtering
	countText := ""
	if m.filterText != "" {
		countText = subtleStyle.Render(
			fmt.Sprintf(" (%d of %d)", len(m.filteredIdxs), len(m.configs)),
		)
	}

	return "  " + filterIndicator + countText
}

func (m Model) renderMachineStatus() string {
	var parts []string

	headerStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor).Bold(true)
	parts = append(parts, headerStyle.Render("  Overrides:"))

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)

	for _, status := range m.machineStatus {
		icon := ""
		switch status.Status {
		case "configured":
			icon = okStyle.Render("✓")
		case "missing":
			icon = errStyle.Render("✗")
		case "error":
			icon = errStyle.Render("!")
		}

		parts = append(parts, fmt.Sprintf("%s %s", icon, status.ID))
	}

	return strings.Join(parts, "  ")
}

// configStatusInfo holds detailed status information for a config
type configStatusInfo struct {
	icon       string   // Primary status icon
	statusText string   // "X/Y" display
	statusTags []string // Additional status tags (conflicts, deps, external)
}

// getConfigStatusInfo analyzes a config and returns detailed status information
func (m Model) getConfigStatusInfo(cfg config.ConfigItem, linkStatus *stow.ConfigLinkStatus, drift *stow.DriftResult) configStatusInfo {
	info := configStatusInfo{
		statusTags: []string{},
	}

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle

	// Analyze link status
	if linkStatus != nil {
		// Check for conflicts
		conflictCount := 0
		for _, f := range linkStatus.Files {
			if !f.IsLinked && (strings.Contains(strings.ToLower(f.Issue), "conflict") ||
				strings.Contains(strings.ToLower(f.Issue), "exists") ||
				strings.Contains(strings.ToLower(f.Issue), "elsewhere")) {
				conflictCount++
			}
		}

		// Determine primary icon
		if conflictCount > 0 {
			info.icon = warnStyle.Render("⚠")
			info.statusTags = append(info.statusTags, fmt.Sprintf("conflicts (%d)", conflictCount))
		} else if linkStatus.IsFullyLinked() {
			info.icon = okStyle.Render("✓")
		} else if linkStatus.LinkedCount > 0 {
			info.icon = warnStyle.Render("◆") // Partial link indicator
		} else {
			info.icon = errStyle.Render("✗")
		}

		// File count
		info.statusText = fmt.Sprintf("%d/%d", linkStatus.LinkedCount, linkStatus.TotalCount)
	} else if drift != nil {
		// Fallback to drift-based display if no link status
		if drift.HasDrift {
			info.icon = warnStyle.Render("◆")
			info.statusText = fmt.Sprintf("%d new", len(drift.NewFiles))
		} else {
			info.icon = okStyle.Render("●")
			info.statusText = fmt.Sprintf("%d files", drift.CurrentCount)
		}
	} else {
		info.icon = ui.SubtleStyle.Render("○")
		info.statusText = "unknown"
	}

	// Check external dependencies
	if len(cfg.ExternalDeps) > 0 {
		missingExternal := false
		home := os.Getenv("HOME")
		for _, ext := range cfg.ExternalDeps {
			dest := ext.Destination
			if dest == "" {
				continue
			}
			// Resolve destination relative to home if not absolute
			fullDest := dest
			if !filepath.IsAbs(dest) {
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

	// Check module dependencies
	if len(cfg.DependsOn) > 0 {
		missingDep := false
		for _, depName := range cfg.DependsOn {
			depStatus, ok := m.linkStatus[depName]
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

func (m Model) renderConfigList() string {
	var lines []string

	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	subtleStyle := ui.SubtleStyle

	// Build a map of drift results for quick lookup
	driftMap := make(map[string]*stow.DriftResult)
	if m.driftSummary != nil {
		for i := range m.driftSummary.Results {
			r := &m.driftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}

	// If filtered but no matches, show message
	if m.filterText != "" && len(m.filteredIdxs) == 0 {
		return subtleStyle.Render("  No configs match filter: \"" + m.filterText + "\"")
	}

	for _, i := range m.filteredIdxs {
		cfg := m.configs[i]
		var line string
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}

		checkbox := "[ ]"
		if m.selectedConfigs[cfg.Name] {
			checkbox = okStyle.Render("[✓]")
		}

		nameStyle := normalStyle
		if i == m.selectedIdx {
			nameStyle = selectedStyle
		}

		// Get link status for this config
		linkStatus := m.linkStatus[cfg.Name]
		drift := driftMap[cfg.Name]

		// Get enhanced status info
		statusInfo := m.getConfigStatusInfo(cfg, linkStatus, drift)

		// Pad name to align status
		maxNameLen := 18
		nameLen := len(cfg.Name)
		if nameLen > maxNameLen {
			nameLen = maxNameLen
		}
		dots := subtleStyle.Render(strings.Repeat(".", maxNameLen-nameLen+2))

		// Build status display
		statusDisplay := statusInfo.icon + " " + subtleStyle.Render(statusInfo.statusText)
		if len(statusInfo.statusTags) > 0 {
			statusDisplay += " " + subtleStyle.Render("•") + " " + subtleStyle.Render(strings.Join(statusInfo.statusTags, " • "))
		}

		line = fmt.Sprintf("%s%s %s %s %s",
			prefix,
			checkbox,
			nameStyle.Render(cfg.Name),
			dots,
			statusDisplay,
		)

		lines = append(lines, line)

		// Show expanded details if this config is expanded
		if i == m.expandedIdx {
			if linkStatus != nil {
				details := m.renderConfigDetails(cfg, linkStatus)
				lines = append(lines, details)
			} else {
				lines = append(lines, subtleStyle.Render("      No status information available"))
			}
		} else if i == m.selectedIdx && linkStatus != nil {
			// Show summary hint when selected but not expanded
			if !linkStatus.IsFullyLinked() {
				lines = append(lines, subtleStyle.Render("      press [e] to expand"))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderActions() string {
	style := lipgloss.NewStyle().Foreground(ui.SubtleColor)
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)

	actions := []string{
		keyStyle.Render("[?]") + style.Render(" Help"),
		keyStyle.Render("[/]") + style.Render(" Filter"),
		keyStyle.Render("[space]") + style.Render(" Select"),
		keyStyle.Render("[s]") + style.Render(" Sync All"),
		keyStyle.Render("[r]") + style.Render(" Refresh"),
		keyStyle.Render("[i]") + style.Render(" Install"),
		keyStyle.Render("[d]") + style.Render(" Doctor"),
		keyStyle.Render("[m]") + style.Render(" Overrides"),
		keyStyle.Render("[u]") + style.Render(" Update"),
		keyStyle.Render("[tab]") + style.Render(" More"),
		keyStyle.Render("[q]") + style.Render(" Quit"),
	}

	return strings.Join(actions, "  ")
}

// renderConfigDetails renders comprehensive details for an expanded config
func (m Model) renderConfigDetails(cfg config.ConfigItem, linkStatus *stow.ConfigLinkStatus) string {
	var lines []string
	indent := "      "

	// Styles
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle
	subtleStyle := ui.SubtleStyle
	headerStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)

	// Description header
	if cfg.Description != "" {
		lines = append(lines, subtleStyle.Render(indent+cfg.Description))
		lines = append(lines, "")
	}

	// File breakdown section
	if linkStatus != nil {
		lines = append(lines, headerStyle.Render(indent+"Files:"))

		// Get file lists
		var linked []stow.FileStatus
		var missing []stow.FileStatus
		var conflicts []stow.FileStatus

		for _, f := range linkStatus.Files {
			if f.IsLinked {
				linked = append(linked, f)
			} else {
				// Check if it's a conflict or just missing
				issue := strings.ToLower(f.Issue)
				if strings.Contains(issue, "conflict") ||
					strings.Contains(issue, "exists") ||
					strings.Contains(issue, "elsewhere") {
					conflicts = append(conflicts, f)
				} else {
					missing = append(missing, f)
				}
			}
		}

		// Linked files
		if len(linked) > 0 {
			lines = append(lines, okStyle.Render(fmt.Sprintf(indent+"  ✓ %d linked", len(linked))))
			displayCount := min(3, len(linked))
			for i := 0; i < displayCount; i++ {
				lines = append(lines, subtleStyle.Render(indent+"    "+linked[i].RelPath))
			}
			if len(linked) > 3 {
				lines = append(lines, subtleStyle.Render(
					fmt.Sprintf(indent+"    ... %d more", len(linked)-3)))
			}
		}

		// Conflicts
		if len(conflicts) > 0 {
			lines = append(lines, warnStyle.Render(fmt.Sprintf(indent+"  ⚠ %d conflicts", len(conflicts))))
			for _, f := range conflicts {
				reason := f.Issue
				if reason == "" {
					reason = "file exists"
				}
				lines = append(lines, subtleStyle.Render(
					fmt.Sprintf(indent+"    %s (%s)", f.RelPath, reason)))
			}
		}

		// Missing/not linked files
		if len(missing) > 0 {
			lines = append(lines, errStyle.Render(fmt.Sprintf(indent+"  ✗ %d not linked", len(missing))))
			displayCount := min(3, len(missing))
			for i := 0; i < displayCount; i++ {
				reason := missing[i].Issue
				if reason == "" {
					reason = "not linked"
				}
				lines = append(lines, subtleStyle.Render(
					fmt.Sprintf(indent+"    %s (%s)", missing[i].RelPath, reason)))
			}
			if len(missing) > 3 {
				lines = append(lines, subtleStyle.Render(
					fmt.Sprintf(indent+"    ... %d more", len(missing)-3)))
			}
		}

		lines = append(lines, "")
	}

	// Dependencies section
	if len(cfg.DependsOn) > 0 {
		lines = append(lines, headerStyle.Render(indent+"Dependencies:"))
		displayCount := min(5, len(cfg.DependsOn))
		for i := 0; i < displayCount; i++ {
			lines = append(lines, subtleStyle.Render(indent+"  • "+cfg.DependsOn[i]))
		}
		if len(cfg.DependsOn) > 5 {
			lines = append(lines, subtleStyle.Render(
				fmt.Sprintf(indent+"  ... %d more", len(cfg.DependsOn)-5)))
		}
		lines = append(lines, "")
	}

	// External dependencies section
	if len(cfg.ExternalDeps) > 0 {
		lines = append(lines, headerStyle.Render(indent+"External:"))
		displayCount := min(3, len(cfg.ExternalDeps))
		for i := 0; i < displayCount; i++ {
			extDep := cfg.ExternalDeps[i]
			// Show URL
			displayURL := extDep.URL
			lines = append(lines, subtleStyle.Render(indent+"  • "+displayURL))
		}
		if len(cfg.ExternalDeps) > 3 {
			lines = append(lines, subtleStyle.Render(
				fmt.Sprintf(indent+"  ... %d more", len(cfg.ExternalDeps)-3)))
		}
		lines = append(lines, "")
	}

	// Statistics summary
	if linkStatus != nil {
		statsLine := fmt.Sprintf("Total: %d files", linkStatus.TotalCount)
		lines = append(lines, subtleStyle.Render(indent+statsLine))
	}

	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
