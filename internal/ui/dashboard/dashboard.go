package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

type view int

const (
	viewDashboard view = iota
	viewMenu
	viewNoConfig
	viewOperation
)

// State holds all the shared data for the dashboard.
type State struct {
	Platform       *platform.Platform
	DriftSummary   *stow.DriftSummary
	LinkStatus     map[string]*stow.ConfigLinkStatus
	MachineStatus  []MachineStatus
	Configs        []config.ConfigItem
	DotfilesPath   string
	UpdateMsg      string
	HasBaseline    bool
	FilterText     string
	SelectedConfig string
	HasConfig      bool

	// Operation mode - start with an operation instead of dashboard view
	StartOperation OperationType
	OperationArg   string   // For single config operations
	OperationArgs  []string // For bulk operations
	AutoStart      bool     // Automatically start the operation
}

// Model is the main container for the dashboard.
type Model struct {
	state           State
	width           int
	height          int
	quitting        bool
	result          *Result
	filterMode      bool
	filterText      string
	selectedConfigs map[string]bool
	showHelp        bool
	currentView     view

	// Components
	header     Header
	summary    Summary
	sidebar    Sidebar
	details    Details
	footer     Footer
	help       Help
	menu       *Menu
	noconfig   NoConfig
	operations Operations
}

// New creates a new dashboard model.
func New(s State) Model {
	m := Model{
		state:           s,
		selectedConfigs: make(map[string]bool),
		filterText:      s.FilterText,
	}
	if s.SelectedConfig != "" {
		m.selectedConfigs[s.SelectedConfig] = true
	}

	// Determine initial view
	if s.AutoStart {
		m.currentView = viewOperation
		m.operations = NewOperations(s.StartOperation, s.OperationArg, s.OperationArgs)
	} else if !s.HasConfig {
		m.currentView = viewNoConfig
	} else {
		m.currentView = viewDashboard
	}

	m.header = NewHeader(s)
	m.summary = NewSummary(s)
	m.sidebar = NewSidebar(s, m.selectedConfigs)
	m.details = NewDetails(s)
	m.footer = NewFooter()
	m.help = NewHelp()
	m.menu = &Menu{}
	*m.menu = NewMenu()
	m.noconfig = NewNoConfig()
	return m
}

func (m Model) Init() tea.Cmd {
	if m.currentView == viewOperation {
		return m.operations.Init()
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.Help), key.Matches(msg, keys.Quit):
				m.showHelp = false
			}
		}
		return m, nil
	}

	switch m.currentView {
	case viewMenu:
		return m.updateMenu(msg)
	case viewNoConfig:
		return m.updateNoConfig(msg)
	case viewOperation:
		return m.updateOperation(msg)
	default:
		return m.updateDashboard(msg)
	}
}

func (m *Model) setResult(action Action, names ...string) {
	m.result = &Result{
		Action:     action,
		FilterText: m.filterText,
	}
	if len(names) > 0 {
		if action == ActionSyncConfig {
			m.result.ConfigName = names[0]
		} else {
			m.result.ConfigNames = names
		}
	}
	if len(m.selectedConfigs) == 1 {
		for name := range m.selectedConfigs {
			m.result.SelectedConfig = name
		}
	}
}

func (m *Model) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterMode {
			switch {
			case key.Matches(msg, keys.Quit): // esc
				m.filterMode = false
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				m.filterMode = false
			case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
				}
			default:
				m.filterText += msg.String()
			}
			m.updateFilter()
			m.details.selectedIdx = m.sidebar.selectedIdx // Ensure details are updated
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Help):
			m.showHelp = true
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			m.setResult(ActionQuit)
			return m, tea.Quit
		case key.Matches(msg, keys.Filter):
			m.filterMode = true
			return m, nil
		case key.Matches(msg, keys.Sync):
			m.setResult(ActionSync)
			return m, tea.Quit
		case key.Matches(msg, keys.Doctor):
			m.setResult(ActionDoctor)
			return m, tea.Quit
		case key.Matches(msg, keys.Install):
			m.setResult(ActionInstall)
			return m, tea.Quit
		case key.Matches(msg, keys.Machine):
			m.setResult(ActionMachineConfig)
			return m, tea.Quit
		case key.Matches(msg, keys.Update):
			m.setResult(ActionUpdate)
			return m, tea.Quit
		case key.Matches(msg, keys.Menu):
			m.currentView = viewMenu
		case key.Matches(msg, keys.Enter):
			if len(m.state.Configs) > 0 && m.sidebar.selectedIdx < len(m.state.Configs) {
				m.setResult(ActionSyncConfig, m.state.Configs[m.sidebar.selectedIdx].Name)
				return m, tea.Quit
			}
		case key.Matches(msg, keys.Select):
			if len(m.state.Configs) > 0 && m.sidebar.selectedIdx < len(m.state.Configs) {
				cfgName := m.state.Configs[m.sidebar.selectedIdx].Name
				if m.selectedConfigs[cfgName] {
					delete(m.selectedConfigs, cfgName)
				} else {
					m.selectedConfigs[cfgName] = true
				}
			}
		case key.Matches(msg, keys.All):
			allSelected := true
			for _, idx := range m.sidebar.filteredIdxs {
				if !m.selectedConfigs[m.state.Configs[idx].Name] {
					allSelected = false
					break
				}
			}
			if allSelected {
				for _, idx := range m.sidebar.filteredIdxs {
					delete(m.selectedConfigs, m.state.Configs[idx].Name)
				}
			} else {
				for _, idx := range m.sidebar.filteredIdxs {
					m.selectedConfigs[m.state.Configs[idx].Name] = true
				}
			}
		case key.Matches(msg, keys.Bulk):
			if len(m.selectedConfigs) > 0 {
				names := make([]string, 0, len(m.selectedConfigs))
				for name := range m.selectedConfigs {
					names = append(names, name)
				}
				m.setResult(ActionBulkSync, names...)
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.width = msg.Width / 3
		m.sidebar.height = msg.Height - 10 // placeholder
		m.details.width = msg.Width - m.sidebar.width
		m.details.height = m.sidebar.height
		m.footer.width = msg.Width
		m.help.width = msg.Width
		m.help.height = msg.Height
	}

	cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	m.details.selectedIdx = m.sidebar.selectedIdx

	return m, tea.Batch(cmds...)
}

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.currentView = viewDashboard
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			item, ok := m.menu.list.SelectedItem().(menuItem)
			if ok {
				m.setResult(item.action)
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu.(*Menu)
	return m, cmd
}

func (m *Model) updateNoConfig(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			m.setResult(ActionQuit)
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("i"), key.WithKeys("enter"))):
			m.setResult(ActionInit)
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) updateOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle operation-specific messages
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.operations.IsDone() {
			switch {
			case key.Matches(msg, keys.Quit):
				m.quitting = true
				m.setResult(ActionQuit)
				return m, tea.Quit
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				// Return to dashboard
				m.currentView = viewDashboard
				return m, nil
			}
		} else {
			// Allow cancellation during operation
			switch {
			case key.Matches(msg, keys.Quit):
				m.quitting = true
				m.setResult(ActionQuit)
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.operations.width = msg.Width
		m.operations.height = msg.Height

	case OperationProgressMsg, OperationStepCompleteMsg, OperationLogMsg, OperationDoneMsg:
		// Forward operation messages to operations component
		m.operations, cmd = m.operations.Update(msg)
		return m, cmd
	}

	// Update spinner
	m.operations, cmd = m.operations.Update(msg)
	return m, cmd
}

// StartOperation switches to operation view and starts an operation
func (m *Model) StartOperation(opType OperationType, configName string, configNames []string) tea.Cmd {
	m.operations = NewOperations(opType, configName, configNames)
	m.operations.width = m.width
	m.operations.height = m.height
	m.currentView = viewOperation
	return m.operations.Init()
}

func (m *Model) updateFilter() {
	filtered := []int{}
	if m.filterText == "" {
		for i := range m.state.Configs {
			filtered = append(filtered, i)
		}
	} else {
		for i, cfg := range m.state.Configs {
			if strings.Contains(strings.ToLower(cfg.Name), strings.ToLower(m.filterText)) {
				filtered = append(filtered, i)
			}
		}
	}
	m.sidebar.filteredIdxs = filtered
	m.sidebar.listOffset = 0
	if len(filtered) > 0 {
		m.sidebar.selectedIdx = filtered[0]
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.help.View())
	}

	switch m.currentView {
	case viewMenu:
		return m.menu.View()
	case viewNoConfig:
		return m.noconfig.View()
	case viewOperation:
		return m.viewOperation()
	default:
		return m.viewDashboard()
	}
}

func (m Model) viewDashboard() string {
	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.SubtleColor).
		Width(m.sidebar.width - 2).
		Height(m.sidebar.height).
		Render(m.sidebar.View())

	detailsView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Width(m.details.width-2).
		Height(m.details.height).
		Padding(0, 1).
		Render(m.details.View())

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarView,
		detailsView,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.header.View(),
		m.summary.View(),
		mainContent,
		m.footer.View(),
	)
}

func (m Model) viewOperation() string {
	// Ensure non-negative dimensions
	safeWidth := m.width - 4
	if safeWidth < 1 {
		safeWidth = 1
	}
	safeHeight := m.height - 4
	if safeHeight < 1 {
		safeHeight = 1
	}

	// Container with padding and border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Padding(1, 2).
		Width(safeWidth).
		Height(safeHeight)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		containerStyle.Render(m.operations.View()),
	)
}

// Result is returned when the dashboard exits
type Result struct {
	Action         Action
	ConfigName     string
	ConfigNames    []string
	FilterText     string
	SelectedConfig string
}

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

// MachineStatus represents the status of a machine config for the dashboard
type MachineStatus struct {
	ID          string
	Description string
	Status      string // "configured", "missing", "error"
}

// Run starts the dashboard and returns the selected action
func Run(s State) (*Result, error) {
	m := New(s)
	p := tea.NewProgram(&m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(*Model).result, nil
}

// RunWithOperation starts the dashboard in operation mode and executes the operation
// The operationFunc is called with the program to send progress updates
func RunWithOperation(s State, opType OperationType, configName string, configNames []string, operationFunc func(runner *OperationRunner) error) (*Result, error) {
	// Set up state for operation mode
	s.AutoStart = true
	s.StartOperation = opType
	s.OperationArg = configName
	s.OperationArgs = configNames

	m := New(s)
	p := tea.NewProgram(&m, tea.WithAltScreen())

	// Start the operation in a goroutine with panic recovery
	go func() {
		runner := NewOperationRunner(p)
		defer func() {
			if r := recover(); r != nil {
				runner.Done(false, "", fmt.Errorf("operation panicked: %v", r))
			}
		}()
		err := operationFunc(runner)
		if err != nil {
			runner.Done(false, "", err)
		}
	}()

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(*Model).result, nil
}
