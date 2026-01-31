package dashboard

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
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
	// Doctor action now shows inline view instead of quitting
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

		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
		updatedModel, cmd := m.Update(msg)

		// Doctor now runs inline, should get init command
		if cmd == nil {
			t.Error("expected init command for doctor view")
		}
		model := updatedModel.(*Model)
		// Should switch to doctor view
		if model.currentView != viewDoctor {
			t.Errorf("expected viewDoctor, got %v", model.currentView)
		}
		// Doctor view should be initialized
		if model.doctorView == nil {
			t.Error("expected doctorView to be initialized")
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
	if len(model.output.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(model.output.logs))
	}

	// Test OperationDoneMsg
	doneMsg := OperationDoneMsg{Success: true, Summary: "done"}
	updatedModel, _ = model.Update(doneMsg)
	model = updatedModel.(*Model)
	if model.operationActive {
		t.Error("expected operationActive to be false after OperationDoneMsg")
	}
	// Should have 2 logs now (the log + the done summary)
	if len(model.output.logs) != 2 {
		t.Errorf("expected 2 log entries, got %d", len(model.output.logs))
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
	// Initialize filtered indexes directly because the sidebar's filteredIdxs
	// is normally populated by updateFilter() which requires filter mode entry.
	// For unit testing SelectAll behavior in isolation, we set this internal
	// state to avoid coupling this test to filter mode mechanics.
	m.sidebar.filteredIdxs = []int{0, 1, 2}

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

	// Open menu with tab
	msg := tea.KeyMsg{Type: tea.KeyTab}
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
		{"Menu", keys.Menu, []string{"tab"}},
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
	// Verify component widths are set correctly after WindowSizeMsg.
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

	// Sidebar gets 1/3 of width
	expectedSidebarWidth := 120 / 3
	if model.sidebar.width != expectedSidebarWidth {
		t.Errorf("expected sidebar width %d, got %d", expectedSidebarWidth, model.sidebar.width)
	}

	// Details gets remaining 2/3
	expectedDetailsWidth := 120 - expectedSidebarWidth
	if model.details.width != expectedDetailsWidth {
		t.Errorf("expected details width %d, got %d", expectedDetailsWidth, model.details.width)
	}

	// View should render without panic
	view := model.viewDashboard()
	if view == "" {
		t.Error("expected non-empty dashboard view")
	}
}
