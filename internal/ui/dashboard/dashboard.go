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
	viewConflict
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
	viewStack       []view        // Stack for navigation history
	operationActive bool          // true when an operation is running in the output pane
	program         *tea.Program  // reference for inline operations

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
	conflictView *ConflictView

	// Post-onboarding state
	pendingNewConfigPath string
	pendingNewConfig     *config.Config

	// Pending operation state (for conflict resolution)
	pendingOperation   OperationType
	pendingConfigName  string
	pendingConfigNames []string
	pendingConflicts   []stow.ConflictFile
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
	m.footer.SetPlatform(s.Platform)
	m.footer.SetUpdateMsg(s.UpdateMsg)
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
	case viewConflict:
		return m.updateConflict(msg)
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
		opType := m.operations.OperationType()
		m.operations, cmd = m.operations.Update(msg)
		if msg.Error != nil {
			m.outputPanel.AddLog("error", fmt.Sprintf("Operation failed: %v", msg.Error))
		} else if msg.Summary != "" {
			m.outputPanel.AddLog("success", msg.Summary)
		}
		// Refresh appropriate panel after operation completes
		var refreshCmd tea.Cmd
		if opType == OpExternalSingle && msg.Error == nil {
			refreshCmd = m.externalPanel.Refresh()
		}
		return true, tea.Batch(cmd, refreshCmd)
	}
	return false, nil
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
	case OpExternalSingle:
		return "External"
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
	case viewConflict:
		if m.conflictView != nil {
			return m.conflictView.View()
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

	// Panels flush at top, footer at bottom (with optional filter bar above it)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		mainContent,
		filterBar,
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
