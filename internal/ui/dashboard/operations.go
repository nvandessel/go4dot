package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// StepStatus represents the completion status of an operation step.
type StepStatus int

const (
	StepStatusPending StepStatus = iota
	StepStatusRunning
	StepStatusSuccess
	StepStatusWarning
	StepStatusError
)

// OperationProgressMsg signals that an operation has started or updated.
type OperationProgressMsg struct {
	CurrentStep string
	TotalSteps  int
	CurrentIdx  int
}

// OperationStepCompleteMsg signals that a step in an operation has completed.
type OperationStepCompleteMsg struct {
	Step   string
	Status StepStatus
	Detail string
}

// OperationLogMsg signals a log message from an operation.
type OperationLogMsg struct {
	Level   string // "info", "warn", "error", "success"
	Message string
}

// OperationDoneMsg signals that an operation has completed.
type OperationDoneMsg struct {
	Success bool
	Summary string
	Error   error
}

// ProgramSender is an interface for sending messages to a tea.Program.
type ProgramSender interface {
	Send(msg tea.Msg)
}

// OperationRunner provides methods for running operations to report progress.
type OperationRunner struct {
	sender ProgramSender
}

// NewOperationRunner creates a new OperationRunner.
func NewOperationRunner(p *tea.Program) *OperationRunner {
	if p == nil {
		return &OperationRunner{sender: nil}
	}
	return &OperationRunner{sender: p}
}

// NewOperationRunnerWithSender creates a new OperationRunner with a custom sender.
// This is useful for testing.
func NewOperationRunnerWithSender(sender ProgramSender) *OperationRunner {
	return &OperationRunner{sender: sender}
}

// Progress reports the current progress of an operation.
func (r *OperationRunner) Progress(currentStep string, currentIdx, totalSteps int) {
	if r.sender != nil {
		r.sender.Send(OperationProgressMsg{
			CurrentStep: currentStep,
			TotalSteps:  totalSteps,
			CurrentIdx:  currentIdx,
		})
	}
}

// StepComplete reports that a step has completed.
func (r *OperationRunner) StepComplete(step string, status StepStatus, detail string) {
	if r.sender != nil {
		r.sender.Send(OperationStepCompleteMsg{
			Step:   step,
			Status: status,
			Detail: detail,
		})
	}
}

// Log reports a log message.
func (r *OperationRunner) Log(level, message string) {
	if r.sender != nil {
		r.sender.Send(OperationLogMsg{
			Level:   level,
			Message: message,
		})
	}
}

// Done reports that the operation has completed.
func (r *OperationRunner) Done(success bool, summary string, err error) {
	if r.sender != nil {
		r.sender.Send(OperationDoneMsg{
			Success: success,
			Summary: summary,
			Error:   err,
		})
	}
}

// OperationFunc is a function that runs an operation using an OperationRunner.
type OperationFunc func(runner *OperationRunner) error

// Operations is the model for the operations component that shows operation status.
type Operations struct {
	spinner     spinner.Model
	active      bool
	currentStep string
	totalSteps  int
	currentIdx  int
	lastStatus  StepStatus
	lastDetail  string
}

// NewOperations creates a new Operations component.
func NewOperations() Operations {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)
	return Operations{
		spinner: s,
	}
}

// Init initializes the operations component.
func (o Operations) Init() tea.Cmd {
	return o.spinner.Tick
}

// Update handles messages for the operations component.
func (o Operations) Update(msg tea.Msg) (Operations, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case OperationProgressMsg:
		o.active = true
		o.currentStep = msg.CurrentStep
		o.totalSteps = msg.TotalSteps
		o.currentIdx = msg.CurrentIdx
		return o, o.spinner.Tick
	case OperationStepCompleteMsg:
		o.lastStatus = msg.Status
		o.lastDetail = msg.Detail
	case OperationDoneMsg:
		o.active = false
	case spinner.TickMsg:
		if o.active {
			o.spinner, cmd = o.spinner.Update(msg)
		}
	}

	return o, cmd
}

// View renders the operations component.
func (o Operations) View() string {
	if !o.active {
		return ""
	}

	var b strings.Builder

	// Progress indicator
	progress := ""
	if o.totalSteps > 0 {
		progress = fmt.Sprintf(" [%d/%d]", o.currentIdx+1, o.totalSteps)
	}

	b.WriteString(fmt.Sprintf("%s %s%s", o.spinner.View(), o.currentStep, progress))

	return b.String()
}

// IsActive returns whether an operation is currently running.
func (o Operations) IsActive() bool {
	return o.active
}

// stepStatusToString converts a StepStatus to a log level string.
func stepStatusToString(status StepStatus) string {
	switch status {
	case StepStatusSuccess:
		return "success"
	case StepStatusWarning:
		return "warn"
	case StepStatusError:
		return "error"
	default:
		return "info"
	}
}

// StartInlineOperation starts an operation that runs in a goroutine.
// The operationFunc receives an OperationRunner to report progress.
// IMPORTANT: This function ensures runner.Done is called on both success and failure.
func StartInlineOperation(p *tea.Program, operationFunc OperationFunc) tea.Cmd {
	return func() tea.Msg {
		runner := NewOperationRunner(p)

		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("operation panicked: %v", r)
				runner.Done(false, "", err)
			}
		}()

		err := operationFunc(runner)
		if err != nil {
			runner.Done(false, "", err)
		} else {
			// CRITICAL: Call Done on success to ensure operationActive becomes false.
			// Without this, the operation would remain marked as active indefinitely.
			runner.Done(true, "", nil)
		}

		return nil
	}
}

// StartBackgroundOperation starts an operation in a separate goroutine.
// The operationFunc receives an OperationRunner to report progress.
// IMPORTANT: This function ensures runner.Done is called on both success and failure.
func StartBackgroundOperation(p *tea.Program, operationFunc OperationFunc) {
	go func() {
		runner := NewOperationRunner(p)

		// Recover from panics
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("operation panicked: %v", r)
				runner.Done(false, "", err)
			}
		}()

		err := operationFunc(runner)
		if err != nil {
			runner.Done(false, "", err)
		} else {
			// CRITICAL: Call Done on success to ensure operationActive becomes false.
			// Without this, the operation would remain marked as active indefinitely.
			runner.Done(true, "", nil)
		}
	}()
}
