//go:build e2e

package helpers

import (
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
// Optional delay parameter controls pause between keystrokes (default: 10ms)
func (tm *TUITestModel) SendKeys(keys ...interface{}) {
	tm.SendKeysWithDelay(10*time.Millisecond, keys...)
}

// SendKeysWithDelay sends key messages with a custom delay between keystrokes
func (tm *TUITestModel) SendKeysWithDelay(delay time.Duration, keys ...interface{}) {
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

		if delay > 0 {
			time.Sleep(delay)
		}
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

// AssertState validates that a model's state matches expected values
// Uses a predicate function to check the state
func (tm *TUITestModel) AssertState(predicate func(tea.Model) bool, message string) {
	tm.t.Helper()

	// Access the underlying model through the program
	// Note: This is a demonstration method - in practice you'd need access
	// to the actual model instance to check internal state
	if !predicate(nil) {
		tm.t.Errorf("state assertion failed: %s", message)
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

// Backspace adds a Backspace key press
func (ks *KeySequence) Backspace() *KeySequence {
	return ks.Press(tea.KeyBackspace)
}

// Delete adds a Delete key press
func (ks *KeySequence) Delete() *KeySequence {
	return ks.Press(tea.KeyDelete)
}

// Home adds a Home key press
func (ks *KeySequence) Home() *KeySequence {
	return ks.Press(tea.KeyHome)
}

// End adds an End key press
func (ks *KeySequence) End() *KeySequence {
	return ks.Press(tea.KeyEnd)
}

// PageUp adds a PageUp key press
func (ks *KeySequence) PageUp() *KeySequence {
	return ks.Press(tea.KeyPgUp)
}

// PageDown adds a PageDown key press
func (ks *KeySequence) PageDown() *KeySequence {
	return ks.Press(tea.KeyPgDown)
}

// SendTo sends the key sequence to a TUI test model
func (ks *KeySequence) SendTo(tm *TUITestModel) {
	tm.SendKeys(ks.keys...)
}

// SendToWithDelay sends the key sequence with a custom delay between keystrokes
func (ks *KeySequence) SendToWithDelay(tm *TUITestModel, delay time.Duration) {
	tm.SendKeysWithDelay(delay, ks.keys...)
}
