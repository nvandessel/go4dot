package dashboard

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// updateNoConfig handles messages when no config file is found
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

// startOnboarding initializes and starts the onboarding wizard
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

// updateOnboarding handles messages during the onboarding flow
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

		// Store the new config for potential install
		m.pendingNewConfigPath = msg.ConfigPath
		m.pendingNewConfig = msg.Config

		// Show in-TUI confirmation dialog instead of quitting
		m.onboarding = nil
		m.confirm = NewConfirm(
			"post-onboarding-install",
			"Configuration created!",
			"Would you like to run install now? This will set up dependencies and clone external repos.",
		).WithLabels("Yes, install", "Skip for now")
		m.confirm.selected = 0 // Default to "Yes, install"
		m.confirm.SetSize(m.width, m.height)

		// Switch from onboarding view to confirm view
		m.popView()          // Remove onboarding from stack
		m.pushView(viewConfirm)
		return m, nil
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

// updateConfirm handles messages for the confirmation dialog
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

		if msg.ID == "post-onboarding-install" {
			m.popView()
			m.confirm = nil

			if msg.Confirmed && m.pendingNewConfig != nil && m.pendingNewConfigPath != "" {
				// User chose to install - set up dashboard first, then check conflicts
				dotfilesPath := filepath.Dir(m.pendingNewConfigPath)

				// Update the model state with the new config
				m.state.Config = m.pendingNewConfig
				m.state.DotfilesPath = dotfilesPath
				m.state.HasConfig = true
				m.state.Configs = m.pendingNewConfig.GetAllConfigs()

				// Reinitialize all panels with the new config
				m.reinitializePanels()

				// Clear pending onboarding state
				m.pendingNewConfig = nil
				m.pendingNewConfigPath = ""

				// Switch to dashboard view
				m.clearViewStack()
				m.currentView = viewDashboard

				// Initialize panels
				var initCmds []tea.Cmd
				initCmds = append(initCmds, m.healthPanel.Init())
				initCmds = append(initCmds, m.externalPanel.Init())

				// Check for conflicts before installing
				conflicts, err := CheckForConflicts(m.state.Config, m.state.DotfilesPath, nil)
				if err != nil {
					m.outputPanel.AddLog("error", fmt.Sprintf("Failed to check conflicts: %v", err))
					return m, tea.Batch(initCmds...)
				}

				if len(conflicts) > 0 {
					// Show conflict resolution modal
					m.conflictView = NewConflictView(conflicts)
					m.conflictView.SetSize(m.width, m.height)
					m.pendingOperation = OpInstall
					m.pendingConflicts = conflicts
					m.pushView(viewConflict)
					return m, tea.Batch(initCmds...)
				}

				// No conflicts, proceed with install
				opts := InstallOptions{}
				installCmd := m.StartInlineOperation(OpInstall, "", nil, func(runner *OperationRunner) error {
					_, err := RunInstallOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
					return err
				})
				if installCmd != nil {
					initCmds = append(initCmds, installCmd)
				}

				return m, tea.Batch(initCmds...)
			}

			// User declined install - return to dashboard with new config loaded
			if m.pendingNewConfigPath != "" {
				dotfilesPath := filepath.Dir(m.pendingNewConfigPath)

				// Update the model state with the new config if available
				if m.pendingNewConfig != nil {
					m.state.Config = m.pendingNewConfig
					m.state.DotfilesPath = dotfilesPath
					m.state.HasConfig = true
					m.state.Configs = m.pendingNewConfig.GetAllConfigs()

					// Reinitialize all panels with the new config
					m.reinitializePanels()
				}
			}

			// Clear pending state
			m.pendingNewConfig = nil
			m.pendingNewConfigPath = ""

			// Switch to dashboard view
			m.clearViewStack()
			m.currentView = viewDashboard

			// Initialize panels
			return m, tea.Batch(m.healthPanel.Init(), m.externalPanel.Init())
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

// updateConfigList handles messages for the config list view
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

// updateExternal handles messages for the external dependencies view
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

// updateMachine handles messages for the machine configuration view
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

// updateConflict handles messages for the conflict resolution view
func (m *Model) updateConflict(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.conflictView != nil {
			m.conflictView.SetSize(msg.Width, msg.Height)
		}

	case ConflictResolvedMsg:
		m.popView()
		m.conflictView = nil

		if !msg.Resolved {
			// User cancelled or error occurred
			if msg.Error != nil {
				m.outputPanel.AddLog("error", fmt.Sprintf("Failed to resolve conflicts: %v", msg.Error))
			} else {
				m.outputPanel.AddLog("info", "Operation cancelled")
			}
			// Clear pending operation state
			m.pendingOperation = 0
			m.pendingConfigName = ""
			m.pendingConfigNames = nil
			m.pendingConflicts = nil
			return m, nil
		}

		// Conflicts resolved, execute the pending operation
		m.outputPanel.AddLog("success", fmt.Sprintf("Resolved %d conflict(s)", len(m.pendingConflicts)))
		return m.executePendingOperation()
	}

	if m.conflictView != nil {
		model, cmd := m.conflictView.Update(msg)
		if cv, ok := model.(*ConflictView); ok {
			m.conflictView = cv
		}
		return m, cmd
	}

	return m, nil
}

// updateOperation handles messages during operation execution
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

// executePendingOperation executes the operation that was pending conflict resolution
func (m *Model) executePendingOperation() (tea.Model, tea.Cmd) {
	opType := m.pendingOperation
	configName := m.pendingConfigName
	configNames := m.pendingConfigNames

	// Clear pending state
	m.pendingOperation = 0
	m.pendingConfigName = ""
	m.pendingConfigNames = nil
	m.pendingConflicts = nil

	// Execute the operation based on type
	switch opType {
	case OpSync:
		opts := SyncOptions{Force: false, Interactive: false}
		return m, m.StartInlineOperation(OpSync, "", nil, func(runner *OperationRunner) error {
			_, err := RunSyncAllOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
			return err
		})

	case OpInstall:
		opts := InstallOptions{}
		return m, m.StartInlineOperation(OpInstall, "", nil, func(runner *OperationRunner) error {
			_, err := RunInstallOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
			return err
		})

	case OpSyncSingle:
		opts := SyncOptions{Force: false, Interactive: false}
		return m, m.StartInlineOperation(OpSyncSingle, configName, nil, func(runner *OperationRunner) error {
			_, err := RunSyncSingleOperation(runner, m.state.Config, m.state.DotfilesPath, configName, opts)
			return err
		})

	case OpBulkSync:
		opts := SyncOptions{Force: false, Interactive: false}
		return m, m.StartInlineOperation(OpBulkSync, "", configNames, func(runner *OperationRunner) error {
			_, err := RunBulkSyncOperation(runner, m.state.Config, m.state.DotfilesPath, configNames, opts)
			return err
		})
	}

	return m, nil
}

// reinitializePanels recreates all panels with the current state
func (m *Model) reinitializePanels() {
	m.summaryPanel = NewSummaryPanel(m.state)
	m.configsPanel = NewConfigsPanel(m.state, m.selectedConfigs)
	m.healthPanel = NewHealthPanel(m.state.Config, m.state.DotfilesPath)
	m.overridesPanel = NewOverridesPanel(m.state.Config)
	m.externalPanel = NewExternalPanel(m.state.Config, m.state.DotfilesPath, m.state.Platform)
	m.detailsPanel = NewDetailsPanel(m.state)
	m.detailsPanel.SetPanels(m.configsPanel, m.healthPanel, m.overridesPanel, m.externalPanel)
	m.outputPanel = NewOutputPanel()

	// Re-register all panels
	m.panels[PanelSummary] = m.summaryPanel
	m.panels[PanelConfigs] = m.configsPanel
	m.panels[PanelHealth] = m.healthPanel
	m.panels[PanelOverrides] = m.overridesPanel
	m.panels[PanelExternal] = m.externalPanel
	m.panels[PanelDetails] = m.detailsPanel
	m.panels[PanelOutput] = m.outputPanel

	// Apply layout to set panel dimensions
	m.layout.Calculate(m.width, m.height)
	m.layout.ApplyToPanels(m.panels)

	// Set initial focus
	m.configsPanel.SetFocused(true)
}
