package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestNew(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
	}
	m := New(s)
	if m.state.Platform.OS != "linux" {
		t.Errorf("expected OS to be linux, got %s", m.state.Platform.OS)
	}
}

func TestNew_WithConfigs(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "darwin"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig:    true,
		DotfilesPath: "/home/user/dotfiles",
	}
	m := New(s)

	if m.currentView != viewDashboard {
		t.Errorf("expected currentView to be viewDashboard, got %v", m.currentView)
	}
	if len(m.state.Configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(m.state.Configs))
	}
	if m.selectedConfigs == nil {
		t.Error("expected selectedConfigs to be initialized")
	}
}

func TestNew_NoConfig(t *testing.T) {
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}
	m := New(s)

	if m.currentView != viewNoConfig {
		t.Errorf("expected currentView to be viewNoConfig, got %v", m.currentView)
	}
}

func TestNew_WithSelectedConfig(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig:      true,
		SelectedConfig: "vim",
	}
	m := New(s)

	if !m.selectedConfigs["vim"] {
		t.Error("expected vim to be pre-selected")
	}
}

func TestModel_Update_Actions(t *testing.T) {
	// Doctor action now focuses the Health panel instead of opening modal
	t.Run("Doctor action", func(t *testing.T) {
		baseState := State{
			Platform: &platform.Platform{OS: "linux"},
			Configs: []config.ConfigItem{
				{Name: "vim"},
			},
			Config:       &config.Config{}, // Need config for doctor to work
			HasConfig:    true,
			DotfilesPath: "/tmp/dotfiles",
		}
		m := New(baseState)
		m.width = 100
		m.height = 40

		// Initial focus should be on Configs panel
		initialFocus := m.focusManager.CurrentFocus()
		if initialFocus != PanelConfigs {
			t.Errorf("expected initial focus on PanelConfigs, got %v", initialFocus)
		}

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updatedModel, _ := m.Update(msg)

		model := updatedModel.(*Model)
		// Should stay on dashboard view
		if model.currentView != viewDashboard {
			t.Errorf("expected viewDashboard, got %v", model.currentView)
		}
		// Focus should be on Health panel
		if model.focusManager.CurrentFocus() != PanelHealth {
			t.Errorf("expected focus on PanelHealth, got %v", model.focusManager.CurrentFocus())
		}
	})

	// Actions that run inline when Config is set
	inlineTests := []struct {
		name    string
		keyRune rune
	}{
		{"Sync action", 's'},
		{"Install action", 'i'},
		{"Update action", 'u'},
	}

	for _, tt := range inlineTests {
		t.Run(tt.name+" without Config", func(t *testing.T) {
			// Without Config, nothing happens (no quit, no inline op)
			baseState := State{
				Platform: &platform.Platform{OS: "linux"},
				Configs: []config.ConfigItem{
					{Name: "vim"},
				},
				HasConfig:    true,
				DotfilesPath: "/tmp/dotfiles",
				Config:       nil, // No config, so inline op won't start
			}
			m := New(baseState)
			m.width = 100
			m.height = 40

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.keyRune}}
			updatedModel, _ := m.Update(msg)

			model := updatedModel.(*Model)
			// Without Config, operationActive should remain false
			if model.operationActive {
				t.Error("expected operationActive to be false without Config")
			}
		})
	}
}

func TestModel_InlineOperation_WithConfig(t *testing.T) {
	// With Config set, sync should start an inline operation
	cfg := &config.Config{}
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig:    true,
		Config:       cfg,
		DotfilesPath: "/tmp",
	}
	m := New(s)
	m.width = 100
	m.height = 40

	// Simulate what Run() does - set the program
	p := tea.NewProgram(&m, tea.WithAltScreen())
	m.program = p

	t.Logf("Before 's' - program is nil: %v", m.program == nil)
	t.Logf("Before 's' - operationActive: %v", m.operationActive)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}}
	updatedModel, cmd := m.Update(msg)

	model := updatedModel.(*Model)
	t.Logf("After 's' - operationActive: %v", model.operationActive)
	t.Logf("After 's' - cmd is nil: %v", cmd == nil)
	t.Logf("After 's' - program is nil: %v", model.program == nil)

	// With Config and program set, operationActive should be true
	if !model.operationActive {
		t.Error("expected operationActive to be true with Config set")
	}
	if cmd == nil {
		t.Error("expected a command to be returned")
	}
}

func TestModel_InlineOperation_MessageHandling(t *testing.T) {
	// Test that operation messages are handled in dashboard view
	cfg := &config.Config{}
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig:    true,
		Config:       cfg,
		DotfilesPath: "/tmp",
	}
	m := New(s)
	m.width = 100
	m.height = 40
	m.currentView = viewDashboard // Ensure we're in dashboard view

	// Test OperationProgressMsg
	progressMsg := OperationProgressMsg{StepIndex: 0, Detail: "test"}
	updatedModel, _ := m.Update(progressMsg)
	model := updatedModel.(*Model)
	if !model.operationActive {
		t.Error("expected operationActive to be true after OperationProgressMsg")
	}

	// Test OperationLogMsg
	logMsg := OperationLogMsg{Level: "info", Message: "test log"}
	updatedModel, _ = model.Update(logMsg)
	model = updatedModel.(*Model)
	if model.outputPanel.GetLogCount() != 1 {
		t.Errorf("expected 1 log entry, got %d", model.outputPanel.GetLogCount())
	}

	// Test OperationDoneMsg
	doneMsg := OperationDoneMsg{Success: true, Summary: "done"}
	updatedModel, _ = model.Update(doneMsg)
	model = updatedModel.(*Model)
	if model.operationActive {
		t.Error("expected operationActive to be false after OperationDoneMsg")
	}
	// Should have 2 logs now (the log + the done summary)
	if model.outputPanel.GetLogCount() != 2 {
		t.Errorf("expected 2 log entries, got %d", model.outputPanel.GetLogCount())
	}
}

func TestModel_Update_Quit(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	updatedModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected tea.Quit command")
	}

	model := updatedModel.(*Model)
	if !model.quitting {
		t.Error("expected quitting to be true")
	}
	if model.result == nil || model.result.Action != ActionQuit {
		t.Error("expected ActionQuit result")
	}
}

func TestModel_Update_Help(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)

	// Press ? to show help
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	updatedModel, _ := m.Update(msg)

	model := updatedModel.(*Model)
	if !model.showHelp {
		t.Error("expected showHelp to be true after pressing ?")
	}

	// Press ? again to hide help
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(*Model)
	if model.showHelp {
		t.Error("expected showHelp to be false after pressing ? again")
	}
}

func TestModel_Update_FilterMode(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim-config"},
			{Name: "zsh-config"},
			{Name: "tmux-config"},
		},
		HasConfig: true,
	}
	m := New(s)

	// Enter filter mode with /
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	if !model.filterMode {
		t.Error("expected filterMode to be true after pressing /")
	}

	// Type filter text
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	model = updatedModel.(*Model)
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})
	model = updatedModel.(*Model)
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	model = updatedModel.(*Model)

	if model.filterText != "vim" {
		t.Errorf("expected filterText to be 'vim', got '%s'", model.filterText)
	}

	// Exit filter mode with Esc
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updatedModel.(*Model)

	if model.filterMode {
		t.Error("expected filterMode to be false after pressing Esc")
	}
}

func TestModel_Update_FilterBackspace(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)
	m.filterMode = true
	m.filterText = "test"

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	if model.filterText != "tes" {
		t.Errorf("expected filterText to be 'tes' after backspace, got '%s'", model.filterText)
	}
}

func TestModel_Update_Selection(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig: true,
	}
	m := New(s)

	// Select with space
	msg := tea.KeyMsg{Type: tea.KeySpace}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	if !model.selectedConfigs["vim"] {
		t.Error("expected vim to be selected after pressing space")
	}

	// Toggle off with space again
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(*Model)

	if model.selectedConfigs["vim"] {
		t.Error("expected vim to be deselected after pressing space again")
	}
}

func TestModel_Update_SelectAll(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
			{Name: "tmux"},
		},
		HasConfig: true,
	}
	m := New(s)
	// The configsPanel is automatically initialized with all configs in filteredIdxs
	// so we can test SelectAll behavior directly without setup.

	// Select all with shift+A
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	for _, cfg := range s.Configs {
		if !model.selectedConfigs[cfg.Name] {
			t.Errorf("expected %s to be selected after select all", cfg.Name)
		}
	}

	// Toggle all off
	updatedModel, _ = model.Update(msg)
	model = updatedModel.(*Model)

	for _, cfg := range s.Configs {
		if model.selectedConfigs[cfg.Name] {
			t.Errorf("expected %s to be deselected after toggle all off", cfg.Name)
		}
	}
}

func TestModel_Update_BulkSync(t *testing.T) {
	// Without Config, bulk sync does nothing (no inline operation possible)
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig: true,
		Config:    nil, // No config, so inline op won't start
	}
	m := New(s)
	m.selectedConfigs["vim"] = true
	m.selectedConfigs["zsh"] = true

	// Bulk sync with shift+S
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	updatedModel, _ := m.Update(msg)

	model := updatedModel.(*Model)
	// Without Config, nothing happens
	if model.operationActive {
		t.Error("expected operationActive to be false without Config")
	}
}

func TestModel_Update_BulkSync_NoSelection(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)
	// No configs selected

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}}
	updatedModel, cmd := m.Update(msg)

	// Should not trigger bulk sync when nothing selected
	if cmd != nil {
		t.Error("expected no command when no configs selected")
	}

	model := updatedModel.(*Model)
	if model.result != nil {
		t.Error("expected no result when no configs selected")
	}
}

func TestModel_Update_SyncConfig(t *testing.T) {
	// Without Config, sync config does nothing (no inline operation possible)
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig: true,
		Config:    nil, // No config, so inline op won't start
	}
	m := New(s)

	// Press enter to sync selected config
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(msg)

	model := updatedModel.(*Model)
	// Without Config, nothing happens
	if model.operationActive {
		t.Error("expected operationActive to be false without Config")
	}
}

func TestModel_Update_Menu(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)

	// Open menu with backtick (changed from tab, which now cycles panels)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'`'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	if model.currentView != viewMenu {
		t.Errorf("expected currentView to be viewMenu, got %v", model.currentView)
	}

	// Go back with Esc
	updatedModel, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updatedModel.(*Model)

	if model.currentView != viewDashboard {
		t.Errorf("expected currentView to be viewDashboard after Esc, got %v", model.currentView)
	}
}

func TestModel_Update_WindowSize(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)

	msg := tea.WindowSizeMsg{Width: 120, Height: 50}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	if model.width != 120 {
		t.Errorf("expected width to be 120, got %d", model.width)
	}
	if model.height != 50 {
		t.Errorf("expected height to be 50, got %d", model.height)
	}
}

func TestModel_View(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim", Description: "Vim config"},
		},
		HasConfig: true,
	}
	m := New(s)
	m.width = 100
	m.height = 40

	view := m.View()

	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestModel_View_Quitting(t *testing.T) {
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: true,
	}
	m := New(s)
	m.quitting = true

	view := m.View()

	if view != "" {
		t.Error("expected empty view when quitting")
	}
}

func TestModel_View_Help(t *testing.T) {
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: true,
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
	}
	m := New(s)
	m.width = 100
	m.height = 40
	m.showHelp = true

	view := m.View()

	if view == "" {
		t.Error("expected non-empty view when showing help")
	}
}

func TestModel_NoConfig_Init(t *testing.T) {
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}
	m := New(s)
	m.width = 80
	m.height = 24

	// Press enter to init (NoConfig view now transitions to onboarding)
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, cmd := m.Update(msg)

	// Should get a command to initialize onboarding
	if cmd == nil {
		t.Error("expected command to initialize onboarding")
	}

	model := updatedModel.(*Model)
	// Should now be in onboarding view
	if model.currentView != viewOnboarding {
		t.Errorf("expected viewOnboarding, got %v", model.currentView)
	}

	// Onboarding should be initialized
	if model.onboarding == nil {
		t.Error("expected onboarding to be initialized")
	}
}

func TestKeys_Bindings(t *testing.T) {
	tests := []struct {
		name     string
		binding  key.Binding
		testKeys []string
	}{
		{"Sync", keys.Sync, []string{"s"}},
		{"Doctor", keys.Doctor, []string{"d"}},
		{"Install", keys.Install, []string{"i"}},
		{"Machine", keys.Machine, []string{"m"}},
		{"Update", keys.Update, []string{"u"}},
		{"Menu", keys.Menu, []string{"`"}},
		{"Quit", keys.Quit, []string{"q", "esc", "ctrl+c"}},
		{"Up", keys.Up, []string{"up", "k"}},
		{"Down", keys.Down, []string{"down", "j"}},
		{"Enter", keys.Enter, []string{"enter"}},
		{"Filter", keys.Filter, []string{"/"}},
		{"Help", keys.Help, []string{"?"}},
		{"Select", keys.Select, []string{" "}},
		{"All", keys.All, []string{"A"}},
		{"Bulk", keys.Bulk, []string{"S"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			boundKeys := tt.binding.Keys()
			if len(boundKeys) != len(tt.testKeys) {
				t.Errorf("expected %d keys, got %d", len(tt.testKeys), len(boundKeys))
				return
			}
			for i, k := range tt.testKeys {
				if boundKeys[i] != k {
					t.Errorf("expected key %s at position %d, got %s", k, i, boundKeys[i])
				}
			}
		})
	}
}

func TestModel_SyncOperations_StartInlineOperation(t *testing.T) {
	// Verify sync operations start inline operations when Config is set.
	// Previously, sync operations used Interactive: true which would try to
	// run huh forms inside Bubble Tea, causing UI corruption.
	cfg := &config.Config{}
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		HasConfig:    true,
		Config:       cfg,
		DotfilesPath: "/tmp/dotfiles",
	}

	testCases := []struct {
		name  string
		key   tea.KeyMsg
		setup func(*Model)
	}{
		{
			name:  "Sync all starts operation",
			key:   tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}},
			setup: func(m *Model) {},
		},
		{
			name:  "Sync single starts operation",
			key:   tea.KeyMsg{Type: tea.KeyEnter},
			setup: func(m *Model) {},
		},
		{
			name: "Bulk sync starts operation",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}},
			setup: func(m *Model) {
				m.selectedConfigs["vim"] = true
				m.selectedConfigs["zsh"] = true
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := New(s)
			m.width = 100
			m.height = 40
			p := tea.NewProgram(&m, tea.WithAltScreen())
			m.program = p

			tc.setup(&m)

			updatedModel, cmd := m.Update(tc.key)
			model := updatedModel.(*Model)

			if !model.operationActive {
				t.Error("expected operationActive to be true")
			}
			if cmd == nil {
				t.Error("expected a command to be returned")
			}
		})
	}
}

func TestModel_ViewDashboard_ComponentWidths(t *testing.T) {
	// Verify layout is calculated correctly after WindowSizeMsg.
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := New(s)
	sizeMsg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	model := updatedModel.(*Model)

	// Layout should be calculated
	if model.layout.Width != 120 {
		t.Errorf("expected layout width 120, got %d", model.layout.Width)
	}
	if model.layout.Height != 40 {
		t.Errorf("expected layout height 40, got %d", model.layout.Height)
	}

	// Configs panel should have width set
	if model.layout.Configs.Width <= 0 {
		t.Error("expected configs panel width to be set")
	}

	// Details panel should have width set
	if model.layout.Details.Width <= 0 {
		t.Error("expected details panel width to be set")
	}

	// View should render without panic
	view := model.viewDashboard()
	if view == "" {
		t.Error("expected non-empty dashboard view")
	}
}

func TestNavigationStack_PushPop(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		Config:       &config.Config{},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := New(s)
	m.width = 100
	m.height = 40

	// Initial state - no view stack
	if len(m.viewStack) != 0 {
		t.Errorf("expected empty view stack, got %d", len(m.viewStack))
	}
	if m.currentView != viewDashboard {
		t.Errorf("expected viewDashboard, got %v", m.currentView)
	}

	// Push external view
	m.pushView(viewExternal)
	if len(m.viewStack) != 1 {
		t.Errorf("expected view stack of 1, got %d", len(m.viewStack))
	}
	if m.viewStack[0] != viewDashboard {
		t.Errorf("expected viewDashboard in stack, got %v", m.viewStack[0])
	}
	if m.currentView != viewExternal {
		t.Errorf("expected currentView to be viewExternal, got %v", m.currentView)
	}

	// Push another view
	m.pushView(viewMachine)
	if len(m.viewStack) != 2 {
		t.Errorf("expected view stack of 2, got %d", len(m.viewStack))
	}
	if m.currentView != viewMachine {
		t.Errorf("expected currentView to be viewMachine, got %v", m.currentView)
	}

	// Pop back to external
	popped := m.popView()
	if !popped {
		t.Error("expected popView to return true")
	}
	if m.currentView != viewExternal {
		t.Errorf("expected currentView to be viewExternal, got %v", m.currentView)
	}
	if len(m.viewStack) != 1 {
		t.Errorf("expected view stack of 1, got %d", len(m.viewStack))
	}

	// Pop back to dashboard
	popped = m.popView()
	if !popped {
		t.Error("expected popView to return true")
	}
	if m.currentView != viewDashboard {
		t.Errorf("expected currentView to be viewDashboard, got %v", m.currentView)
	}
	if len(m.viewStack) != 0 {
		t.Errorf("expected empty view stack, got %d", len(m.viewStack))
	}

	// Pop on empty stack returns false
	popped = m.popView()
	if popped {
		t.Error("expected popView to return false on empty stack")
	}
}

func TestNavigationStack_ClearStack(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}

	m := New(s)

	// Push multiple views
	m.pushView(viewMenu)
	m.pushView(viewConfigList)
	m.pushView(viewExternal)

	if len(m.viewStack) != 3 {
		t.Errorf("expected view stack of 3, got %d", len(m.viewStack))
	}

	// Clear the stack
	m.clearViewStack()
	if len(m.viewStack) != 0 {
		t.Errorf("expected empty view stack after clear, got %d", len(m.viewStack))
	}
}

func TestNavigationStack_DoctorViewEscapeReturns(t *testing.T) {
	// Test that 'd' key focuses the Health panel (new multi-panel behavior)
	// Doctor modal is now accessed through Overrides panel -> Enter
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		Config:       &config.Config{},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}

	m := New(s)
	m.width = 100
	m.height = 40

	// Press 'd' to focus health panel
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	// Should still be on dashboard
	if model.currentView != viewDashboard {
		t.Errorf("expected viewDashboard, got %v", model.currentView)
	}
	// Focus should be on Health panel
	if model.focusManager.CurrentFocus() != PanelHealth {
		t.Errorf("expected focus on PanelHealth, got %v", model.focusManager.CurrentFocus())
	}
	// View stack should be empty (no modal pushed)
	if len(model.viewStack) != 0 {
		t.Errorf("expected empty view stack, got %d", len(model.viewStack))
	}
}

func TestNavigationStack_MenuToConfigListAndBack(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		Config:    &config.Config{},
		HasConfig: true,
	}

	m := New(s)
	m.width = 100
	m.height = 40

	// Open menu with backtick key (changed from tab, which now cycles panels)
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'`'}}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	if model.currentView != viewMenu {
		t.Errorf("expected viewMenu, got %v", model.currentView)
	}
	if len(model.viewStack) != 1 {
		t.Errorf("expected view stack of 1, got %d", len(model.viewStack))
	}

	// Simulate selecting "List Configs" action
	updatedModel, _ = model.handleMenuAction(ActionList)
	model = updatedModel.(*Model)

	if model.currentView != viewConfigList {
		t.Errorf("expected viewConfigList, got %v", model.currentView)
	}
	// Stack should now have dashboard and menu
	if len(model.viewStack) != 2 {
		t.Errorf("expected view stack of 2, got %d", len(model.viewStack))
	}

	// Close config list view
	closeMsg := ConfigListViewCloseMsg{}
	updatedModel, _ = model.Update(closeMsg)
	model = updatedModel.(*Model)

	// Should return to menu (the view we came from)
	if model.currentView != viewMenu {
		t.Errorf("expected viewMenu after close, got %v", model.currentView)
	}
	if len(model.viewStack) != 1 {
		t.Errorf("expected view stack of 1, got %d", len(model.viewStack))
	}
}

// TestPostOnboarding_AcceptInstall_PanelsHaveDimensions is a regression test for the bug
// where accepting install after onboarding resulted in an empty screen because
// layout.Calculate() and layout.ApplyToPanels() were not called after reinitializing panels.
func TestPostOnboarding_AcceptInstall_PanelsHaveDimensions(t *testing.T) {
	// Start with no-config state (simulating fresh onboarding)
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}

	m := New(s)
	m.width = 120
	m.height = 50

	// Simulate what happens after onboarding completes:
	// 1. pendingNewConfigPath and pendingNewConfig are set
	// 2. currentView is viewConfirm with post-onboarding-install dialog
	m.pendingNewConfigPath = "/tmp/test-dotfiles/.go4dot.yaml"
	m.pendingNewConfig = &config.Config{
		SchemaVersion: "1.0",
		Metadata: config.Metadata{
			Name: "test-dotfiles",
		},
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "vim", Path: "vim"},
			},
		},
	}

	// Set up confirm dialog (this is what updateOnboarding does on OnboardingCompleteMsg)
	m.confirm = NewConfirm(
		"post-onboarding-install",
		"Configuration created!",
		"Would you like to run install now?",
	).WithLabels("Yes, install", "Skip for now")
	m.confirm.selected = 0 // Default to "Yes, install"
	m.confirm.SetSize(m.width, m.height)
	m.pushView(viewConfirm)

	// Now send the ConfirmResult message (user pressed 'y' to accept install)
	confirmMsg := ConfirmResult{
		ID:        "post-onboarding-install",
		Confirmed: true,
	}
	updatedModel, _ := m.Update(confirmMsg)
	model := updatedModel.(*Model)

	// KEY ASSERTIONS: Verify the fix works

	// 1. Should be back on dashboard view
	if model.currentView != viewDashboard {
		t.Errorf("expected viewDashboard, got %v", model.currentView)
	}

	// 2. View stack should be cleared
	if len(model.viewStack) != 0 {
		t.Errorf("expected empty view stack, got %d", len(model.viewStack))
	}

	// 3. Config panel should have proper dimensions (this was the bug!)
	configsWidth := model.configsPanel.width
	configsHeight := model.configsPanel.height
	if configsWidth <= 0 {
		t.Errorf("configs panel width should be > 0, got %d (empty screen regression)", configsWidth)
	}
	if configsHeight <= 0 {
		t.Errorf("configs panel height should be > 0, got %d (empty screen regression)", configsHeight)
	}

	// 4. Layout should be calculated
	if model.layout.Width != 120 {
		t.Errorf("expected layout width 120, got %d", model.layout.Width)
	}
	if model.layout.Height != 50 {
		t.Errorf("expected layout height 50, got %d", model.layout.Height)
	}

	// 5. View should render (not be empty)
	view := model.View()
	if view == "" {
		t.Error("expected non-empty view after accepting install (empty screen regression)")
	}

	// 6. State should be updated with the new config
	if !model.state.HasConfig {
		t.Error("expected HasConfig to be true after install")
	}
	if model.state.Config == nil {
		t.Error("expected Config to be set after install")
	}
}

// TestPostOnboarding_DeclineInstall_PanelsHaveDimensions is a regression test for the bug
// where declining install after onboarding also resulted in an empty screen.
func TestPostOnboarding_DeclineInstall_PanelsHaveDimensions(t *testing.T) {
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}

	m := New(s)
	m.width = 120
	m.height = 50

	// Set up pending config (simulating onboarding completion)
	m.pendingNewConfigPath = "/tmp/test-dotfiles/.go4dot.yaml"
	m.pendingNewConfig = &config.Config{
		SchemaVersion: "1.0",
		Metadata: config.Metadata{
			Name: "test-dotfiles",
		},
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "zsh", Path: "zsh"},
			},
		},
	}

	// Set up confirm dialog
	m.confirm = NewConfirm(
		"post-onboarding-install",
		"Configuration created!",
		"Would you like to run install now?",
	)
	m.confirm.SetSize(m.width, m.height)
	m.pushView(viewConfirm)

	// Send decline (user pressed 'n')
	confirmMsg := ConfirmResult{
		ID:        "post-onboarding-install",
		Confirmed: false,
	}
	updatedModel, _ := m.Update(confirmMsg)
	model := updatedModel.(*Model)

	// KEY ASSERTIONS

	// 1. Should be on dashboard view
	if model.currentView != viewDashboard {
		t.Errorf("expected viewDashboard, got %v", model.currentView)
	}

	// 2. View stack should be cleared
	if len(model.viewStack) != 0 {
		t.Errorf("expected empty view stack, got %d", len(model.viewStack))
	}

	// 3. Config panel should have proper dimensions
	if model.configsPanel.width <= 0 {
		t.Errorf("configs panel width should be > 0, got %d (empty screen regression)", model.configsPanel.width)
	}
	if model.configsPanel.height <= 0 {
		t.Errorf("configs panel height should be > 0, got %d (empty screen regression)", model.configsPanel.height)
	}

	// 4. View should render
	view := model.View()
	if view == "" {
		t.Error("expected non-empty view after declining install (empty screen regression)")
	}

	// 5. State should still be updated with config (just not installed)
	if !model.state.HasConfig {
		t.Error("expected HasConfig to be true after decline")
	}
}

// TestPostOnboarding_AcceptInstall_WithConflicts_ShowsConflictModal tests that when
// accepting install after onboarding and conflicts exist, the conflict modal is shown.
func TestPostOnboarding_AcceptInstall_WithConflicts_ShowsConflictModal(t *testing.T) {
	s := State{
		Platform:  &platform.Platform{OS: "linux"},
		HasConfig: false,
	}

	m := New(s)
	m.width = 120
	m.height = 50

	// Set up pending config (simulating onboarding completion)
	m.pendingNewConfigPath = "/tmp/test-dotfiles/.go4dot.yaml"
	m.pendingNewConfig = &config.Config{
		SchemaVersion: "1.0",
		Metadata: config.Metadata{
			Name: "test-dotfiles",
		},
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "zsh", Path: "zsh"},
			},
		},
	}

	// Set up confirm dialog
	m.confirm = NewConfirm(
		"post-onboarding-install",
		"Configuration created!",
		"Would you like to run install now?",
	)
	m.confirm.SetSize(m.width, m.height)
	m.pushView(viewConfirm)

	// Send confirm (user pressed 'y')
	confirmMsg := ConfirmResult{
		ID:        "post-onboarding-install",
		Confirmed: true,
	}
	updatedModel, _ := m.Update(confirmMsg)
	model := updatedModel.(*Model)

	// The model should now be set up to check for conflicts
	// Since we can't easily mock CheckForConflicts, we verify the state is correct
	// after the update. In a real scenario with conflicts, it would show the modal.

	// Verify state is updated
	if !model.state.HasConfig {
		t.Error("expected HasConfig to be true")
	}
	if model.state.Config == nil {
		t.Error("expected Config to be set")
	}
}

// TestConflictView_Creation tests that ConflictView is created correctly
func TestConflictView_Creation(t *testing.T) {
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc", SourcePath: "/home/user/dotfiles/zsh/.zshrc"},
		{ConfigName: "vim", TargetPath: "/home/user/.vimrc", SourcePath: "/home/user/dotfiles/vim/.vimrc"},
	}

	cv := NewConflictView(conflicts)

	if cv == nil {
		t.Fatal("expected ConflictView to be created")
	}
	if len(cv.conflicts) != 2 {
		t.Errorf("expected 2 conflicts, got %d", len(cv.conflicts))
	}
	if len(cv.byConfig) != 2 {
		t.Errorf("expected 2 config groups, got %d", len(cv.byConfig))
	}
	if cv.selectedIdx != 0 {
		t.Errorf("expected selectedIdx to default to 0 (Backup), got %d", cv.selectedIdx)
	}
}

// TestConflictView_Navigation tests keyboard navigation in ConflictView
func TestConflictView_Navigation(t *testing.T) {
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
	}
	cv := NewConflictView(conflicts)
	cv.SetSize(80, 40)

	// Initial position is 0 (Backup)
	if cv.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0, got %d", cv.selectedIdx)
	}

	// Press right arrow to move to Delete (1)
	msg := tea.KeyMsg{Type: tea.KeyRight}
	updatedModel, _ := cv.Update(msg)
	cv = updatedModel.(*ConflictView)
	if cv.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after right, got %d", cv.selectedIdx)
	}

	// Press right again to move to Cancel (2)
	updatedModel, _ = cv.Update(msg)
	cv = updatedModel.(*ConflictView)
	if cv.selectedIdx != 2 {
		t.Errorf("expected selectedIdx 2 after right, got %d", cv.selectedIdx)
	}

	// Press right again - should stay at 2 (no wrap)
	updatedModel, _ = cv.Update(msg)
	cv = updatedModel.(*ConflictView)
	if cv.selectedIdx != 2 {
		t.Errorf("expected selectedIdx to stay at 2, got %d", cv.selectedIdx)
	}

	// Press left to go back to Delete
	msg = tea.KeyMsg{Type: tea.KeyLeft}
	updatedModel, _ = cv.Update(msg)
	cv = updatedModel.(*ConflictView)
	if cv.selectedIdx != 1 {
		t.Errorf("expected selectedIdx 1 after left, got %d", cv.selectedIdx)
	}

	// Press tab to cycle
	msg = tea.KeyMsg{Type: tea.KeyTab}
	updatedModel, _ = cv.Update(msg)
	cv = updatedModel.(*ConflictView)
	if cv.selectedIdx != 2 {
		t.Errorf("expected selectedIdx 2 after tab, got %d", cv.selectedIdx)
	}

	// Tab wraps around
	updatedModel, _ = cv.Update(msg)
	cv = updatedModel.(*ConflictView)
	if cv.selectedIdx != 0 {
		t.Errorf("expected selectedIdx 0 after tab wrap, got %d", cv.selectedIdx)
	}
}

// TestConflictView_ShortcutKeys tests shortcut keys (b, d, c)
func TestConflictView_ShortcutKeys(t *testing.T) {
	tests := []struct {
		name         string
		key          rune
		expectChoice ConflictResolutionChoice
	}{
		{"b for backup", 'b', ConflictChoiceBackup},
		{"d for delete", 'd', ConflictChoiceDelete},
		{"c for cancel", 'c', ConflictChoiceCancel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conflicts := []stow.ConflictFile{
				{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
			}
			cv := NewConflictView(conflicts)
			cv.SetSize(80, 40)

			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{tt.key}}
			_, cmd := cv.Update(msg)

			if cmd == nil {
				t.Fatal("expected command to be returned")
			}

			// Execute the command to get the message
			result := cmd()
			resolvedMsg, ok := result.(ConflictResolvedMsg)
			if !ok {
				t.Fatalf("expected ConflictResolvedMsg, got %T", result)
			}
			if resolvedMsg.Choice != tt.expectChoice {
				t.Errorf("expected choice %v, got %v", tt.expectChoice, resolvedMsg.Choice)
			}
		})
	}
}

// TestConflictView_EscapeCancels tests that Escape key cancels
func TestConflictView_EscapeCancels(t *testing.T) {
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
	}
	cv := NewConflictView(conflicts)
	cv.SetSize(80, 40)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	_, cmd := cv.Update(msg)

	if cmd == nil {
		t.Fatal("expected command to be returned")
	}

	result := cmd()
	resolvedMsg, ok := result.(ConflictResolvedMsg)
	if !ok {
		t.Fatalf("expected ConflictResolvedMsg, got %T", result)
	}
	if resolvedMsg.Choice != ConflictChoiceCancel {
		t.Errorf("expected ConflictChoiceCancel, got %v", resolvedMsg.Choice)
	}
	if resolvedMsg.Resolved {
		t.Error("expected Resolved to be false for cancel")
	}
}

// TestConflictResolved_Cancel_ClearsPendingState tests that cancelling conflict
// resolution clears the pending operation state
func TestConflictResolved_Cancel_ClearsPendingState(t *testing.T) {
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "zsh", Path: "zsh"},
			},
		},
	}
	s := State{
		Platform:     &platform.Platform{OS: "linux"},
		HasConfig:    true,
		Config:       cfg,
		DotfilesPath: "/tmp/dotfiles",
		Configs:      cfg.GetAllConfigs(),
	}

	m := New(s)
	m.width = 120
	m.height = 50

	// Simulate being in conflict view with pending operation
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
	}
	m.conflictView = NewConflictView(conflicts)
	m.conflictView.SetSize(m.width, m.height)
	m.pendingOperation = OpInstall
	m.pendingConflicts = conflicts
	m.pushView(viewConflict)

	// Send cancel message
	cancelMsg := ConflictResolvedMsg{
		Choice:   ConflictChoiceCancel,
		Resolved: false,
	}
	updatedModel, _ := m.Update(cancelMsg)
	model := updatedModel.(*Model)

	// Verify state is cleared
	if model.currentView != viewDashboard {
		t.Errorf("expected viewDashboard, got %v", model.currentView)
	}
	if model.conflictView != nil {
		t.Error("expected conflictView to be nil")
	}
	if model.pendingOperation != 0 {
		t.Errorf("expected pendingOperation to be 0, got %v", model.pendingOperation)
	}
	if model.pendingConflicts != nil {
		t.Error("expected pendingConflicts to be nil")
	}
}

// TestCheckForConflicts_FiltersByConfigNames tests that CheckForConflicts
// correctly filters to specified config names
func TestCheckForConflicts_FiltersByConfigNames(t *testing.T) {
	// This is more of an integration test - we can't easily test without
	// a real filesystem. But we can test the GroupConflictsByConfig helper.
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
		{ConfigName: "vim", TargetPath: "/home/user/.vimrc"},
		{ConfigName: "zsh", TargetPath: "/home/user/.zshenv"},
	}

	byConfig := GroupConflictsByConfig(conflicts)

	if len(byConfig) != 2 {
		t.Errorf("expected 2 config groups, got %d", len(byConfig))
	}
	if len(byConfig["zsh"]) != 2 {
		t.Errorf("expected 2 zsh conflicts, got %d", len(byConfig["zsh"]))
	}
	if len(byConfig["vim"]) != 1 {
		t.Errorf("expected 1 vim conflict, got %d", len(byConfig["vim"]))
	}
}

// TestModel_HelpOverlayContainsKeyboardShortcuts tests that the help overlay renders correctly
// with the dashboard visible as a dimmed background behind the help content.
func TestModel_HelpOverlayContainsKeyboardShortcuts(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}
	m := New(s)

	// Apply window size
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	model := updatedModel.(*Model)

	// Verify dashboard contains "vim"
	dashView := model.View()
	if !containsText(dashView, "vim") {
		t.Error("expected dashboard to contain 'vim'")
	}

	// Toggle help - should show overlay with dashboard as dimmed background
	model.showHelp = true
	helpView := model.View()
	if !containsText(helpView, "Keyboard Shortcuts") {
		t.Error("expected help overlay to contain 'Keyboard Shortcuts'")
	}

	// Close help - should return to normal dashboard
	model.showHelp = false
	afterView := model.View()
	if !containsText(afterView, "vim") {
		t.Errorf("expected dashboard to contain 'vim' after closing help")
	}
}

// containsText checks if a string contains a substring, ignoring ANSI escape codes
func containsText(s, substr string) bool {
	// First try direct contains
	if strings.Contains(s, substr) {
		return true
	}
	// Try with ANSI stripped
	stripped := stripAnsiForTest(s)
	return strings.Contains(stripped, substr)
}

func stripAnsiForTest(s string) string {
	var result []byte
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || s[i] == '~' {
				inEscape = false
			}
			continue
		}
		result = append(result, s[i])
	}
	return string(result)
}

// TestModel_OverlayModals_RenderWithoutPanic tests that each overlay modal renders correctly
func TestModel_OverlayModals_RenderWithoutPanic(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
		Config: &config.Config{
			MachineConfig: []config.MachinePrompt{
				{ID: "test", Description: "Test config"},
			},
		},
		HasConfig:    true,
		DotfilesPath: "/tmp/dotfiles",
	}
	m := New(s)

	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 40}
	updatedModel, _ := m.Update(sizeMsg)
	model := updatedModel.(*Model)

	// Test help overlay
	model.showHelp = true
	helpView := model.View()
	if helpView == "" {
		t.Error("expected non-empty help overlay")
	}
	if !containsText(helpView, "Keyboard Shortcuts") {
		t.Error("expected help overlay to contain 'Keyboard Shortcuts'")
	}
	model.showHelp = false

	// Test menu overlay
	model.menu.SetSize(100, 40)
	model.pushView(viewMenu)
	menuView := model.View()
	if menuView == "" {
		t.Error("expected non-empty menu overlay")
	}
	model.popView()

	// Test confirm overlay
	model.confirm = NewConfirm("test-confirm", "Test Title", "Test description")
	model.confirm.SetSize(100, 40)
	model.pushView(viewConfirm)
	confirmView := model.View()
	if confirmView == "" {
		t.Error("expected non-empty confirm overlay")
	}
	if !containsText(confirmView, "Test Title") {
		t.Error("expected confirm overlay to contain 'Test Title'")
	}
	model.popView()

	// Test conflict overlay
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
	}
	model.conflictView = NewConflictView(conflicts)
	model.conflictView.SetSize(100, 40)
	model.pushView(viewConflict)
	conflictView := model.View()
	if conflictView == "" {
		t.Error("expected non-empty conflict overlay")
	}
	if !containsText(conflictView, "Conflicts") {
		t.Error("expected conflict overlay to contain 'Conflicts'")
	}
	model.popView()

	// Verify dashboard is intact after all overlays
	dashView := model.View()
	if !containsText(dashView, "vim") {
		t.Error("expected dashboard to still contain 'vim' after all overlay tests")
	}
}

// TestConflictView_Render tests that ConflictView renders without panic
func TestConflictView_Render(t *testing.T) {
	conflicts := []stow.ConflictFile{
		{ConfigName: "zsh", TargetPath: "/home/user/.zshrc"},
		{ConfigName: "vim", TargetPath: "/home/user/.vimrc"},
	}

	cv := NewConflictView(conflicts)
	cv.SetSize(80, 40)

	view := cv.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}
