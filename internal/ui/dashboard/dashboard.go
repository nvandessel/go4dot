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
	viewOnboarding
	viewConfirm
	viewConfigList
	viewExternal
	viewDoctor
	viewMachine
)

// State holds all the shared data for the dashboard.
type State struct {
	Platform       *platform.Platform
	DriftSummary   *stow.DriftSummary
	LinkStatus     map[string]*stow.ConfigLinkStatus
	MachineStatus  []MachineStatus
	Configs        []config.ConfigItem
	Config         *config.Config // Full config for operations
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
	viewStack       []view // Stack for navigation history
	operationActive bool   // true when an operation is running in the output pane
	program         *tea.Program // reference for inline operations

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
	output     OutputPane
	onboarding *Onboarding

	// Modal views
	confirm      *Confirm
	configList   *ConfigListView
	externalView *ExternalView
	doctorView   *DoctorView
	machineView  *MachineView
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
	m.output = NewOutputPane()
	return m
}

func (m Model) Init() tea.Cmd {
	if m.currentView == viewOperation {
		return m.operations.Init()
	}
	return nil
}

// pushView pushes the current view onto the stack and switches to a new view
func (m *Model) pushView(newView view) {
	m.viewStack = append(m.viewStack, m.currentView)
	m.currentView = newView
}

// popView returns to the previous view from the stack
// Returns true if there was a view to pop, false if stack was empty
func (m *Model) popView() bool {
	if len(m.viewStack) == 0 {
		return false
	}
	// Pop the last view from the stack
	lastIdx := len(m.viewStack) - 1
	m.currentView = m.viewStack[lastIdx]
	m.viewStack = m.viewStack[:lastIdx]
	return true
}

// clearViewStack clears the navigation stack
func (m *Model) clearViewStack() {
	m.viewStack = nil
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
	case viewOnboarding:
		return m.updateOnboarding(msg)
	case viewConfirm:
		return m.updateConfirm(msg)
	case viewConfigList:
		return m.updateConfigList(msg)
	case viewExternal:
		return m.updateExternal(msg)
	case viewDoctor:
		return m.updateDoctor(msg)
	case viewMachine:
		return m.updateMachine(msg)
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
			m.details.updateContent()
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
			if m.state.Config != nil && !m.operationActive {
				// Interactive: false because we can't run huh forms inside Bubble Tea
				opts := SyncOptions{Force: false, Interactive: false}
				cmd := m.StartInlineOperation(OpSync, "", nil, func(runner *OperationRunner) error {
					_, err := RunSyncAllOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
					return err
				})
				return m, cmd
			}
		case key.Matches(msg, keys.Doctor):
			if m.state.Config != nil {
				m.doctorView = NewDoctorView(m.state.Config, m.state.DotfilesPath)
				m.doctorView.SetSize(m.width, m.height)
				m.pushView(viewDoctor)
				return m, m.doctorView.Init()
			}
		case key.Matches(msg, keys.Install):
			if m.state.Config != nil && !m.operationActive {
				opts := InstallOptions{}
				cmd := m.StartInlineOperation(OpInstall, "", nil, func(runner *OperationRunner) error {
					_, err := RunInstallOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
					return err
				})
				return m, cmd
			}
		case key.Matches(msg, keys.Machine):
			if m.state.Config != nil && len(m.state.Config.MachineConfig) > 0 {
				m.machineView = NewMachineView(m.state.Config)
				m.machineView.SetSize(m.width, m.height)
				m.pushView(viewMachine)
				return m, m.machineView.Init()
			}
		case key.Matches(msg, keys.Update):
			if m.state.Config != nil && !m.operationActive {
				opts := UpdateOptions{UpdateExternal: true}
				cmd := m.StartInlineOperation(OpUpdate, "", nil, func(runner *OperationRunner) error {
					_, err := RunUpdateOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
					return err
				})
				return m, cmd
			}
		case key.Matches(msg, keys.Menu):
			m.menu.SetSize(m.width, m.height)
			m.pushView(viewMenu)
		case key.Matches(msg, keys.Enter):
			if len(m.state.Configs) > 0 && m.sidebar.selectedIdx < len(m.state.Configs) && m.state.Config != nil && !m.operationActive {
				configName := m.state.Configs[m.sidebar.selectedIdx].Name
				// Interactive: false because we can't run huh forms inside Bubble Tea
				opts := SyncOptions{Force: false, Interactive: false}
				cmd := m.StartInlineOperation(OpSyncSingle, configName, nil, func(runner *OperationRunner) error {
					_, err := RunSyncSingleOperation(runner, m.state.Config, m.state.DotfilesPath, configName, opts)
					return err
				})
				return m, cmd
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
			if len(m.selectedConfigs) > 0 && m.state.Config != nil && !m.operationActive {
				names := make([]string, 0, len(m.selectedConfigs))
				for name := range m.selectedConfigs {
					names = append(names, name)
				}
				// Interactive: false because we can't run huh forms inside Bubble Tea
				opts := SyncOptions{Force: false, Interactive: false}
				cmd := m.StartInlineOperation(OpBulkSync, "", names, func(runner *OperationRunner) error {
					_, err := RunBulkSyncOperation(runner, m.state.Config, m.state.DotfilesPath, names, opts)
					return err
				})
				return m, cmd
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Layout calculation:
		// Header: ~3 lines, Summary: ~2 lines, Footer: ~2 lines = ~7 fixed
		// Remaining height split: main content (2/3) and output pane (1/3)
		fixedHeight := 7
		availableHeight := msg.Height - fixedHeight
		if availableHeight < 6 {
			availableHeight = 6
		}

		// Output pane gets 1/3 of available height (min 4 lines)
		outputHeight := availableHeight / 3
		if outputHeight < 4 {
			outputHeight = 4
		}

		// Main content gets the rest
		mainHeight := availableHeight - outputHeight
		if mainHeight < 4 {
			mainHeight = 4
		}

		m.sidebar.width = msg.Width / 3
		m.sidebar.height = mainHeight
		detailsWidth := msg.Width - m.sidebar.width
		m.details.SetSize(detailsWidth, mainHeight)
		m.output.SetSize(msg.Width, outputHeight)
		m.footer.width = msg.Width
		m.help.width = msg.Width
		m.help.height = msg.Height

	case tea.MouseMsg:
		// Forward mouse events to components based on position
		// Sidebar is on the left (x < sidebar.width)
		// Details is on the right (x >= sidebar.width)
		if msg.X < m.sidebar.width {
			m.sidebar.Update(msg)
		} else {
			cmd = m.details.Update(msg)
			cmds = append(cmds, cmd)
		}

	// Handle operation messages for inline operations
	case OperationProgressMsg, OperationStepCompleteMsg, OperationLogMsg, OperationDoneMsg:
		handled, cmd := m.handleOperationMsg(msg)
		if handled {
			// Reset output title when operation completes (inline operations only)
			if _, ok := msg.(OperationDoneMsg); ok {
				m.output.SetTitle("Output")
			}
			return m, cmd
		}
	}

	cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	// Update details if selection changed
	if m.details.selectedIdx != m.sidebar.selectedIdx {
		m.details.selectedIdx = m.sidebar.selectedIdx
		m.details.updateContent()
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.popView()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			item, ok := m.menu.list.SelectedItem().(menuItem)
			if ok {
				return m.handleMenuAction(item.action)
			}
		}
	}

	var cmd tea.Cmd
	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu.(*Menu)
	return m, cmd
}

// handleMenuAction processes a menu selection inline when possible
func (m *Model) handleMenuAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionList:
		// Show config list view inline
		m.configList = NewConfigListView(m.state.Configs)
		m.configList.SetSize(m.width, m.height)
		m.pushView(viewConfigList)
		return m, nil

	case ActionExternal:
		// Show external dependencies view inline
		if m.state.Config == nil {
			return m, nil
		}
		m.externalView = NewExternalView(m.state.Config, m.state.DotfilesPath, m.state.Platform)
		m.externalView.SetSize(m.width, m.height)
		m.pushView(viewExternal)
		return m, m.externalView.Init()

	case ActionUninstall:
		// Show confirmation dialog
		m.confirm = NewConfirm(
			"uninstall",
			"Uninstall go4dot?",
			"This will remove all symlinks and state. This action cannot be undone.",
		).WithLabels("Yes, uninstall", "Cancel")
		m.confirm.SetSize(m.width, m.height)
		m.pushView(viewConfirm)
		return m, nil

	default:
		// Fall back to exiting for actions not yet handled inline
		m.setResult(action)
		return m, tea.Quit
	}
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
			// Start onboarding inline instead of exiting
			return m.startOnboarding()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// startOnboarding initializes and switches to the onboarding view
func (m *Model) startOnboarding() (tea.Model, tea.Cmd) {
	// Use current directory or dotfiles path
	path := "."
	if m.state.DotfilesPath != "" {
		path = m.state.DotfilesPath
	}

	onboarding := NewOnboarding(path)
	onboarding.width = m.width
	onboarding.height = m.height
	m.onboarding = &onboarding
	m.pushView(viewOnboarding)

	return m, m.onboarding.Init()
}

func (m *Model) updateOnboarding(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.onboarding != nil {
			m.onboarding.width = msg.Width
			m.onboarding.height = msg.Height
		}

	case OnboardingCompleteMsg:
		if msg.Error != nil {
			// Onboarding was cancelled or failed, return to previous view
			m.popView()
			m.onboarding = nil
			return m, nil
		}

		// Onboarding completed successfully - set result to reload with new config
		m.setResult(ActionInit)
		m.result.ConfigName = msg.ConfigPath
		return m, tea.Quit
	}

	if m.onboarding != nil {
		model, cmd := m.onboarding.Update(msg)
		if ob, ok := model.(*Onboarding); ok {
			m.onboarding = ob
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.confirm != nil {
			m.confirm.SetSize(msg.Width, msg.Height)
		}

	case ConfirmResult:
		if msg.ID == "uninstall" && msg.Confirmed {
			// User confirmed uninstall - return to main loop to execute
			m.setResult(ActionUninstall)
			return m, tea.Quit
		}
		// Cancel or other - return to previous view
		m.popView()
		m.confirm = nil
		return m, nil
	}

	if m.confirm != nil {
		model, cmd := m.confirm.Update(msg)
		if c, ok := model.(*Confirm); ok {
			m.confirm = c
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateConfigList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.configList != nil {
			m.configList.SetSize(msg.Width, msg.Height)
		}

	case ConfigListViewCloseMsg:
		m.popView()
		m.configList = nil
		return m, nil
	}

	if m.configList != nil {
		model, cmd := m.configList.Update(msg)
		if cl, ok := model.(*ConfigListView); ok {
			m.configList = cl
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateExternal(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.externalView != nil {
			m.externalView.SetSize(msg.Width, msg.Height)
		}

	case ExternalViewCloseMsg:
		m.popView()
		m.externalView = nil
		return m, nil
	}

	if m.externalView != nil {
		model, cmd := m.externalView.Update(msg)
		if ev, ok := model.(*ExternalView); ok {
			m.externalView = ev
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateDoctor(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.doctorView != nil {
			m.doctorView.SetSize(msg.Width, msg.Height)
		}

	case DoctorViewCloseMsg:
		m.popView()
		m.doctorView = nil
		return m, nil
	}

	if m.doctorView != nil {
		model, cmd := m.doctorView.Update(msg)
		if dv, ok := model.(*DoctorView); ok {
			m.doctorView = dv
		}
		return m, cmd
	}

	return m, nil
}

func (m *Model) updateMachine(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.machineView != nil {
			m.machineView.SetSize(msg.Width, msg.Height)
		}

	case MachineViewCloseMsg:
		m.popView()
		m.machineView = nil
		return m, nil

	case MachineConfigCompleteMsg:
		// TODO: Persist machine config values to file
		// For now, just return to previous view without saving
		m.popView()
		m.machineView = nil
		return m, nil
	}

	if m.machineView != nil {
		model, cmd := m.machineView.Update(msg)
		if mv, ok := model.(*MachineView); ok {
			m.machineView = mv
		}
		return m, cmd
	}

	return m, nil
}

// stepStatusToLogLevel converts a StepStatus to a log level string
func stepStatusToLogLevel(status StepStatus) string {
	switch status {
	case StepSuccess:
		return "success"
	case StepWarning:
		return "warning"
	case StepError:
		return "error"
	default:
		return "info"
	}
}

// handleOperationMsg processes operation-related messages and returns whether the message was handled
func (m *Model) handleOperationMsg(msg tea.Msg) (handled bool, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case OperationProgressMsg:
		m.operationActive = true
		m.operations, cmd = m.operations.Update(msg)
		return true, cmd

	case OperationStepCompleteMsg:
		m.operations, cmd = m.operations.Update(msg)
		if msg.Detail != "" {
			m.output.AddLog(stepStatusToLogLevel(msg.Status), msg.Detail)
		}
		return true, cmd

	case OperationLogMsg:
		m.operations, cmd = m.operations.Update(msg)
		m.output.AddLog(msg.Level, msg.Message)
		return true, cmd

	case OperationDoneMsg:
		m.operationActive = false
		m.operations, cmd = m.operations.Update(msg)
		if msg.Error != nil {
			m.output.AddLog("error", fmt.Sprintf("Operation failed: %v", msg.Error))
		} else if msg.Summary != "" {
			m.output.AddLog("success", msg.Summary)
		}
		return true, cmd
	}
	return false, nil
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
				// If launched via RunWithOperation (AutoStart), quit and return result
				if m.state.AutoStart {
					m.quitting = true
					m.setResult(ActionQuit)
					return m, tea.Quit
				}
				// Otherwise return to dashboard
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
		handled, cmd := m.handleOperationMsg(msg)
		if handled {
			return m, cmd
		}
	}

	// Update spinner
	m.operations, cmd = m.operations.Update(msg)
	return m, cmd
}

// StartInlineOperation runs an operation in the background without switching views
// Output is shown in the output pane at the bottom of the dashboard
func (m *Model) StartInlineOperation(opType OperationType, configName string, configNames []string, operationFunc func(runner *OperationRunner) error) tea.Cmd {
	if m.program == nil || m.operationActive {
		return nil
	}

	m.operationActive = true
	m.operations = NewOperations(opType, configName, configNames) // Reset operation state
	m.output.Clear()
	m.output.SetTitle(getOperationTitle(opType))

	// Run operation in goroutine
	go func() {
		runner := NewOperationRunner(m.program)
		defer func() {
			if r := recover(); r != nil {
				runner.Done(false, "", fmt.Errorf("operation panicked: %v", r))
			}
		}()
		err := operationFunc(runner)
		if err != nil {
			runner.Done(false, "", err)
		} else {
			runner.Done(true, "", nil)
		}
	}()

	return m.operations.Init()
}

// getOperationTitle returns a display title for an operation type
func getOperationTitle(opType OperationType) string {
	switch opType {
	case OpInstall:
		return "Installing"
	case OpSync:
		return "Syncing All"
	case OpSyncSingle:
		return "Syncing"
	case OpBulkSync:
		return "Syncing Selected"
	case OpUpdate:
		return "Updating"
	case OpDoctor:
		return "Health Check"
	default:
		return "Operation"
	}
}

// renderFilterBar renders the filter input bar
func (m Model) renderFilterBar() string {
	labelStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	inputStyle := lipgloss.NewStyle().Foreground(ui.TextColor)
	hintStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	label := labelStyle.Render("Filter: ")
	input := inputStyle.Render(m.filterText)

	// Show cursor when in filter mode
	cursor := ""
	if m.filterMode {
		cursor = lipgloss.NewStyle().
			Foreground(ui.PrimaryColor).
			Blink(true).
			Render("▌")
	}

	// Show hint
	hint := ""
	if m.filterText != "" && !m.filterMode {
		hint = hintStyle.Render("  (press / to edit)")
	} else if m.filterMode {
		hint = hintStyle.Render("  (enter/esc to finish)")
	}

	return label + input + cursor + hint
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
	case viewOnboarding:
		if m.onboarding != nil {
			return m.onboarding.View()
		}
		return ""
	case viewConfirm:
		if m.confirm != nil {
			return m.confirm.View()
		}
		return ""
	case viewConfigList:
		if m.configList != nil {
			return m.configList.View()
		}
		return ""
	case viewExternal:
		if m.externalView != nil {
			return m.externalView.View()
		}
		return ""
	case viewDoctor:
		if m.doctorView != nil {
			return m.doctorView.View()
		}
		return ""
	case viewMachine:
		if m.machineView != nil {
			return m.machineView.View()
		}
		return ""
	default:
		return m.viewDashboard()
	}
}

func (m Model) viewDashboard() string {
	// Build sidebar title with filter status
	sidebarTitle := "Configs"
	totalConfigs := len(m.state.Configs)
	filteredConfigs := len(m.sidebar.filteredIdxs)
	if m.filterText != "" {
		sidebarTitle = fmt.Sprintf("Configs (%d/%d)", filteredConfigs, totalConfigs)
	}

	// Sidebar with inline title
	sidebarView := renderPaneWithTitle(
		m.sidebar.View(),
		sidebarTitle,
		m.sidebar.width,
		m.sidebar.height,
		ui.SubtleColor,
	)

	// Details pane with inline title
	detailsView := renderPaneWithTitle(
		m.details.View(),
		"Details",
		m.details.width,
		m.details.height,
		ui.PrimaryColor,
	)

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarView,
		detailsView,
	)

	// Output pane with inline title
	outputTitle := "Output"
	if m.operationActive {
		outputTitle = "Output (running...)"
	}
	outputView := renderPaneWithTitle(
		m.output.View(),
		outputTitle,
		m.width,
		m.output.height,
		ui.SubtleColor,
	)

	// Build filter bar if in filter mode or filter is active
	filterBar := ""
	if m.filterMode || m.filterText != "" {
		filterBar = m.renderFilterBar()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.header.View(),
		m.summary.View(),
		filterBar,
		mainContent,
		outputView,
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
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p // Store program reference for inline operations

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
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())

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
		} else {
			runner.Done(true, "", nil)
		}
	}()

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(*Model).result, nil
}
