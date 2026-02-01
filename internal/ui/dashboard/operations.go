package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// OperationType represents the type of operation being performed
type OperationType int

const (
	OpInstall OperationType = iota
	OpSync
	OpSyncSingle
	OpBulkSync
	OpUpdate
	OpDoctor
	OpUninstall
	OpExternal
	OpExternalSingle
)

// String returns a human-readable name for the operation type
func (op OperationType) String() string {
	switch op {
	case OpInstall:
		return "Installing"
	case OpSync:
		return "Syncing All"
	case OpSyncSingle:
		return "Syncing"
	case OpBulkSync:
		return "Bulk Syncing"
	case OpUpdate:
		return "Updating"
	case OpDoctor:
		return "Running Doctor"
	case OpUninstall:
		return "Uninstalling"
	case OpExternal:
		return "External Dependencies"
	case OpExternalSingle:
		return "External"
	default:
		return "Processing"
	}
}

// OperationStep represents a single step in an operation
type OperationStep struct {
	Name   string
	Status StepStatus
	Detail string
}

// StepStatus represents the status of a step
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepSuccess
	StepWarning
	StepError
	StepSkipped
)

// OperationProgressMsg is sent to update operation progress
type OperationProgressMsg struct {
	Step      string
	Current   int
	Total     int
	Detail    string
	StepIndex int
}

// OperationStepCompleteMsg is sent when a step completes
type OperationStepCompleteMsg struct {
	StepIndex int
	Status    StepStatus
	Detail    string
}

// OperationDoneMsg is sent when an operation completes
type OperationDoneMsg struct {
	Success bool
	Summary string
	Error   error
}

// OperationLogMsg adds a log entry
type OperationLogMsg struct {
	Level   string // "info", "success", "warning", "error"
	Message string
}

// Operations is the model for running operations within the dashboard
type Operations struct {
	operationType OperationType
	configName    string // For single config operations
	configNames   []string
	width         int
	height        int
	spinner       spinner.Model
	steps         []OperationStep
	currentStep   int
	logs          []logEntry
	done          bool
	success       bool
	summary       string
	err           error
}

type logEntry struct {
	level   string
	message string
}

// NewOperations creates a new operations component
func NewOperations(opType OperationType, configName string, configNames []string) Operations {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	steps := getStepsForOperation(opType)

	return Operations{
		operationType: opType,
		configName:    configName,
		configNames:   configNames,
		spinner:       s,
		steps:         steps,
		logs:          []logEntry{},
	}
}

// getStepsForOperation returns the steps for a given operation type
func getStepsForOperation(opType OperationType) []OperationStep {
	switch opType {
	case OpInstall:
		return []OperationStep{
			{Name: "Detecting platform", Status: StepPending},
			{Name: "Installing dependencies", Status: StepPending},
			{Name: "Stowing configs", Status: StepPending},
			{Name: "Cloning external dependencies", Status: StepPending},
			{Name: "Configuring machine settings", Status: StepPending},
		}
	case OpSync, OpSyncSingle, OpBulkSync:
		return []OperationStep{
			{Name: "Checking symlinks", Status: StepPending},
			{Name: "Syncing configs", Status: StepPending},
			{Name: "Updating state", Status: StepPending},
		}
	case OpUpdate:
		return []OperationStep{
			{Name: "Checking external dependencies", Status: StepPending},
			{Name: "Updating repositories", Status: StepPending},
		}
	case OpDoctor:
		return []OperationStep{
			{Name: "Checking dependencies", Status: StepPending},
			{Name: "Checking configs", Status: StepPending},
			{Name: "Checking symlinks", Status: StepPending},
		}
	case OpUninstall:
		return []OperationStep{
			{Name: "Removing symlinks", Status: StepPending},
			{Name: "Removing external dependencies", Status: StepPending},
			{Name: "Cleaning state", Status: StepPending},
		}
	case OpExternalSingle:
		return []OperationStep{
			{Name: "Checking status", Status: StepPending},
			{Name: "Processing", Status: StepPending},
		}
	default:
		return []OperationStep{
			{Name: "Processing", Status: StepPending},
		}
	}
}

// Init initializes the operations component
func (o Operations) Init() tea.Cmd {
	return o.spinner.Tick
}

// Update handles messages for the operations component
func (o *Operations) Update(msg tea.Msg) (Operations, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		var cmd tea.Cmd
		o.spinner, cmd = o.spinner.Update(msg)
		return *o, cmd

	case OperationProgressMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(o.steps) {
			o.steps[msg.StepIndex].Status = StepRunning
			o.steps[msg.StepIndex].Detail = msg.Detail
			o.currentStep = msg.StepIndex
		}
		return *o, nil

	case OperationStepCompleteMsg:
		if msg.StepIndex >= 0 && msg.StepIndex < len(o.steps) {
			o.steps[msg.StepIndex].Status = msg.Status
			o.steps[msg.StepIndex].Detail = msg.Detail
		}
		return *o, nil

	case OperationLogMsg:
		o.logs = append(o.logs, logEntry{level: msg.Level, message: msg.Message})
		// Keep only last 10 logs to avoid scrolling issues
		if len(o.logs) > 10 {
			o.logs = o.logs[len(o.logs)-10:]
		}
		return *o, nil

	case OperationDoneMsg:
		o.done = true
		o.success = msg.Success
		o.summary = msg.Summary
		o.err = msg.Error
		return *o, nil

	case tea.WindowSizeMsg:
		o.width = msg.Width
		o.height = msg.Height
		return *o, nil
	}

	return *o, nil
}

// safeWidth returns a non-negative width value
func safeWidth(w int) int {
	if w < 0 {
		return 0
	}
	return w
}

// View renders the operations component
func (o Operations) View() string {
	var b strings.Builder

	// Calculate safe box width
	boxWidth := safeWidth(o.width - 4)

	// Title
	title := o.operationType.String()
	if o.configName != "" {
		title = fmt.Sprintf("%s: %s", o.operationType.String(), o.configName)
	} else if len(o.configNames) > 0 {
		title = fmt.Sprintf("%s (%d configs)", o.operationType.String(), len(o.configNames))
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.PrimaryColor).
		MarginBottom(1)

	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	// Steps
	for i, step := range o.steps {
		var icon string
		var style lipgloss.Style

		switch step.Status {
		case StepPending:
			icon = "  "
			style = ui.SubtleStyle
		case StepRunning:
			icon = o.spinner.View()
			style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)
		case StepSuccess:
			icon = ui.SuccessStyle.Render("✓")
			style = ui.SuccessStyle
		case StepWarning:
			icon = ui.WarningStyle.Render("⚠")
			style = ui.WarningStyle
		case StepError:
			icon = ui.ErrorStyle.Render("✗")
			style = ui.ErrorStyle
		case StepSkipped:
			icon = ui.SubtleStyle.Render("⊘")
			style = ui.SubtleStyle
		}

		stepNum := ui.SubtleStyle.Render(fmt.Sprintf("%d.", i+1))
		stepLine := fmt.Sprintf(" %s %s %s", icon, stepNum, style.Render(step.Name))

		if step.Detail != "" && step.Status == StepRunning {
			stepLine += ui.SubtleStyle.Render(fmt.Sprintf(" - %s", step.Detail))
		}

		b.WriteString(stepLine)
		b.WriteString("\n")
	}

	// Logs
	if len(o.logs) > 0 {
		b.WriteString("\n")
		logBoxStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ui.SubtleColor).
			Padding(0, 1).
			Width(boxWidth)

		var logLines []string
		for _, log := range o.logs {
			var icon string
			switch log.level {
			case "success":
				icon = ui.SuccessStyle.Render("✓")
			case "warning":
				icon = ui.WarningStyle.Render("⚠")
			case "error":
				icon = ui.ErrorStyle.Render("✗")
			default:
				icon = ui.SubtleStyle.Render("•")
			}
			logLines = append(logLines, fmt.Sprintf("%s %s", icon, log.message))
		}
		b.WriteString(logBoxStyle.Render(strings.Join(logLines, "\n")))
		b.WriteString("\n")
	}

	// Completion message
	if o.done {
		b.WriteString("\n")
		if o.success {
			successBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ui.SecondaryColor).
				Padding(0, 1).
				Width(boxWidth)
			b.WriteString(successBox.Render(ui.SuccessStyle.Render("✓ ") + o.summary))
		} else if o.err != nil {
			errorBox := lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ui.ErrorColor).
				Padding(0, 1).
				Width(boxWidth)
			b.WriteString(errorBox.Render(ui.ErrorStyle.Render("✗ Error: ") + o.err.Error()))
		}
		b.WriteString("\n\n")
		b.WriteString(ui.SubtleStyle.Render("Press Enter to continue, q to quit"))
	}

	return b.String()
}

// IsDone returns whether the operation has completed
func (o Operations) IsDone() bool {
	return o.done
}

// IsSuccess returns whether the operation completed successfully
func (o Operations) IsSuccess() bool {
	return o.success
}

// GetError returns any error from the operation
func (o Operations) GetError() error {
	return o.err
}

// OperationRunner is a helper for running operations and sending progress updates
type OperationRunner struct {
	program *tea.Program
}

// NewOperationRunner creates a new operation runner
func NewOperationRunner(p *tea.Program) *OperationRunner {
	return &OperationRunner{program: p}
}

// Progress sends a progress update
func (r *OperationRunner) Progress(stepIndex int, detail string) {
	r.program.Send(OperationProgressMsg{
		StepIndex: stepIndex,
		Detail:    detail,
	})
}

// StepComplete marks a step as complete
func (r *OperationRunner) StepComplete(stepIndex int, status StepStatus, detail string) {
	r.program.Send(OperationStepCompleteMsg{
		StepIndex: stepIndex,
		Status:    status,
		Detail:    detail,
	})
}

// Log adds a log entry
func (r *OperationRunner) Log(level, message string) {
	r.program.Send(OperationLogMsg{
		Level:   level,
		Message: message,
	})
}

// Done marks the operation as complete
func (r *OperationRunner) Done(success bool, summary string, err error) {
	r.program.Send(OperationDoneMsg{
		Success: success,
		Summary: summary,
		Error:   err,
	})
}
