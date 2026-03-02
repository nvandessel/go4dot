package dashboard

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui"
)

func TestNewOperations(t *testing.T) {
	tests := []struct {
		name        string
		opType      OperationType
		configName  string
		configNames []string
		wantSteps   int
	}{
		{
			name:      "Install operation",
			opType:    OpInstall,
			wantSteps: 5,
		},
		{
			name:      "Sync operation",
			opType:    OpSync,
			wantSteps: 3,
		},
		{
			name:       "Sync single operation",
			opType:     OpSyncSingle,
			configName: "vim",
			wantSteps:  3,
		},
		{
			name:        "Bulk sync operation",
			opType:      OpBulkSync,
			configNames: []string{"vim", "zsh"},
			wantSteps:   3,
		},
		{
			name:      "Update operation",
			opType:    OpUpdate,
			wantSteps: 2,
		},
		{
			name:      "Doctor operation",
			opType:    OpDoctor,
			wantSteps: 3,
		},
		{
			name:      "Uninstall operation",
			opType:    OpUninstall,
			wantSteps: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := NewOperations(tt.opType, tt.configName, tt.configNames)

			if op.operationType != tt.opType {
				t.Errorf("expected operationType %v, got %v", tt.opType, op.operationType)
			}
			if op.configName != tt.configName {
				t.Errorf("expected configName %s, got %s", tt.configName, op.configName)
			}
			if len(op.steps) != tt.wantSteps {
				t.Errorf("expected %d steps, got %d", tt.wantSteps, len(op.steps))
			}
			if op.done {
				t.Error("expected done to be false initially")
			}
		})
	}
}

func TestOperationType_String(t *testing.T) {
	tests := []struct {
		opType OperationType
		want   string
	}{
		{OpInstall, "Installing"},
		{OpSync, "Syncing All"},
		{OpSyncSingle, "Syncing"},
		{OpBulkSync, "Bulk Syncing"},
		{OpUpdate, "Updating"},
		{OpDoctor, "Running Doctor"},
		{OpUninstall, "Uninstalling"},
		{OpExternal, "External Dependencies"},
		{OperationType(99), "Processing"}, // Unknown type
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.opType.String()
			if got != tt.want {
				t.Errorf("expected %s, got %s", tt.want, got)
			}
		})
	}
}

func TestOperations_Init(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	cmd := op.Init()

	if cmd == nil {
		t.Error("expected non-nil command from Init")
	}
}

func TestOperations_Update_Progress(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	// Send progress message
	msg := OperationProgressMsg{
		StepIndex: 0,
		Detail:    "Testing platform...",
	}

	updatedOp, _ := op.Update(msg)

	if updatedOp.steps[0].Status != StepRunning {
		t.Errorf("expected step 0 status to be StepRunning, got %v", updatedOp.steps[0].Status)
	}
	if updatedOp.steps[0].Detail != "Testing platform..." {
		t.Errorf("expected detail 'Testing platform...', got '%s'", updatedOp.steps[0].Detail)
	}
}

func TestOperations_Update_StepComplete(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	// Send step complete message
	msg := OperationStepCompleteMsg{
		StepIndex: 0,
		Status:    StepSuccess,
		Detail:    "Linux detected",
	}

	updatedOp, _ := op.Update(msg)

	if updatedOp.steps[0].Status != StepSuccess {
		t.Errorf("expected step 0 status to be StepSuccess, got %v", updatedOp.steps[0].Status)
	}
	if updatedOp.steps[0].Detail != "Linux detected" {
		t.Errorf("expected detail 'Linux detected', got '%s'", updatedOp.steps[0].Detail)
	}
}

func TestOperations_Update_Log(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	// Send log message
	msg := OperationLogMsg{
		Level:   "info",
		Message: "Installing package...",
	}

	updatedOp, _ := op.Update(msg)

	if len(updatedOp.logs) != 1 {
		t.Errorf("expected 1 log entry, got %d", len(updatedOp.logs))
	}
	if updatedOp.logs[0].message != "Installing package..." {
		t.Errorf("expected log message 'Installing package...', got '%s'", updatedOp.logs[0].message)
	}
}

func TestOperations_Update_LogLimit(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	// Send 15 log messages
	for i := 0; i < 15; i++ {
		msg := OperationLogMsg{
			Level:   "info",
			Message: "Message",
		}
		op, _ = op.Update(msg)
	}

	// Should only keep last 10
	if len(op.logs) != 10 {
		t.Errorf("expected 10 log entries (limited), got %d", len(op.logs))
	}
}

func TestOperations_Update_Done(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	// Send done message
	msg := OperationDoneMsg{
		Success: true,
		Summary: "Installation complete",
		Error:   nil,
	}

	updatedOp, _ := op.Update(msg)

	if !updatedOp.done {
		t.Error("expected done to be true")
	}
	if !updatedOp.success {
		t.Error("expected success to be true")
	}
	if updatedOp.summary != "Installation complete" {
		t.Errorf("expected summary 'Installation complete', got '%s'", updatedOp.summary)
	}
}

func TestOperations_Update_WindowSize(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	msg := tea.WindowSizeMsg{Width: 120, Height: 50}
	updatedOp, _ := op.Update(msg)

	if updatedOp.width != 120 {
		t.Errorf("expected width 120, got %d", updatedOp.width)
	}
	if updatedOp.height != 50 {
		t.Errorf("expected height 50, got %d", updatedOp.height)
	}
}

func TestOperations_Update_SpinnerTick(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	// Get a spinner tick message
	msg := spinner.TickMsg{}
	_, cmd := op.Update(msg)

	// Should return a command for next tick
	if cmd == nil {
		t.Error("expected non-nil command from spinner tick")
	}
}

func TestOperations_View(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	op.width = 80
	op.height = 40

	view := op.View()

	if view == "" {
		t.Error("expected non-empty view")
	}

	// Should contain the operation title
	if !strings.Contains(view, "Installing") {
		t.Error("expected view to contain 'Installing'")
	}
}

func TestOperations_View_WithConfigName(t *testing.T) {
	op := NewOperations(OpSyncSingle, "vim", nil)
	op.width = 80
	op.height = 40

	view := op.View()

	if !strings.Contains(view, "vim") {
		t.Error("expected view to contain config name 'vim'")
	}
}

func TestOperations_View_WithConfigNames(t *testing.T) {
	op := NewOperations(OpBulkSync, "", []string{"vim", "zsh", "tmux"})
	op.width = 80
	op.height = 40

	view := op.View()

	if !strings.Contains(view, "3 configs") {
		t.Error("expected view to contain '3 configs'")
	}
}

func TestOperations_View_Done(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	op.width = 80
	op.height = 40
	op.done = true
	op.success = true
	op.summary = "All done!"

	view := op.View()

	if !strings.Contains(view, "All done!") {
		t.Error("expected view to contain summary 'All done!'")
	}
	if !strings.Contains(view, "Press Enter to continue") {
		t.Error("expected view to contain 'Press Enter to continue'")
	}
}

func TestOperations_View_WithLogs(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	op.width = 80
	op.height = 40
	op.logs = []logEntry{
		{level: "info", message: "Starting..."},
		{level: "success", message: "Done!"},
		{level: "warning", message: "Check this"},
		{level: "error", message: "Failed!"},
	}

	view := op.View()

	if !strings.Contains(view, "Starting...") {
		t.Error("expected view to contain log message")
	}
}

func TestOperations_View_AllStepStatuses(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	op.width = 80
	op.height = 40

	// Set various step statuses
	if len(op.steps) >= 5 {
		op.steps[0].Status = StepSuccess
		op.steps[0].Detail = "Platform detected"
		op.steps[1].Status = StepRunning
		op.steps[1].Detail = "Installing..."
		op.steps[2].Status = StepWarning
		op.steps[3].Status = StepError
		op.steps[4].Status = StepSkipped
	}

	view := op.View()

	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestOperations_View_DoneWithError(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	op.width = 80
	op.height = 40
	op.done = true
	op.success = false
	op.err = &TestError{msg: "installation failed"}

	view := op.View()

	if !strings.Contains(view, "installation failed") {
		t.Error("expected view to contain error message")
	}
}

func TestOperations_View_NegativeWidth(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)
	op.width = -10 // Negative width to test safeWidth
	op.height = 40
	op.logs = []logEntry{{level: "info", message: "Test"}}
	op.done = true
	op.success = true
	op.summary = "Done"

	// Should not panic
	view := op.View()
	if view == "" {
		t.Error("expected non-empty view even with negative width")
	}
}

func TestOperations_IsDone(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	if op.IsDone() {
		t.Error("expected IsDone to be false initially")
	}

	op.done = true
	if !op.IsDone() {
		t.Error("expected IsDone to be true after setting done")
	}
}

func TestOperations_IsSuccess(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	if op.IsSuccess() {
		t.Error("expected IsSuccess to be false initially")
	}

	op.success = true
	if !op.IsSuccess() {
		t.Error("expected IsSuccess to be true after setting success")
	}
}

func TestOperations_GetError(t *testing.T) {
	op := NewOperations(OpInstall, "", nil)

	if op.GetError() != nil {
		t.Error("expected GetError to be nil initially")
	}

	testErr := &TestError{msg: "test error"}
	op.err = testErr

	if op.GetError() != testErr {
		t.Error("expected GetError to return set error")
	}
}

// TestError is a simple error implementation for testing
type TestError struct {
	msg string
}

func (e *TestError) Error() string {
	return e.msg
}

func TestGetStepsForOperation(t *testing.T) {
	tests := []struct {
		opType    OperationType
		wantSteps int
	}{
		{OpInstall, 5},
		{OpSync, 3},
		{OpSyncSingle, 3},
		{OpBulkSync, 3},
		{OpUpdate, 2},
		{OpDoctor, 3},
		{OpUninstall, 3},
		{OpExternal, 1}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.opType.String(), func(t *testing.T) {
			steps := getStepsForOperation(tt.opType)
			if len(steps) != tt.wantSteps {
				t.Errorf("expected %d steps for %v, got %d", tt.wantSteps, tt.opType, len(steps))
			}

			// All steps should start as pending
			for i, step := range steps {
				if step.Status != StepPending {
					t.Errorf("step %d should be pending, got %v", i, step.Status)
				}
			}
		})
	}
}

func TestModel_Update_OperationView(t *testing.T) {
	s := State{
		AutoStart:      true,
		StartOperation: OpInstall,
		HasConfig:      true,
	}
	m := New(s)

	if m.currentView != viewOperation {
		t.Errorf("expected currentView to be viewOperation when AutoStart is true, got %v", m.currentView)
	}
}

func TestModel_Init_OperationMode(t *testing.T) {
	s := State{
		AutoStart:      true,
		StartOperation: OpInstall,
		HasConfig:      true,
	}
	m := New(s)

	cmd := m.Init()
	if cmd == nil {
		t.Error("expected non-nil command from Init in operation mode")
	}
}

func TestModel_UpdateOperation_Done(t *testing.T) {
	// Test with AutoStart = true: should quit on Enter
	t.Run("AutoStart_Quit", func(t *testing.T) {
		s := State{
			AutoStart:      true,
			StartOperation: OpInstall,
			HasConfig:      true,
		}
		m := New(s)
		m.width = 80
		m.height = 40

		// Mark operation as done
		m.operations.done = true
		m.operations.success = true

		// Press enter should quit when AutoStart is true
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(*Model)

		if !model.quitting {
			t.Error("expected quitting to be true after Enter on completed AutoStart operation")
		}
	})

	// Test with AutoStart = false: should return to dashboard on Enter
	t.Run("NonAutoStart_ReturnToDashboard", func(t *testing.T) {
		s := State{
			AutoStart:      false,
			StartOperation: OpInstall,
			HasConfig:      true,
		}
		m := New(s)
		m.width = 80
		m.height = 40
		m.currentView = viewOperation // Manually set to operation view

		// Initialize operations
		m.operations = NewOperations(OpInstall, "", nil)
		m.operations.done = true
		m.operations.success = true

		// Press enter should return to dashboard when AutoStart is false
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		updatedModel, _ := m.Update(msg)
		model := updatedModel.(*Model)

		if model.currentView != viewDashboard {
			t.Errorf("expected currentView to be viewDashboard after Enter on completed non-AutoStart operation, got %v", model.currentView)
		}
	})
}

func TestModel_UpdateOperation_Quit(t *testing.T) {
	s := State{
		AutoStart:      true,
		StartOperation: OpInstall,
		HasConfig:      true,
	}
	m := New(s)

	// Press q to quit
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("expected tea.Quit command")
	}

	model := updatedModel.(*Model)
	if !model.quitting {
		t.Error("expected quitting to be true")
	}
}

func TestModel_View_Operation(t *testing.T) {
	s := State{
		AutoStart:      true,
		StartOperation: OpInstall,
		HasConfig:      true,
	}
	m := New(s)
	m.width = 80
	m.height = 40

	view := m.View()

	if view == "" {
		t.Error("expected non-empty view in operation mode")
	}
	if !strings.Contains(view, "Installing") {
		t.Error("expected view to contain 'Installing' in operation mode")
	}
}

func TestModel_View_OperationUsesOverlay(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)
	m.width = 100
	m.height = 40
	m.currentView = viewOperation
	m.operations = NewOperations(OpSync, "", nil)
	m.operations.width = 80
	m.operations.height = 30

	view := m.View()

	// Should contain the operation content (from the overlay)
	if !strings.Contains(view, "Syncing All") {
		t.Error("expected overlay view to contain 'Syncing All'")
	}

	// The view should NOT be the same as what viewDashboard() would return,
	// confirming that the overlay compositing is happening
	dashboardOnly := m.viewDashboard()
	if view == dashboardOnly {
		t.Error("expected operation overlay view to differ from plain dashboard view")
	}
}

func TestModel_UpdateOperation_WindowSizeUsesOverlayContentSize(t *testing.T) {
	s := State{
		Platform: &platform.Platform{OS: "linux"},
		Configs: []config.ConfigItem{
			{Name: "vim"},
		},
		HasConfig: true,
	}
	m := New(s)
	m.currentView = viewOperation
	m.operations = NewOperations(OpInstall, "", nil)

	// Send a WindowSizeMsg
	msg := tea.WindowSizeMsg{Width: 120, Height: 50}
	updatedModel, _ := m.Update(msg)
	model := updatedModel.(*Model)

	// The operations width/height should be the overlay content size,
	// which is smaller than the raw terminal dimensions due to border and padding
	style := ui.DefaultOverlayStyle()
	expectedWidth, expectedHeight := overlayContentSize(120, 50, style)

	if model.operations.width != expectedWidth {
		t.Errorf("expected operations width to be %d (overlay content size), got %d", expectedWidth, model.operations.width)
	}
	if model.operations.height != expectedHeight {
		t.Errorf("expected operations height to be %d (overlay content size), got %d", expectedHeight, model.operations.height)
	}

	// Ensure the overlay content size is strictly less than raw terminal size
	if expectedWidth >= 120 {
		t.Errorf("expected overlay content width (%d) to be less than terminal width (120)", expectedWidth)
	}
	if expectedHeight >= 50 {
		t.Errorf("expected overlay content height (%d) to be less than terminal height (50)", expectedHeight)
	}
}
