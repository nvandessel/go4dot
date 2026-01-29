//go:build e2e

package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

// TUITestModel wraps a teatest.TestModel with additional helper methods
type TUITestModel struct {
	*teatest.TestModel
	t *testing.T
}

// NewTUITestModel creates a new TUI test model with default terminal size
func NewTUITestModel(t *testing.T, model tea.Model, opts ...teatest.TestOption) *TUITestModel {
	t.Helper()

	// Apply default terminal size if not specified
	if len(opts) == 0 {
		opts = append(opts, teatest.WithInitialTermSize(80, 24))
	}

	tm := teatest.NewTestModel(t, model, opts...)
	return &TUITestModel{
		TestModel: tm,
		t:         t,
	}
}

// SendKeys sends a sequence of key messages to the TUI model
// Supports multiple key types: runes, special keys, and key combinations
func (tm *TUITestModel) SendKeys(keys ...interface{}) {
	tm.t.Helper()

	for _, key := range keys {
		switch k := key.(type) {
		case tea.KeyMsg:
			tm.Send(k)
		case tea.KeyType:
			tm.Send(tea.KeyMsg{Type: k})
		case rune:
			tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{k}})
		case string:
			// Send each character in the string
			for _, r := range k {
				tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
			}
		default:
			tm.t.Fatalf("unsupported key type: %T", key)
		}

		// Small delay between keystrokes to simulate realistic interaction
		time.Sleep(10 * time.Millisecond)
	}
}

// WaitForText waits for specific text to appear in the TUI output
// Failures are reported via testing.TB
func (tm *TUITestModel) WaitForText(text string, timeout ...time.Duration) {
	tm.t.Helper()

	timeoutDuration := 3 * time.Second
	if len(timeout) > 0 {
		timeoutDuration = timeout[0]
	}

	teatest.WaitFor(tm.t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), text)
	}, teatest.WithCheckInterval(100*time.Millisecond), teatest.WithDuration(timeoutDuration))
}

// WaitForNotText waits for specific text to disappear from the TUI output
// Failures are reported via testing.TB
func (tm *TUITestModel) WaitForNotText(text string, timeout ...time.Duration) {
	tm.t.Helper()

	timeoutDuration := 3 * time.Second
	if len(timeout) > 0 {
		timeoutDuration = timeout[0]
	}

	teatest.WaitFor(tm.t, tm.Output(), func(out []byte) bool {
		return !strings.Contains(string(out), text)
	}, teatest.WithCheckInterval(100*time.Millisecond), teatest.WithDuration(timeoutDuration))
}

// ReadOutput reads and collects output from the TUI model
// Note: This is a helper that tries to read what's available but teatest
// operates on streams, so this may not capture everything reliably.
// Prefer using WaitForText for most assertions.
func (tm *TUITestModel) ReadOutput() []byte {
	tm.t.Helper()

	// Create a buffer to capture output
	buf := make([]byte, 4096)
	n, _ := tm.Output().Read(buf)
	return buf[:n]
}

// GetOutputString returns the current TUI output as a string
// Note: See ReadOutput() caveat - prefer WaitForText for assertions
func (tm *TUITestModel) GetOutputString() string {
	return string(tm.ReadOutput())
}

// CompareGolden compares the current TUI output with a golden file
// If updateGolden is true, it updates the golden file instead of comparing
// Note: This captures a snapshot of output which may not be complete
func (tm *TUITestModel) CompareGolden(goldenPath string, updateGolden bool) error {
	tm.t.Helper()

	// Wait a bit for rendering
	time.Sleep(200 * time.Millisecond)

	// Get current output
	output := tm.ReadOutput()

	if updateGolden {
		return updateGoldenFile(goldenPath, output)
	}

	return CompareWithGolden(tm.t, output, goldenPath)
}

// AssertState validates that a model's state matches expected values
// Uses a predicate function to check the state
func AssertState(t *testing.T, model tea.Model, predicate func(tea.Model) bool, message string) {
	t.Helper()

	if !predicate(model) {
		t.Errorf("state assertion failed: %s", message)
	}
}

// AssertContains checks that the output contains the expected text
func (tm *TUITestModel) AssertContains(text string) {
	tm.t.Helper()

	output := tm.GetOutputString()
	if !strings.Contains(output, text) {
		tm.t.Errorf("expected output to contain %q\nActual output:\n%s", text, output)
	}
}

// AssertNotContains checks that the output does not contain the text
func (tm *TUITestModel) AssertNotContains(text string) {
	tm.t.Helper()

	output := tm.GetOutputString()
	if strings.Contains(output, text) {
		tm.t.Errorf("expected output to not contain %q\nActual output:\n%s", text, output)
	}
}

// WaitFinished waits for the TUI program to finish with a reasonable timeout
func (tm *TUITestModel) WaitFinished(timeout ...time.Duration) {
	tm.t.Helper()

	timeoutDuration := 3 * time.Second
	if len(timeout) > 0 {
		timeoutDuration = timeout[0]
	}

	tm.TestModel.WaitFinished(tm.t, teatest.WithFinalTimeout(timeoutDuration))
}

// updateGoldenFile writes output to a golden file
func updateGoldenFile(goldenPath string, output []byte) error {
	return updateGoldenFileWithNormalization(goldenPath, output)
}

// updateGoldenFileWithNormalization writes normalized output to a golden file
func updateGoldenFileWithNormalization(goldenPath string, output []byte) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(goldenPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create golden file directory: %w", err)
	}

	// Normalize the output before writing
	normalized := normalizeOutput(string(output))
	return writeGoldenFile(goldenPath, []byte(normalized))
}

// writeGoldenFile is a helper to write golden files
func writeGoldenFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// KeySequence represents a sequence of keys to send to the TUI
type KeySequence struct {
	keys []interface{}
}

// NewKeySequence creates a new key sequence builder
func NewKeySequence() *KeySequence {
	return &KeySequence{
		keys: make([]interface{}, 0),
	}
}

// Type adds text characters to the sequence
func (ks *KeySequence) Type(text string) *KeySequence {
	ks.keys = append(ks.keys, text)
	return ks
}

// Press adds a special key to the sequence
func (ks *KeySequence) Press(keyType tea.KeyType) *KeySequence {
	ks.keys = append(ks.keys, keyType)
	return ks
}

// Key adds a specific key message to the sequence
func (ks *KeySequence) Key(key tea.KeyMsg) *KeySequence {
	ks.keys = append(ks.keys, key)
	return ks
}

// Rune adds a single rune to the sequence
func (ks *KeySequence) Rune(r rune) *KeySequence {
	ks.keys = append(ks.keys, r)
	return ks
}

// Enter adds an Enter key press
func (ks *KeySequence) Enter() *KeySequence {
	return ks.Press(tea.KeyEnter)
}

// Esc adds an Escape key press
func (ks *KeySequence) Esc() *KeySequence {
	return ks.Press(tea.KeyEsc)
}

// Tab adds a Tab key press
func (ks *KeySequence) Tab() *KeySequence {
	return ks.Press(tea.KeyTab)
}

// Space adds a Space key press
func (ks *KeySequence) Space() *KeySequence {
	return ks.Press(tea.KeySpace)
}

// Up adds an Up arrow key press
func (ks *KeySequence) Up() *KeySequence {
	return ks.Press(tea.KeyUp)
}

// Down adds a Down arrow key press
func (ks *KeySequence) Down() *KeySequence {
	return ks.Press(tea.KeyDown)
}

// Left adds a Left arrow key press
func (ks *KeySequence) Left() *KeySequence {
	return ks.Press(tea.KeyLeft)
}

// Right adds a Right arrow key press
func (ks *KeySequence) Right() *KeySequence {
	return ks.Press(tea.KeyRight)
}

// Build returns the key sequence
func (ks *KeySequence) Build() []interface{} {
	return ks.keys
}

// SendTo sends the key sequence to a TUI test model
func (ks *KeySequence) SendTo(tm *TUITestModel) {
	tm.SendKeys(ks.keys...)
}
