package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/ui"
)

// updateDashboard handles messages when in the main dashboard view
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
			contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.DefaultOverlayStyle())
			m.menu.SetSize(contentWidth, contentHeight)
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
		helpWidth, helpHeight := overlayContentSize(msg.Width, msg.Height, ui.HelpOverlayStyle())
		m.help.width = helpWidth
		m.help.height = helpHeight

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

	// Handle unconfigured machine configs detection
	case machineConfigsUnconfiguredMsg:
		desc := fmt.Sprintf("%d machine config(s) need setup. Configure now?", len(msg.missing))
		m.confirm = NewConfirm(
			"machine-setup-prompt",
			"Machine Settings Detected",
			desc,
		).WithLabels("Yes, set up", "Skip for now")
		m.confirm.selected = 0
		contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.ConfirmOverlayStyle())
		m.confirm.SetSize(contentWidth, contentHeight)
		m.pushView(viewConfirm)
		return m, nil

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

// handleFilterMode handles key input while in filter mode
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
		// Only append printable characters (single runes), ignore special keys
		keyStr := msg.String()
		if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] < 127 {
			m.filterText += keyStr
		}
	}
	m.configsPanel.SetFilter(m.filterText)
	m.updateDetailsContext()
	return m, nil
}

// handlePanelNavigation handles keyboard navigation between panels
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

// handlePanelActions handles action keys based on the currently focused panel
func (m *Model) handlePanelActions(msg tea.KeyMsg) tea.Cmd {
	focused := m.focusManager.CurrentFocus()

	switch {
	// Global operations (s, i, u)
	case key.Matches(msg, keys.Sync):
		if m.state.Config != nil && !m.operationActive {
			// Check for conflicts before syncing
			conflicts, err := CheckForConflicts(m.state.Config, m.state.DotfilesPath, nil)
			if err != nil {
				m.outputPanel.AddLog("error", fmt.Sprintf("Failed to check conflicts: %v", err))
				return nil
			}
			if len(conflicts) > 0 {
				// Show conflict resolution modal
				m.conflictView = NewConflictView(conflicts)
				contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.ConflictOverlayStyle())
				m.conflictView.SetSize(contentWidth, contentHeight)
				m.pendingOperation = OpSync
				m.pendingConflicts = conflicts
				m.pushView(viewConflict)
				return nil
			}
			// No conflicts, proceed normally
			opts := SyncOptions{Force: false, Interactive: false}
			return m.StartInlineOperation(OpSync, "", nil, func(runner *OperationRunner) error {
				_, err := RunSyncAllOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
				if err != nil {
					return fmt.Errorf("sync all: %w", err)
				}
				return nil
			})
		}

	case key.Matches(msg, keys.Install):
		if m.state.Config != nil && !m.operationActive {
			// Check for conflicts before installing
			conflicts, err := CheckForConflicts(m.state.Config, m.state.DotfilesPath, nil)
			if err != nil {
				m.outputPanel.AddLog("error", fmt.Sprintf("Failed to check conflicts: %v", err))
				return nil
			}
			if len(conflicts) > 0 {
				// Show conflict resolution modal
				m.conflictView = NewConflictView(conflicts)
				contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.ConflictOverlayStyle())
				m.conflictView.SetSize(contentWidth, contentHeight)
				m.pendingOperation = OpInstall
				m.pendingConflicts = conflicts
				m.pushView(viewConflict)
				return nil
			}
			// No conflicts, proceed normally
			opts := InstallOptions{}
			return m.StartInlineOperation(OpInstall, "", nil, func(runner *OperationRunner) error {
				_, err := RunInstallOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
				if err != nil {
					return fmt.Errorf("install: %w", err)
				}
				return nil
			})
		}

	case key.Matches(msg, keys.Update):
		if m.state.Config != nil && !m.operationActive {
			opts := UpdateOptions{UpdateExternal: true}
			return m.StartInlineOperation(OpUpdate, "", nil, func(runner *OperationRunner) error {
				_, err := RunUpdateOperation(runner, m.state.Config, m.state.DotfilesPath, opts)
				if err != nil {
					return fmt.Errorf("update: %w", err)
				}
				return nil
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
			m.summaryPanel.SetSelectedCount(len(m.selectedConfigs))
		}

	// Select All (A)
	case key.Matches(msg, keys.All):
		if focused == PanelConfigs {
			// Toggle select all: compare selection count to total config count
			totalConfigs := m.configsPanel.GetTotalCount()
			if len(m.selectedConfigs) == totalConfigs && totalConfigs > 0 {
				m.configsPanel.DeselectAll()
			} else {
				m.configsPanel.SelectAll()
			}
			m.selectedConfigs = m.configsPanel.GetSelected()
			m.summaryPanel.SetSelectedCount(len(m.selectedConfigs))
		}

	// Bulk sync (S)
	case key.Matches(msg, keys.Bulk):
		if len(m.selectedConfigs) > 0 && m.state.Config != nil && !m.operationActive {
			names := make([]string, 0, len(m.selectedConfigs))
			for name := range m.selectedConfigs {
				names = append(names, name)
			}
			// Check for conflicts for selected configs only
			conflicts, err := CheckForConflicts(m.state.Config, m.state.DotfilesPath, names)
			if err != nil {
				m.outputPanel.AddLog("error", fmt.Sprintf("Failed to check conflicts: %v", err))
				return nil
			}
			if len(conflicts) > 0 {
				// Show conflict resolution modal
				m.conflictView = NewConflictView(conflicts)
				contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.ConflictOverlayStyle())
				m.conflictView.SetSize(contentWidth, contentHeight)
				m.pendingOperation = OpBulkSync
				m.pendingConfigNames = names
				m.pendingConflicts = conflicts
				m.pushView(viewConflict)
				return nil
			}
			// No conflicts, proceed normally
			opts := SyncOptions{Force: false, Interactive: false}
			return m.StartInlineOperation(OpBulkSync, "", names, func(runner *OperationRunner) error {
				_, err := RunBulkSyncOperation(runner, m.state.Config, m.state.DotfilesPath, names, opts)
				if err != nil {
					return fmt.Errorf("bulk sync: %w", err)
				}
				return nil
			})
		}
	}

	return nil
}

// handleEnterAction handles the Enter key based on the currently focused panel
func (m *Model) handleEnterAction(focused PanelID) tea.Cmd {
	switch focused {
	case PanelConfigs:
		// Sync selected config
		cfg := m.configsPanel.GetSelectedConfig()
		if cfg != nil && m.state.Config != nil && !m.operationActive {
			// Check for conflicts for this specific config
			conflicts, err := CheckForConflicts(m.state.Config, m.state.DotfilesPath, []string{cfg.Name})
			if err != nil {
				m.outputPanel.AddLog("error", fmt.Sprintf("Failed to check conflicts: %v", err))
				return nil
			}
			if len(conflicts) > 0 {
				// Show conflict resolution modal
				m.conflictView = NewConflictView(conflicts)
				contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.ConflictOverlayStyle())
				m.conflictView.SetSize(contentWidth, contentHeight)
				m.pendingOperation = OpSyncSingle
				m.pendingConfigName = cfg.Name
				m.pendingConflicts = conflicts
				m.pushView(viewConflict)
				return nil
			}
			// No conflicts, proceed normally
			opts := SyncOptions{Force: false, Interactive: false}
			return m.StartInlineOperation(OpSyncSingle, cfg.Name, nil, func(runner *OperationRunner) error {
				_, err := RunSyncSingleOperation(runner, m.state.Config, m.state.DotfilesPath, cfg.Name, opts)
				if err != nil {
					return fmt.Errorf("sync %s: %w", cfg.Name, err)
				}
				return nil
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
			contentWidth, contentHeight := overlayContentSize(m.width, m.height, ui.DefaultOverlayStyle())
			m.machineView.SetSize(contentWidth, contentHeight)
			m.pushView(viewMachine)
			return m.machineView.Init()
		}

	case PanelExternal:
		// Clone/update external dep
		ext := m.externalPanel.GetSelectedExternal()
		if ext != nil && m.state.Config != nil && !m.operationActive {
			extID := ext.Dep.ID
			// If already installed, update; if missing, clone
			shouldUpdate := ext.Status == "installed"
			opts := ExternalSingleOptions{Update: shouldUpdate}
			return m.StartInlineOperation(OpExternalSingle, ext.Dep.Name, nil, func(runner *OperationRunner) error {
				_, err := RunExternalSingleOperation(runner, m.state.Config, m.state.DotfilesPath, extID, opts)
				if err != nil {
					return fmt.Errorf("external %s: %w", ext.Dep.Name, err)
				}
				return nil
			})
		}
	}

	return nil
}

// changeFocus changes the currently focused panel
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

// updateDetailsContext updates the details panel based on the current focus
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
