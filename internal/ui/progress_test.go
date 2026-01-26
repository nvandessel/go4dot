package ui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewProgressTracker(t *testing.T) {
	steps := []string{"Step 1", "Step 2", "Step 3"}
	tracker := NewProgressTracker(steps)

	if tracker.totalSteps != 3 {
		t.Errorf("expected totalSteps to be 3, got %d", tracker.totalSteps)
	}
	if !tracker.showProgress {
		t.Error("expected showProgress to be true")
	}
	if len(tracker.steps) != 3 {
		t.Errorf("expected 3 steps, got %d", len(tracker.steps))
	}
}

func TestProgressTracker_StartStep(t *testing.T) {
	steps := []string{"Initialize", "Install", "Configure"}
	tracker := NewProgressTracker(steps)

	tracker.StartStep(1)

	if tracker.currentStep != 1 {
		t.Errorf("expected currentStep to be 1, got %d", tracker.currentStep)
	}
	if tracker.currentItem != 0 {
		t.Errorf("expected currentItem to be 0, got %d", tracker.currentItem)
	}
}

func TestProgressTracker_StartStep_Invalid(t *testing.T) {
	steps := []string{"Step 1", "Step 2"}
	tracker := NewProgressTracker(steps)

	// Invalid step numbers should be ignored
	tracker.StartStep(0)
	if tracker.currentStep != 0 {
		t.Errorf("expected currentStep to remain 0 for invalid step 0, got %d", tracker.currentStep)
	}

	tracker.StartStep(5)
	if tracker.currentStep != 0 {
		t.Errorf("expected currentStep to remain 0 for out-of-range step, got %d", tracker.currentStep)
	}
}

func TestProgressTracker_SetItemCount(t *testing.T) {
	tracker := NewProgressTracker([]string{"Step 1"})
	tracker.SetItemCount(10)

	if tracker.totalItems != 10 {
		t.Errorf("expected totalItems to be 10, got %d", tracker.totalItems)
	}
}

func TestProgressTracker_NextItem(t *testing.T) {
	tracker := NewProgressTracker([]string{"Step 1"})
	tracker.SetItemCount(5)

	tracker.NextItem("First task")
	if tracker.currentItem != 1 {
		t.Errorf("expected currentItem to be 1, got %d", tracker.currentItem)
	}
	if tracker.currentTask != "First task" {
		t.Errorf("expected currentTask to be 'First task', got '%s'", tracker.currentTask)
	}

	tracker.NextItem("Second task")
	if tracker.currentItem != 2 {
		t.Errorf("expected currentItem to be 2, got %d", tracker.currentItem)
	}
}

func TestFormatProgress(t *testing.T) {
	tests := []struct {
		name     string
		current  int
		total    int
		msg      string
		expected string
	}{
		{
			name:     "with counter",
			current:  3,
			total:    10,
			msg:      "Processing",
			expected: "[3/10] Processing",
		},
		{
			name:     "no counter when total is 0",
			current:  1,
			total:    0,
			msg:      "Processing",
			expected: "Processing",
		},
		{
			name:     "no counter when current is 0",
			current:  0,
			total:    10,
			msg:      "Processing",
			expected: "Processing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatProgress(tt.current, tt.total, tt.msg)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestFormatProgressWithIcon(t *testing.T) {
	tests := []struct {
		name     string
		icon     string
		current  int
		total    int
		msg      string
		contains []string
	}{
		{
			name:     "with counter and icon",
			icon:     "✓",
			current:  5,
			total:    10,
			msg:      "Done",
			contains: []string{"✓", "[5/10]", "Done"},
		},
		{
			name:     "icon only when no counter",
			icon:     "✖",
			current:  0,
			total:    0,
			msg:      "Error",
			contains: []string{"✖", "Error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatProgressWithIcon(tt.icon, tt.current, tt.total, tt.msg)
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain '%s', got '%s'", s, result)
				}
			}
		})
	}
}

func TestProgressBarModel_Init(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)

	m := newProgressBarModel("Testing progress", updateChan, doneChan)
	cmd := m.Init()

	if cmd == nil {
		t.Error("expected Init to return a command batch")
	}

	close(updateChan)
	close(doneChan)
}

func TestProgressBarModel_Update_Quit(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)

	tests := []struct {
		name string
		key  string
	}{
		{"quit with q", "q"},
		{"quit with esc", "esc"},
		{"quit with ctrl+c", "ctrl+c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newProgressBarModel("Testing", updateChan, doneChan)

			var msg tea.KeyMsg
			switch tt.key {
			case "esc":
				msg = tea.KeyMsg{Type: tea.KeyEsc}
			case "ctrl+c":
				msg = tea.KeyMsg{Type: tea.KeyCtrlC}
			default:
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.key)}
			}

			_, cmd := m.Update(msg)

			if cmd == nil {
				t.Error("expected tea.Quit command")
			}
		})
	}

	close(updateChan)
	close(doneChan)
}

func TestProgressBarModel_Update_WindowSize(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Testing", updateChan, doneChan)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedModel, _ := m.Update(msg)

	pm := updatedModel.(progressBarModel)
	// Width should be msg.Width - 10, capped at 60
	expectedWidth := 60
	if pm.width != expectedWidth {
		t.Errorf("expected width to be %d, got %d", expectedWidth, pm.width)
	}
}

func TestProgressBarModel_Update_WindowSize_Small(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Testing", updateChan, doneChan)

	msg := tea.WindowSizeMsg{Width: 25, Height: 50}
	updatedModel, _ := m.Update(msg)

	pm := updatedModel.(progressBarModel)
	// Width should be at least 20
	if pm.width < 20 {
		t.Errorf("expected width to be at least 20, got %d", pm.width)
	}
}

func TestProgressBarModel_Update_ProgressUpdate(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(doneChan)

	m := newProgressBarModel("Testing", updateChan, doneChan)

	update := progressUpdate{percent: 0.5, message: "Halfway there"}
	updatedModel, _ := m.Update(update)

	pm := updatedModel.(progressBarModel)
	if pm.percent != 0.5 {
		t.Errorf("expected percent to be 0.5, got %f", pm.percent)
	}
	if pm.message != "Halfway there" {
		t.Errorf("expected message to be 'Halfway there', got '%s'", pm.message)
	}

	close(updateChan)
}

func TestProgressBarModel_Update_Done(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Testing", updateChan, doneChan)

	doneMsg := progressDoneMsg{err: nil}
	updatedModel, cmd := m.Update(doneMsg)

	pm := updatedModel.(progressBarModel)
	if !pm.done {
		t.Error("expected done to be true")
	}
	if pm.err != nil {
		t.Error("expected no error")
	}
	if cmd == nil {
		t.Error("expected tea.Quit command")
	}
}

func TestProgressBarModel_Update_DoneWithError(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Testing", updateChan, doneChan)

	testErr := errors.New("something went wrong")
	doneMsg := progressDoneMsg{err: testErr}
	updatedModel, _ := m.Update(doneMsg)

	pm := updatedModel.(progressBarModel)
	if !pm.done {
		t.Error("expected done to be true")
	}
	if pm.err == nil {
		t.Error("expected error to be set")
	}
	if pm.err.Error() != "something went wrong" {
		t.Errorf("expected error message 'something went wrong', got '%s'", pm.err.Error())
	}
}

func TestProgressBarModel_View(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Installing packages", updateChan, doneChan)

	view := m.View()

	if !strings.Contains(view, "Installing packages") {
		t.Errorf("expected view to contain 'Installing packages', got '%s'", view)
	}
}

func TestProgressBarModel_View_WithProgress(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Installing", updateChan, doneChan)
	m.percent = 0.5

	view := m.View()

	if !strings.Contains(view, "Installing") {
		t.Errorf("expected view to contain 'Installing', got '%s'", view)
	}
	// Progress bar should be rendered when percent > 0
	if view == "" {
		t.Error("expected non-empty view with progress")
	}
}

func TestProgressBarModel_View_Done(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Installing", updateChan, doneChan)
	m.done = true

	view := m.View()

	if view != "" {
		t.Errorf("expected empty view when done successfully, got '%s'", view)
	}
}

func TestProgressBarModel_View_DoneWithError(t *testing.T) {
	updateChan := make(chan progressUpdate, 10)
	doneChan := make(chan error, 1)
	defer close(updateChan)
	defer close(doneChan)

	m := newProgressBarModel("Installing", updateChan, doneChan)
	m.done = true
	m.err = errors.New("network error")

	view := m.View()

	if !strings.Contains(view, "Installing") {
		t.Errorf("expected view to contain message, got '%s'", view)
	}
	if !strings.Contains(view, "network error") {
		t.Errorf("expected view to contain error message, got '%s'", view)
	}
	if !strings.Contains(view, "✖") {
		t.Errorf("expected view to contain error icon, got '%s'", view)
	}
}
