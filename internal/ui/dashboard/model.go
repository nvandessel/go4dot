package dashboard

import (
	"fmt"
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

		line = fmt.Sprintf("%s%s %s %s %s %s",
			prefix,
			checkbox,
			nameStyle.Render(cfg.Name),
			dots,
			statusIcon,
			subtleStyle.Render(statusText),
		)

		lines = append(lines, line)

		// Show expanded file list if this config is expanded
		if i == m.expandedIdx && linkStatus != nil && len(linkStatus.Files) > 0 {
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
					lines = append(lines, subtleStyle.Render(fmt.Sprintf("      ... %d more (scroll with shift+j/k)", remaining)))
				}
			}
		} else if i == m.selectedIdx && linkStatus != nil {
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
		keyStyle.Render("[/]") + style.Render(" Filter"),
		keyStyle.Render("[space]") + style.Render(" Select"),
		keyStyle.Render("[shift+a]") + style.Render(" All"),
		keyStyle.Render("[shift+s]") + style.Render(" Sync Selected"),
		keyStyle.Render("[s]") + style.Render(" Sync All"),
		keyStyle.Render("[i]") + style.Render(" Install"),
		keyStyle.Render("[d]") + style.Render(" Doctor"),
		keyStyle.Render("[m]") + style.Render(" Overrides"),
		keyStyle.Render("[u]") + style.Render(" Update"),
		keyStyle.Render("[tab]") + style.Render(" More"),
		keyStyle.Render("[enter]") + style.Render(" Sync Config"),
		keyStyle.Render("[q]") + style.Render(" Quit"),
	}

	return strings.Join(actions, "   ")
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
