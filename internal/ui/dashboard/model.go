package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	Action         Action
	ConfigName     string   // For ActionSyncConfig
	ConfigNames    []string // For ActionBulkSync
	FilterText     string   // For preserving state
	SelectedConfig string   // For preserving state
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
	listOffset      int // Scroll offset for the config list
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
	refreshing      bool
	lastListHeight  int // Last calculated height of the config list
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
func New(p *platform.Platform, driftSummary *stow.DriftSummary, linkStatus map[string]*stow.ConfigLinkStatus, machineStatus []MachineStatus, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool, initialFilter string, initialSelected string) Model {
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
		filterText:      initialFilter,
		filteredIdxs:    []int{},
		selectedConfigs: make(map[string]bool),
		showHelp:        false,
	}
	m.updateFilter()
	m.updateLayout()

	// Try to restore selection
	if initialSelected != "" {
		for i, cfg := range m.configs {
			if cfg.Name == initialSelected {
				// Check if it's in the filtered list
				for _, fIdx := range m.filteredIdxs {
					if fIdx == i {
						m.selectedIdx = i
						break
					}
				}
				break
			}
		}
	}

	return m
}

// Init initializes the dashboard model
func (m Model) Init() tea.Cmd {
	return nil
}

type refreshMsg struct{}

func doRefresh() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return refreshMsg{}
	})
}

// Update handles messages and updates the dashboard model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case refreshMsg:
		m.result = &Result{
			Action:         ActionRefresh,
			FilterText:     m.filterText,
			SelectedConfig: m.getSelectedConfigName(),
		}
		return m, tea.Quit

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
			m.result = &Result{
				Action:         ActionQuit,
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Filter):
			m.filterMode = true
			return m, nil

		case key.Matches(msg, keys.Help):
			m.showHelp = true
			return m, nil

		case key.Matches(msg, keys.Refresh):
			m.refreshing = true
			return m, doRefresh()

		case key.Matches(msg, keys.Sync):
			m.result = &Result{
				Action:         ActionSync,
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Bulk):
			if len(m.selectedConfigs) > 0 {
				names := make([]string, 0, len(m.selectedConfigs))
				for name := range m.selectedConfigs {
					names = append(names, name)
				}
				m.result = &Result{
					Action:         ActionBulkSync,
					ConfigNames:    names,
					FilterText:     m.filterText,
					SelectedConfig: m.getSelectedConfigName(),
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
			m.result = &Result{
				Action:         ActionDoctor,
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Install):
			m.result = &Result{
				Action:         ActionInstall,
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Machine):
			m.result = &Result{
				Action:         ActionMachineConfig,
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Update):
			m.result = &Result{
				Action:         ActionUpdate,
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
			return m, tea.Quit

		case key.Matches(msg, keys.Menu):
			// For now, we'll just return a special action to show the menu
			// In a more complex app, we might switch models here
			m.result = &Result{
				Action:         ActionList, // Using ActionList as a placeholder for "More"
				FilterText:     m.filterText,
				SelectedConfig: m.getSelectedConfigName(),
			}
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
					m.ensureVisible()
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
					m.ensureVisible()
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
					Action:         ActionSyncConfig,
					ConfigName:     m.configs[m.selectedIdx].Name,
					FilterText:     m.filterText,
					SelectedConfig: m.getSelectedConfigName(),
				}
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
	}

	return m, nil
}

// updateLayout recalculates layout-dependent values
func (m *Model) updateLayout() {
	// Rough estimate of fixed heights
	fixedHeight := 15 // Header, status, filter, machine, actions
	m.lastListHeight = m.height - fixedHeight
	if m.lastListHeight < 5 {
		m.lastListHeight = 5
	}
	m.ensureVisible()
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
	m.ensureVisible()
}

// ensureVisible ensures the selected item is within the visible area of the list
func (m *Model) ensureVisible() {
	if len(m.filteredIdxs) == 0 {
		return
	}

	// Find current position in filtered list
	currentPos := -1
	for i, idx := range m.filteredIdxs {
		if idx == m.selectedIdx {
			currentPos = i
			break
		}
	}

	if currentPos == -1 {
		return
	}

	// Use last calculated height or a default
	listHeight := m.lastListHeight
	if listHeight < 1 {
		listHeight = 10
	}

	if currentPos < m.listOffset {
		m.listOffset = currentPos
	} else if currentPos >= m.listOffset+listHeight {
		m.listOffset = currentPos - listHeight + 1
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

// View renders the dashboard view
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.refreshing {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().
				Foreground(ui.PrimaryColor).
				Bold(true).
				Render("Refreshing dashboard..."),
		)
	}

	// Show help overlay if active
	if m.showHelp {
		help := m.renderHelp()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, help, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(ui.SubtleColor))
	}

	var b strings.Builder

	// 1. Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// 2. Status summary
	status := m.renderStatus()
	b.WriteString(status)
	b.WriteString("\n\n")

	// 3. Filter bar
	filterBar := ""
	if m.filterMode || m.filterText != "" {
		filterBar = m.renderFilterBar()
		b.WriteString(filterBar)
		b.WriteString("\n\n")
	}

	// 4. Machine status (if any)
	machineStatus := ""
	if len(m.machineStatus) > 0 {
		machineStatus = m.renderMachineStatus()
		b.WriteString(machineStatus)
		b.WriteString("\n\n")
	}

	// Calculate available height for the main content area
	headerHeight := lipgloss.Height(header) + 2
	statusHeight := lipgloss.Height(status) + 2
	filterHeight := 0
	if filterBar != "" {
		filterHeight = lipgloss.Height(filterBar) + 2
	}
	machineHeight := 0
	if machineStatus != "" {
		machineHeight = lipgloss.Height(machineStatus) + 2
	}
	actions := m.renderActions()
	actionsHeight := lipgloss.Height(actions) + 1

	availableHeight := m.height - headerHeight - statusHeight - filterHeight - machineHeight - actionsHeight - 2
	if availableHeight < 5 {
		availableHeight = 5 // Minimum height
	}
	m.lastListHeight = availableHeight

	// 5. Main Content (Split View)
	sidebarWidth := 30
	if m.width < 80 {
		sidebarWidth = m.width / 3
	}
	if sidebarWidth < 20 {
		sidebarWidth = 20
	}

	mainWidth := m.width - sidebarWidth - 3 // -3 for borders/spacing

	// Sidebar (Config List)
	sidebarContent := m.renderConfigList(sidebarWidth, availableHeight)

	// Add scroll indicators if needed
	sidebarLines := strings.Split(sidebarContent, "\n")
	if m.listOffset > 0 && len(sidebarLines) > 0 {
		sidebarLines[0] = lipgloss.NewStyle().Foreground(ui.PrimaryColor).Render("  ↑ more")
	}
	if m.listOffset+availableHeight < len(m.filteredIdxs) && len(sidebarLines) > 0 {
		sidebarLines[len(sidebarLines)-1] = lipgloss.NewStyle().Foreground(ui.PrimaryColor).Render("  ↓ more")
	}
	sidebarContent = strings.Join(sidebarLines, "\n")

	sidebar := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(ui.SubtleColor).
		Width(sidebarWidth).
		Height(availableHeight).
		Render(sidebarContent)

	// Main Panel (Details)
	var mainContent string
	if len(m.configs) > 0 && m.selectedIdx < len(m.configs) {
		cfg := m.configs[m.selectedIdx]
		linkStatus := m.linkStatus[cfg.Name]
		mainContent = m.renderConfigDetails(cfg, linkStatus, mainWidth, availableHeight)
	} else {
		mainContent = lipgloss.Place(mainWidth, availableHeight, lipgloss.Center, lipgloss.Center,
			ui.SubtleStyle.Render("No configuration selected"),
		)
	}

	mainPanel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true, true, true, true).
		BorderForeground(ui.PrimaryColor).
		Width(mainWidth).
		Height(availableHeight).
		Padding(0, 1).
		Render(mainContent)

	// Join Sidebar and Main Panel
	content := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", mainPanel)
	b.WriteString(content)

	// 6. Action bar (pinned to bottom)
	finalView := b.String()

	// Fill remaining space with newlines to push actions to bottom
	contentHeight := lipgloss.Height(finalView)
	padding := m.height - contentHeight - actionsHeight
	if padding > 0 {
		finalView += strings.Repeat("\n", padding)
	}

	return finalView + "\n" + actions
}

// renderHeader renders the dashboard header
func (m Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.PrimaryColor).
		Bold(true).
		Padding(0, 2)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		MarginLeft(1)

	title := titleStyle.Render("GO4DOT DASHBOARD")

	platformInfo := ""
	if m.platform != nil {
		platformInfo = fmt.Sprintf("%s (%s)", m.platform.OS, m.platform.PackageManager)
	}

	subtitle := subtitleStyle.Render(platformInfo)

	updateInfo := ""
	if m.updateMsg != "" {
		updateInfo = lipgloss.NewStyle().
			Foreground(ui.SecondaryColor).
			Bold(true).
			MarginLeft(2).
			Render(m.updateMsg)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, title, subtitle, updateInfo)
}

// renderStatus renders the overall sync status
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

// renderMachineStatus renders the status of machine-specific overrides
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

	// Check module dependencies
	if len(cfg.DependsOn) > 0 && m.linkStatus != nil {
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

// renderConfigList renders the list of dotfile configurations with scrolling support
func (m Model) renderConfigList(width, height int) string {
	var lines []string

	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle.Copy().Width(width - 2)
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

	// Calculate visible range
	endIdx := m.listOffset + height
	if endIdx > len(m.filteredIdxs) {
		endIdx = len(m.filteredIdxs)
	}

	for i := m.listOffset; i < endIdx; i++ {
		idx := m.filteredIdxs[i]
		cfg := m.configs[idx]
		var line string
		prefix := "  "
		if idx == m.selectedIdx {
			prefix = "> "
		}

		checkbox := "[ ]"
		if m.selectedConfigs[cfg.Name] {
			checkbox = okStyle.Render("[✓]")
		}

		nameStyle := normalStyle
		if idx == m.selectedIdx {
			nameStyle = selectedStyle
		}

		// Get link status for this config
		linkStatus := m.linkStatus[cfg.Name]
		drift := driftMap[cfg.Name]

		// Get enhanced status info
		statusInfo := m.getConfigStatusInfo(cfg, linkStatus, drift)

		// Pad name to align status
		maxNameLen := width - 15
		if maxNameLen < 10 {
			maxNameLen = 10
		}
		name := cfg.Name
		if len(name) > maxNameLen {
			name = name[:maxNameLen-3] + "..."
		}

		line = fmt.Sprintf("%s%s %s %s",
			prefix,
			checkbox,
			nameStyle.Render(name),
			statusInfo.icon,
		)

		lines = append(lines, line)
	}

	// Fill remaining height with empty lines
	for len(lines) < height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// renderActions renders the bottom action bar responsively
func (m Model) renderActions() string {
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	type action struct {
		key      string
		label    string
		priority int // Lower is higher priority
	}

	allActions := []action{
		{"?", "Help", 0},
		{"q", "Quit", 0},
		{"s", "Sync All", 1},
		{"/", "Filter", 1},
		{"space", "Select", 2},
		{"r", "Refresh", 2},
		{"i", "Install", 3},
		{"u", "Update", 3},
		{"d", "Doctor", 4},
		{"m", "Overrides", 4},
		{"tab", "More", 5},
	}

	var visibleActions []string
	currentWidth := 0
	margin := 3

	for _, a := range allActions {
		rendered := keyStyle.Render("["+a.key+"]") + " " + descStyle.Render(a.label)
		width := lipgloss.Width(rendered)

		if currentWidth+width+margin > m.width && len(visibleActions) > 0 {
			// If we're out of space, we could wrap or just stop.
			// For a "sexy" UI, let's try to fit as many as possible on one line,
			// and maybe hide lower priority ones if the screen is too small.
			if a.priority > 2 {
				continue
			}
		}

		visibleActions = append(visibleActions, rendered)
		currentWidth += width + margin
	}

	return strings.Join(visibleActions, "   ")
}

// renderConfigDetails renders comprehensive details for an expanded config
func (m Model) renderConfigDetails(cfg config.ConfigItem, linkStatus *stow.ConfigLinkStatus, width, height int) string {
	var lines []string

	// Styles
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

	// Title & Status Badge
	driftMap := make(map[string]*stow.DriftResult)
	if m.driftSummary != nil {
		for i := range m.driftSummary.Results {
			r := &m.driftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}
	statusInfo := m.getConfigStatusInfo(cfg, linkStatus, driftMap[cfg.Name])

	title := titleStyle.Render(strings.ToUpper(cfg.Name))
	statusBadge := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.SubtleColor).
		Padding(0, 1).
		MarginLeft(1).
		Render(statusInfo.statusText)

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, title, statusBadge))
	lines = append(lines, "")

	// Description
	if cfg.Description != "" {
		descStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Italic(true).Width(width - 4)
		lines = append(lines, descStyle.Render(cfg.Description))
		lines = append(lines, "")
	}

	// File breakdown section
	if linkStatus != nil {
		lines = append(lines, headerStyle.Render("FILES"))

		// Get file lists
		var linked []stow.FileStatus
		var missing []stow.FileStatus
		var conflicts []stow.FileStatus

		for _, f := range linkStatus.Files {
			if f.IsLinked {
				linked = append(linked, f)
			} else {
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
			lines = append(lines, okStyle.Render(fmt.Sprintf("✓ %d linked", len(linked))))
			displayCount := min(5, len(linked))
			for i := 0; i < displayCount; i++ {
				lines = append(lines, subtleStyle.Render("  "+linked[i].RelPath))
			}
			if len(linked) > 5 {
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("  ... %d more", len(linked)-5)))
			}
		}

		// Conflicts
		if len(conflicts) > 0 {
			lines = append(lines, warnStyle.Render(fmt.Sprintf("⚠ %d conflicts", len(conflicts))))
			for _, f := range conflicts {
				reason := f.Issue
				if reason == "" {
					reason = "file exists"
				}
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("  %s (%s)", f.RelPath, reason)))
			}
		}

		// Missing/not linked files
		if len(missing) > 0 {
			lines = append(lines, errStyle.Render(fmt.Sprintf("✗ %d not linked", len(missing))))
			displayCount := min(5, len(missing))
			for i := 0; i < displayCount; i++ {
				reason := missing[i].Issue
				if reason == "" {
					reason = "not linked"
				}
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("  %s (%s)", missing[i].RelPath, reason)))
			}
			if len(missing) > 5 {
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("  ... %d more", len(missing)-5)))
			}
		}

		lines = append(lines, "")
	}

	// Dependencies section
	if len(cfg.DependsOn) > 0 {
		lines = append(lines, headerStyle.Render("DEPENDENCIES"))
		for _, dep := range cfg.DependsOn {
			lines = append(lines, subtleStyle.Render("• "+dep))
		}
		lines = append(lines, "")
	}

	// External dependencies section
	if len(cfg.ExternalDeps) > 0 {
		lines = append(lines, headerStyle.Render("EXTERNAL"))
		for _, extDep := range cfg.ExternalDeps {
			lines = append(lines, subtleStyle.Render("• "+extDep.URL))
		}
		lines = append(lines, "")
	}

	// Statistics summary (pinned to bottom of panel if possible)
	if linkStatus != nil {
		statsLine := fmt.Sprintf("Total: %d files", linkStatus.TotalCount)
		statsStyle := lipgloss.NewStyle().
			Foreground(ui.SubtleColor).
			Align(lipgloss.Right).
			Width(width - 4)

		// If we have space, push stats to the bottom
		currentHeight := lipgloss.Height(strings.Join(lines, "\n"))
		if height > currentHeight+2 {
			lines = append(lines, strings.Repeat("\n", height-currentHeight-2))
		}
		lines = append(lines, statsStyle.Render(statsLine))
	}

	return strings.Join(lines, "\n")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getSelectedConfigName returns the name of the currently selected config
func (m Model) getSelectedConfigName() string {
	if len(m.configs) > 0 && m.selectedIdx < len(m.configs) {
		return m.configs[m.selectedIdx].Name
	}
	return ""
}

// GetResult returns the action result after the model exits
func (m Model) GetResult() *Result {
	return m.result
}

// Run starts the dashboard and returns the selected action
func Run(p *platform.Platform, driftSummary *stow.DriftSummary, linkStatus map[string]*stow.ConfigLinkStatus, machineStatus []MachineStatus, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool, initialFilter string, initialSelected string) (*Result, error) {
	m := New(p, driftSummary, linkStatus, machineStatus, configs, dotfilesPath, updateMsg, hasBaseline, initialFilter, initialSelected)

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

// Init initializes the setup model
func (m SetupModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the setup model
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

// View renders the setup view
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
