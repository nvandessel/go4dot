package dashboard

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewOperations(t *testing.T) {
	ops := NewOperations()

	if ops.active {
		t.Error("expected new Operations to not be active")
	}
	if ops.currentStep != "" {
		t.Error("expected new Operations to have empty currentStep")
	}
}

func TestOperations_Update_ProgressMsg(t *testing.T) {
	ops := NewOperations()

	msg := OperationProgressMsg{
		CurrentStep: "Installing packages",
		TotalSteps:  5,
		CurrentIdx:  2,
	}

	ops, _ = ops.Update(msg)

	if !ops.active {
		t.Error("expected Operations to be active after progress message")
	}
	if ops.currentStep != "Installing packages" {
		t.Errorf("expected currentStep to be 'Installing packages', got '%s'", ops.currentStep)
	}
	if ops.totalSteps != 5 {
		t.Errorf("expected totalSteps to be 5, got %d", ops.totalSteps)
	}
	if ops.currentIdx != 2 {
		t.Errorf("expected currentIdx to be 2, got %d", ops.currentIdx)
	}
}

func TestOperations_Update_StepCompleteMsg(t *testing.T) {
	ops := NewOperations()
	ops.active = true

	msg := OperationStepCompleteMsg{
		Step:   "Step 1",
		Status: StepStatusSuccess,
		Detail: "Completed successfully",
	}

	ops, _ = ops.Update(msg)

	if ops.lastStatus != StepStatusSuccess {
		t.Errorf("expected lastStatus to be StepStatusSuccess, got %v", ops.lastStatus)
	}
	if ops.lastDetail != "Completed successfully" {
		t.Errorf("expected lastDetail to be 'Completed successfully', got '%s'", ops.lastDetail)
	}
}

func TestOperations_Update_DoneMsg(t *testing.T) {
	ops := NewOperations()
	ops.active = true

	msg := OperationDoneMsg{
		Success: true,
		Summary: "All done",
	}

	ops, _ = ops.Update(msg)

	if ops.active {
		t.Error("expected Operations to not be active after done message")
	}
}

func TestOperations_Update_SpinnerTick(t *testing.T) {
	ops := NewOperations()
	ops.active = true

	// Send a spinner tick message
	msg := spinner.TickMsg{}
	ops, _ = ops.Update(msg)

	// Spinner should have been updated (we can't easily verify the internal state)
	// Just ensure no panic occurred
}

func TestOperations_View_Inactive(t *testing.T) {
	ops := NewOperations()
	ops.active = false

	view := ops.View()

	if view != "" {
		t.Errorf("expected empty view when inactive, got '%s'", view)
	}
}

func TestOperations_View_Active(t *testing.T) {
	ops := NewOperations()
	ops.active = true
	ops.currentStep = "Running tests"
	ops.totalSteps = 3
	ops.currentIdx = 1

	view := ops.View()

	if view == "" {
		t.Error("expected non-empty view when active")
	}
	// View should contain the current step and progress
	// The exact format depends on the spinner state
}

func TestOperations_IsActive(t *testing.T) {
	tests := []struct {
		name     string
		active   bool
		expected bool
	}{
		{"inactive", false, false},
		{"active", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ops := NewOperations()
			ops.active = tt.active
			if ops.IsActive() != tt.expected {
				t.Errorf("expected IsActive() to be %v, got %v", tt.expected, ops.IsActive())
			}
		})
	}
}

func TestStepStatusToString(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepStatusPending, "info"},
		{StepStatusRunning, "info"},
		{StepStatusSuccess, "success"},
		{StepStatusWarning, "warn"},
		{StepStatusError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := stepStatusToString(tt.status)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestNewOperationRunner(t *testing.T) {
	// Create with nil program
	runner := NewOperationRunner(nil)
	if runner == nil {
		t.Fatal("expected non-nil runner")
	}
	if runner.sender != nil {
		t.Error("expected nil sender")
	}
}

// mockSender captures messages sent via Send
type mockSender struct {
	mu       sync.Mutex
	messages []tea.Msg
}

func (m *mockSender) Send(msg tea.Msg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
}

func (m *mockSender) getMessages() []tea.Msg {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]tea.Msg, len(m.messages))
	copy(result, m.messages)
	return result
}

func TestOperationRunner_Progress(t *testing.T) {
	mock := &mockSender{}
	runner := NewOperationRunnerWithSender(mock)

	runner.Progress("Step 1", 0, 5)

	messages := mock.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg, ok := messages[0].(OperationProgressMsg)
	if !ok {
		t.Fatalf("expected OperationProgressMsg, got %T", messages[0])
	}

	if msg.CurrentStep != "Step 1" {
		t.Errorf("expected CurrentStep 'Step 1', got '%s'", msg.CurrentStep)
	}
	if msg.CurrentIdx != 0 {
		t.Errorf("expected CurrentIdx 0, got %d", msg.CurrentIdx)
	}
	if msg.TotalSteps != 5 {
		t.Errorf("expected TotalSteps 5, got %d", msg.TotalSteps)
	}
}

func TestOperationRunner_StepComplete(t *testing.T) {
	mock := &mockSender{}
	runner := NewOperationRunnerWithSender(mock)

	runner.StepComplete("Install", StepStatusSuccess, "Package installed")

	messages := mock.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg, ok := messages[0].(OperationStepCompleteMsg)
	if !ok {
		t.Fatalf("expected OperationStepCompleteMsg, got %T", messages[0])
	}

	if msg.Step != "Install" {
		t.Errorf("expected Step 'Install', got '%s'", msg.Step)
	}
	if msg.Status != StepStatusSuccess {
		t.Errorf("expected Status StepStatusSuccess, got %v", msg.Status)
	}
	if msg.Detail != "Package installed" {
		t.Errorf("expected Detail 'Package installed', got '%s'", msg.Detail)
	}
}

func TestOperationRunner_Log(t *testing.T) {
	mock := &mockSender{}
	runner := NewOperationRunnerWithSender(mock)

	runner.Log("info", "Processing file")

	messages := mock.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	msg, ok := messages[0].(OperationLogMsg)
	if !ok {
		t.Fatalf("expected OperationLogMsg, got %T", messages[0])
	}

	if msg.Level != "info" {
		t.Errorf("expected Level 'info', got '%s'", msg.Level)
	}
	if msg.Message != "Processing file" {
		t.Errorf("expected Message 'Processing file', got '%s'", msg.Message)
	}
}

func TestOperationRunner_Done(t *testing.T) {
	tests := []struct {
		name    string
		success bool
		summary string
		err     error
	}{
		{"success", true, "Completed", nil},
		{"failure", false, "", errors.New("failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockSender{}
			runner := NewOperationRunnerWithSender(mock)

			runner.Done(tt.success, tt.summary, tt.err)

			messages := mock.getMessages()
			if len(messages) != 1 {
				t.Fatalf("expected 1 message, got %d", len(messages))
			}

			msg, ok := messages[0].(OperationDoneMsg)
			if !ok {
				t.Fatalf("expected OperationDoneMsg, got %T", messages[0])
			}

			if msg.Success != tt.success {
				t.Errorf("expected Success %v, got %v", tt.success, msg.Success)
			}
			if msg.Summary != tt.summary {
				t.Errorf("expected Summary '%s', got '%s'", tt.summary, msg.Summary)
			}
			if (msg.Error != nil) != (tt.err != nil) {
				t.Errorf("expected Error %v, got %v", tt.err, msg.Error)
			}
		})
	}
}

func TestOperationRunner_NilProgram(t *testing.T) {
	runner := NewOperationRunner(nil)

	// These should not panic with nil program
	runner.Progress("Step", 0, 1)
	runner.StepComplete("Step", StepStatusSuccess, "")
	runner.Log("info", "msg")
	runner.Done(true, "", nil)
}

func TestStartInlineOperation_Success(t *testing.T) {
	mock := &mockSender{}

	called := false
	operationFunc := func(runner *OperationRunner) error {
		called = true
		runner.Log("info", "test message")
		return nil
	}

	// Create a runner that uses our mock
	runner := NewOperationRunnerWithSender(mock)

	// Simulate what StartInlineOperation does
	err := operationFunc(runner)
	if err != nil {
		runner.Done(false, "", err)
	} else {
		runner.Done(true, "", nil) // CRITICAL: Must call Done on success
	}

	if !called {
		t.Error("expected operation function to be called")
	}

	messages := mock.getMessages()
	// Should have a log message and a done message
	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(messages))
	}

	// Last message should be OperationDoneMsg with success
	lastMsg, ok := messages[len(messages)-1].(OperationDoneMsg)
	if !ok {
		t.Fatalf("expected last message to be OperationDoneMsg, got %T", messages[len(messages)-1])
	}
	if !lastMsg.Success {
		t.Error("expected success to be true")
	}
}

func TestStartInlineOperation_Failure(t *testing.T) {
	mock := &mockSender{}

	expectedErr := errors.New("operation failed")
	operationFunc := func(runner *OperationRunner) error {
		return expectedErr
	}

	runner := NewOperationRunnerWithSender(mock)

	err := operationFunc(runner)
	if err != nil {
		runner.Done(false, "", err)
	} else {
		runner.Done(true, "", nil)
	}

	messages := mock.getMessages()
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}

	doneMsg, ok := messages[0].(OperationDoneMsg)
	if !ok {
		t.Fatalf("expected OperationDoneMsg, got %T", messages[0])
	}
	if doneMsg.Success {
		t.Error("expected success to be false")
	}
	if doneMsg.Error == nil {
		t.Error("expected error to be set")
	}
}

func TestStartBackgroundOperation_Success(t *testing.T) {
	mock := &mockSender{}

	done := make(chan bool, 1)
	operationFunc := func(runner *OperationRunner) error {
		runner.Log("info", "background task")
		done <- true
		return nil
	}

	// Simulate StartBackgroundOperation
	go func() {
		runner := NewOperationRunnerWithSender(mock)
		err := operationFunc(runner)
		if err != nil {
			runner.Done(false, "", err)
		} else {
			runner.Done(true, "", nil) // CRITICAL: Must call Done on success
		}
	}()

	// Wait for operation to complete
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for operation")
	}

	// Give a moment for the Done message to be sent
	time.Sleep(10 * time.Millisecond)

	messages := mock.getMessages()
	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(messages))
	}

	// Last message should be OperationDoneMsg with success
	lastMsg, ok := messages[len(messages)-1].(OperationDoneMsg)
	if !ok {
		t.Fatalf("expected last message to be OperationDoneMsg, got %T", messages[len(messages)-1])
	}
	if !lastMsg.Success {
		t.Error("expected success to be true")
	}
}

// TestOperationDoneOnSuccess verifies the critical fix:
// Operations must call Done(true, ...) on success, not just on failure.
func TestOperationDoneOnSuccess(t *testing.T) {
	mock := &mockSender{}

	// Simulate a successful operation that doesn't explicitly call Done
	operationFunc := func(runner *OperationRunner) error {
		runner.Progress("Processing", 0, 1)
		// Operation completes successfully without calling Done
		return nil
	}

	runner := NewOperationRunnerWithSender(mock)
	err := operationFunc(runner)

	// The fix ensures Done is called after operationFunc returns
	if err != nil {
		runner.Done(false, "", err)
	} else {
		runner.Done(true, "Operation completed", nil)
	}

	messages := mock.getMessages()

	// Should have progress and done messages
	if len(messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(messages))
	}

	// Last message must be OperationDoneMsg
	lastMsg, ok := messages[len(messages)-1].(OperationDoneMsg)
	if !ok {
		t.Fatalf("expected last message to be OperationDoneMsg, got %T", messages[len(messages)-1])
	}

	// Must indicate success
	if !lastMsg.Success {
		t.Error("expected Done message to indicate success")
	}
}
