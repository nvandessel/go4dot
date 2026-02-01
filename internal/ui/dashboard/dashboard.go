package dashboard

import (
	"fmt"

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

	// Multi-panel layout
	focusManager *FocusManager
	layout       *Layout
	panels       map[PanelID]Panel

	// Panel references for easy access
	summaryPanel   *SummaryPanel
	healthPanel    *HealthPanel
	overridesPanel *OverridesPanel
	externalPanel  *ExternalPanel
	configsPanel   *ConfigsPanel
	detailsPanel   *DetailsPanel
	outputPanel    *OutputPanel

	// Components (kept for backward compatibility)
	header     Header
	footer     Footer
	help       Help
	menu       *Menu
	noconfig   NoConfig
	operations Operations
	onboarding *Onboarding


	// Modal views
	confirm      *Confirm
	configList   *ConfigListView
	externalView *ExternalView
	machineView  *MachineView
}

// New creates a new dashboard model.
func New(s State) Model {
	m := Model{
		state:           s,
		selectedConfigs: make(map[string]bool),
		filterText:      s.FilterText,
		focusManager:    NewFocusManager(),
		layout:          NewLayout(),
		panels:          make(map[PanelID]Panel),
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

	// Initialize multi-panel components
	m.summaryPanel = NewSummaryPanel(s)
	m.healthPanel = NewHealthPanel(s.Config, s.DotfilesPath)
	m.overridesPanel = NewOverridesPanel(s.Config)
	m.externalPanel = NewExternalPanel(s.Config, s.DotfilesPath, s.Platform)
	m.configsPanel = NewConfigsPanel(s, m.selectedConfigs)
	m.detailsPanel = NewDetailsPanel(s)
	m.outputPanel = NewOutputPanel()

	// Set up panel references for details panel
	m.detailsPanel.SetPanels(m.configsPanel, m.healthPanel, m.overridesPanel, m.externalPanel)

	// Register panels
	m.panels[PanelSummary] = m.summaryPanel
	m.panels[PanelHealth] = m.healthPanel
	m.panels[PanelOverrides] = m.overridesPanel
	m.panels[PanelExternal] = m.externalPanel
	m.panels[PanelConfigs] = m.configsPanel
	m.panels[PanelDetails] = m.detailsPanel
	m.panels[PanelOutput] = m.outputPanel

	// Set initial focus
	m.configsPanel.SetFocused(true)

	// Initialize other components
	m.header = NewHeader(s)
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

	// Start async loading for Health and External panels
	var cmds []tea.Cmd
	if m.currentView == viewDashboard {
		cmds = append(cmds, m.healthPanel.Init())
		cmds = append(cmds, m.externalPanel.Init())
	}

	return tea.Batch(cmds...)
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
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterMode {
			return m.handleFilterMode(msg)
		}

		// Handle global keys first
		switch {
		case key.Matches(msg, keys.Help):
			m.showHelp = true
			return m, nil
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			m.setResult(ActionQuit)
			return m, tea.Quit
		case key.Matches(msg, keys.Filter):
			m.filterMode = true
			return m, nil
		case key.Matches(msg, keys.Menu):
			m.menu.SetSize(m.width, m.height)
			m.pushView(viewMenu)
			return m, nil
		}

		// Handle panel navigation
		if cmd := m.handlePanelNavigation(msg); cmd != nil {
			return m, cmd
		}

		// Handle actions based on focused panel
		if cmd := m.handlePanelActions(msg); cmd != nil {
			return m, cmd
		}

		// Forward to focused panel
		focused := m.focusManager.CurrentFocus()
		if panel, ok := m.panels[focused]; ok {
			cmd := panel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

		// Update details panel context when focus changes
		m.updateDetailsContext()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate layout
		m.layout.Calculate(msg.Width, msg.Height)

		// Apply layout to panels
		m.layout.ApplyToPanels(m.panels)

		// Update other components
		m.footer.width = msg.Width
		m.help.width = msg.Width
		m.help.height = msg.Height

	case tea.MouseMsg:
		// Handle mouse for focused panel
		focused := m.focusManager.CurrentFocus()
		if panel, ok := m.panels[focused]; ok {
			cmd := panel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		m.updateDetailsContext()

	// Handle async panel updates
	case healthResultMsg:
		cmd := m.healthPanel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case externalStatusMsg:
		cmd := m.externalPanel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	// Handle operation messages for inline operations
	case OperationProgressMsg, OperationStepCompleteMsg, OperationLogMsg, OperationDoneMsg:
		handled, cmd := m.handleOperationMsg(msg)
		if handled {
			if _, ok := msg.(OperationDoneMsg); ok {
				m.outputPanel.SetTitle("Output")
			}
			return m, cmd
		}
	}

	// Forward spinner tick to loading panels
	if _, ok := msg.(interface{ Tag() int }); ok {
		if m.healthPanel.IsLoading() {
			cmd := m.healthPanel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if m.externalPanel.IsLoading() {
			cmd := m.externalPanel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	m.configsPanel.SetFilter(m.filterText)
	m.updateDetailsContext()
	return m, nil
}

func (m *Model) handlePanelNavigation(msg tea.KeyMsg) tea.Cmd {
	oldFocus := m.focusManager.CurrentFocus()

	switch {
	// Tab cycles through panels
	case key.Matches(msg, keys.PanelNext):
		m.focusManager.CycleNext()
	case key.Matches(msg, keys.PanelPrev):
		m.focusManager.CyclePrev()

	// Directional navigation (Ctrl+hjkl)
	case key.Matches(msg, keys.PanelLeft):
		m.focusManager.MoveLeft()
	case key.Matches(msg, keys.PanelRight):
		m.focusManager.MoveRight()
	case key.Matches(msg, keys.PanelUp):
		m.focusManager.MoveUp()
	case key.Matches(msg, keys.PanelDown):
		m.focusManager.MoveDown()

	// Direct panel jump (0-6): 0=Output, 1=Summary, 2=Health, etc.
	case key.Matches(msg, keys.Panel0):
		m.focusManager.JumpToPanel(0)
	case key.Matches(msg, keys.Panel1):
		m.focusManager.JumpToPanel(1)
	case key.Matches(msg, keys.Panel2):
		m.focusManager.JumpToPanel(2)
	case key.Matches(msg, keys.Panel3):
		m.focusManager.JumpToPanel(3)
	case key.Matches(msg, keys.Panel4):
		m.focusManager.JumpToPanel(4)
	case key.Matches(msg, keys.Panel5):
		m.focusManager.JumpToPanel(5)
	case key.Matches(msg, keys.Panel6):
		m.focusManager.JumpToPanel(6)

	default:
		return nil
	}

	newFocus := m.focusManager.CurrentFocus()
	if oldFocus != newFocus {
		// Update focus state on panels
		if oldPanel, ok := m.panels[oldFocus]; ok {
			oldPanel.SetFocused(false)
		}
		if newPanel, ok := m.panels[newFocus]; ok {
			newPanel.SetFocused(true)
		}
		m.footer.SetFocusedPanel(newFocus)
		m.updateDetailsContext()
	}

	return nil
}

func (m *Model) handlePanelActions(msg tea.KeyMsg) tea.Cmd {
	focused := m.focusManager.CurrentFocus()

	switch {
	// Global operations (s, i, u)
	case key.Matches(msg, keys.Sync):
		if m.state.Config != nil && !m.operationActive {
			opts := SyncOptions{Force: false, Interactive: false}
			return m.StartInlineOperation(OpSync, "", nil, func(runner *OperationRunner) error {
				_, err := RunSyncAllOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
				return err
			})
		}

	case key.Matches(msg, keys.Install):
		if m.state.Config != nil && !m.operationActive {
			opts := InstallOptions{}
			return m.StartInlineOperation(OpInstall, "", nil, func(runner *OperationRunner) error {
				_, err := RunInstallOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
				return err
			})
		}

	case key.Matches(msg, keys.Update):
		if m.state.Config != nil && !m.operationActive {
			opts := UpdateOptions{UpdateExternal: true}
			return m.StartInlineOperation(OpUpdate, "", nil, func(runner *OperationRunner) error {
				_, err := RunUpdateOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
				return err
			})
		}

	// Doctor (d) - now just focuses Health panel if not already
	case key.Matches(msg, keys.Doctor):
		if focused != PanelHealth {
			m.changeFocus(PanelHealth)
		}
		return nil

	// Machine (m) - now focuses Overrides panel
	case key.Matches(msg, keys.Machine):
		if focused != PanelOverrides {
			m.changeFocus(PanelOverrides)
		}
		return nil

	// Enter - context-specific action
	case key.Matches(msg, keys.Enter):
		return m.handleEnterAction(focused)

	// Select (space) - only for Configs panel
	case key.Matches(msg, keys.Select):
		if focused == PanelConfigs {
			m.configsPanel.ToggleSelection()
			m.selectedConfigs = m.configsPanel.GetSelected()
		}

	// Select All (A)
	case key.Matches(msg, keys.All):
		if focused == PanelConfigs {
			// Toggle select all
			allSelected := true
			for _, name := range m.configsPanel.GetSelectedNames() {
				if !m.selectedConfigs[name] {
					allSelected = false
					break
				}
			}
			if allSelected && len(m.selectedConfigs) > 0 {
				m.configsPanel.DeselectAll()
			} else {
				m.configsPanel.SelectAll()
			}
			m.selectedConfigs = m.configsPanel.GetSelected()
		}

	// Bulk sync (S)
	case key.Matches(msg, keys.Bulk):
		if len(m.selectedConfigs) > 0 && m.state.Config != nil && !m.operationActive {
			names := make([]string, 0, len(m.selectedConfigs))
			for name := range m.selectedConfigs {
				names = append(names, name)
			}
			opts := SyncOptions{Force: false, Interactive: false}
			return m.StartInlineOperation(OpBulkSync, "", names, func(runner *OperationRunner) error {
				_, err := RunBulkSyncOperation(runner, m.state.Config, m.state.DotfilesPath, names, opts)
				return err
			})
		}
	}

	return nil
}

func (m *Model) handleEnterAction(focused PanelID) tea.Cmd {
	switch focused {
	case PanelConfigs:
		// Sync selected config
		cfg := m.configsPanel.GetSelectedConfig()
		if cfg != nil && m.state.Config != nil && !m.operationActive {
			opts := SyncOptions{Force: false, Interactive: false}
			return m.StartInlineOperation(OpSyncSingle, cfg.Name, nil, func(runner *OperationRunner) error {
				_, err := RunSyncSingleOperation(runner, m.state.Config, m.state.DotfilesPath, cfg.Name, opts)
				return err
			})
		}

	case PanelHealth:
		// Re-run health checks
		return m.healthPanel.Refresh()

	case PanelOverrides:
		// Open machine config form (modal)
		mc := m.overridesPanel.GetSelectedConfig()
		if mc != nil && m.state.Config != nil {
			m.machineView = NewMachineView(m.state.Config)
			m.machineView.SetSize(m.width, m.height)
			m.pushView(viewMachine)
			return m.machineView.Init()
		}

	case PanelExternal:
		// Clone/update external dep
		ext := m.externalPanel.GetSelectedExternal()
		if ext != nil && m.state.Config != nil && !m.operationActive {
			// TODO: Implement clone/update operation
			m.outputPanel.AddLog("info", fmt.Sprintf("Would clone/update: %s", ext.Dep.Name))
		}
	}

	return nil
}

func (m *Model) changeFocus(newFocus PanelID) {
	oldFocus := m.focusManager.CurrentFocus()
	if oldFocus == newFocus {
		return
	}

	if oldPanel, ok := m.panels[oldFocus]; ok {
		oldPanel.SetFocused(false)
	}
	m.focusManager.SetFocus(newFocus)
	if newPanel, ok := m.panels[newFocus]; ok {
		newPanel.SetFocused(true)
	}
	m.footer.SetFocusedPanel(newFocus)
	m.updateDetailsContext()
}

func (m *Model) updateDetailsContext() {
	focused := m.focusManager.CurrentFocus()

	switch focused {
	case PanelHealth:
		m.detailsPanel.SetContext(DetailsContextHealth)
	case PanelOverrides:
		m.detailsPanel.SetContext(DetailsContextOverrides)
	case PanelExternal:
		m.detailsPanel.SetContext(DetailsContextExternal)
	default:
		m.detailsPanel.SetContext(DetailsContextConfigs)
	}
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

func (m *Model) handleMenuAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionList:
		m.configList = NewConfigListView(m.state.Configs)
		m.configList.SetSize(m.width, m.height)
		m.pushView(viewConfigList)
		return m, nil

	case ActionExternal:
		if m.state.Config == nil {
			return m, nil
		}
		m.externalView = NewExternalView(m.state.Config, m.state.DotfilesPath, m.state.Platform)
		m.externalView.SetSize(m.width, m.height)
		m.pushView(viewExternal)
		return m, m.externalView.Init()

	case ActionUninstall:
		m.confirm = NewConfirm(
			"uninstall",
			"Uninstall go4dot?",
			"This will remove all symlinks and state. This action cannot be undone.",
		).WithLabels("Yes, uninstall", "Cancel")
		m.confirm.SetSize(m.width, m.height)
		m.pushView(viewConfirm)
		return m, nil

	default:
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
			return m.startOnboarding()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *Model) startOnboarding() (tea.Model, tea.Cmd) {
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
			m.popView()
			m.onboarding = nil
			return m, nil
		}

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
			m.setResult(ActionUninstall)
			return m, tea.Quit
		}
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
		m.popView()
		m.machineView = nil
		m.overridesPanel.RefreshStatus()
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

func (m *Model) handleOperationMsg(msg tea.Msg) (handled bool, cmd tea.Cmd) {
	switch msg := msg.(type) {
	case OperationProgressMsg:
		m.operationActive = true
		m.operations, cmd = m.operations.Update(msg)
		return true, cmd

	case OperationStepCompleteMsg:
		m.operations, cmd = m.operations.Update(msg)
		if msg.Detail != "" {
			m.outputPanel.AddLog(stepStatusToLogLevel(msg.Status), msg.Detail)
		}
		return true, cmd

	case OperationLogMsg:
		m.operations, cmd = m.operations.Update(msg)
		m.outputPanel.AddLog(msg.Level, msg.Message)
		return true, cmd

	case OperationDoneMsg:
		m.operationActive = false
		m.operations, cmd = m.operations.Update(msg)
		if msg.Error != nil {
			m.outputPanel.AddLog("error", fmt.Sprintf("Operation failed: %v", msg.Error))
		} else if msg.Summary != "" {
			m.outputPanel.AddLog("success", msg.Summary)
		}
		return true, cmd
	}
	return false, nil
}

func (m *Model) updateOperation(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.operations.IsDone() {
			switch {
			case key.Matches(msg, keys.Quit):
				m.quitting = true
				m.setResult(ActionQuit)
				return m, tea.Quit
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				if m.state.AutoStart {
					m.quitting = true
					m.setResult(ActionQuit)
					return m, tea.Quit
				}
				m.currentView = viewDashboard
				return m, nil
			}
		} else {
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

	m.operations, cmd = m.operations.Update(msg)
	return m, cmd
}

func (m *Model) StartInlineOperation(opType OperationType, configName string, configNames []string, operationFunc func(runner *OperationRunner) error) tea.Cmd {
	if m.program == nil || m.operationActive {
		return nil
	}

	m.operationActive = true
	m.operations = NewOperations(opType, configName, configNames)
	m.outputPanel.Clear()
	m.outputPanel.SetTitle(getOperationTitle(opType))

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

func (m Model) renderFilterBar() string {
	labelStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	inputStyle := lipgloss.NewStyle().Foreground(ui.TextColor)
	hintStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	label := labelStyle.Render("Filter: ")
	input := inputStyle.Render(m.filterText)

	cursor := ""
	if m.filterMode {
		cursor = lipgloss.NewStyle().
			Foreground(ui.PrimaryColor).
			Blink(true).
			Render("▌")
	}

	hint := ""
	if m.filterText != "" && !m.filterMode {
		hint = hintStyle.Render("  (press / to edit)")
	} else if m.filterMode {
		hint = hintStyle.Render("  (enter/esc to finish)")
	}

	return label + input + cursor + hint
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
	focused := m.focusManager.CurrentFocus()

	// Render mini-column panels (left side, stacked)
	summaryView := RenderPanelFrameCompact(
		m.summaryPanel.View(),
		m.summaryPanel.GetTitle(),
		m.layout.Summary.Width,
		m.layout.Summary.Height,
		focused == PanelSummary,
	)

	healthView := RenderPanelFrameCompact(
		m.healthPanel.View(),
		m.healthPanel.GetTitle(),
		m.layout.Health.Width,
		m.layout.Health.Height,
		focused == PanelHealth,
	)

	overridesView := RenderPanelFrameCompact(
		m.overridesPanel.View(),
		m.overridesPanel.GetTitle(),
		m.layout.Overrides.Width,
		m.layout.Overrides.Height,
		focused == PanelOverrides,
	)

	externalView := RenderPanelFrameCompact(
		m.externalPanel.View(),
		m.externalPanel.GetTitle(),
		m.layout.External.Width,
		m.layout.External.Height,
		focused == PanelExternal,
	)

	miniColumn := lipgloss.JoinVertical(
		lipgloss.Left,
		summaryView,
		healthView,
		overridesView,
		externalView,
	)

	// Build configs panel title with filter status
	configsTitle := "5 Configs"
	totalConfigs := m.configsPanel.GetTotalCount()
	filteredConfigs := m.configsPanel.GetFilteredCount()
	if m.filterText != "" {
		configsTitle = fmt.Sprintf("5 Configs (%d/%d)", filteredConfigs, totalConfigs)
	}

	// Render main panels
	configsView := RenderPanelFrame(
		m.configsPanel.View(),
		configsTitle,
		m.layout.Configs.Width,
		m.layout.Configs.Height,
		focused == PanelConfigs,
	)

	detailsView := RenderPanelFrame(
		m.detailsPanel.View(),
		m.detailsPanel.GetTitle(),
		m.layout.Details.Width,
		m.layout.Details.Height,
		focused == PanelDetails,
	)

	// Output panel title
	outputTitle := "0 Output"
	if m.operationActive {
		outputTitle = "0 Output (running...)"
	}

	outputView := RenderPanelFrame(
		m.outputPanel.View(),
		outputTitle,
		m.layout.Output.Width,
		m.layout.Output.Height,
		focused == PanelOutput,
	)

	// Combine panels horizontally
	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		miniColumn,
		configsView,
		detailsView,
		outputView,
	)

	// Build filter bar if in filter mode or filter is active
	filterBar := ""
	if m.filterMode || m.filterText != "" {
		filterBar = m.renderFilterBar()
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.header.View(),
		filterBar,
		mainContent,
		m.footer.View(),
	)
}

func (m Model) viewOperation() string {
	safeWidth := m.width - 4
	if safeWidth < 1 {
		safeWidth = 1
	}
	safeHeight := m.height - 4
	if safeHeight < 1 {
		safeHeight = 1
	}

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
	m.program = p

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(*Model).result, nil
}

// RunWithOperation starts the dashboard in operation mode and executes the operation
func RunWithOperation(s State, opType OperationType, configName string, configNames []string, operationFunc func(runner *OperationRunner) error) (*Result, error) {
	s.AutoStart = true
	s.StartOperation = opType
	s.OperationArg = configName
	s.OperationArgs = configNames

	m := New(s)
	p := tea.NewProgram(&m, tea.WithAltScreen(), tea.WithMouseCellMotion())

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
