package ui

import (
	"os"
	"sync"

	"github.com/mattn/go-isatty"
)

var (
	contextMu      sync.RWMutex
	nonInteractive bool
)

// SetNonInteractive sets the global non-interactive mode.
// This should be called from the CLI layer when --non-interactive or -y is used.
func SetNonInteractive(value bool) {
	contextMu.Lock()
	defer contextMu.Unlock()
	nonInteractive = value
}

// IsInteractive returns true if the tool should run in interactive mode.
// It checks:
// 1. Explicit non-interactive flag was set
// 2. stdin is a TTY
func IsInteractive() bool {
	contextMu.RLock()
	defer contextMu.RUnlock()

	if nonInteractive {
		return false
	}

	// Check if stdin is a terminal
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// IsNonInteractive returns true if running in non-interactive mode.
func IsNonInteractive() bool {
	return !IsInteractive()
}

// RunContext provides context about the current execution environment.
type RunContext struct {
	Interactive  bool
	HasConfig    bool
	ConfigPath   string
	DotfilesPath string
}

// NewRunContext creates a new RunContext with the current state.
func NewRunContext() *RunContext {
	return &RunContext{
		Interactive: IsInteractive(),
	}
}

// WithConfig sets the config information on the context.
func (c *RunContext) WithConfig(configPath, dotfilesPath string) *RunContext {
	c.HasConfig = configPath != ""
	c.ConfigPath = configPath
	c.DotfilesPath = dotfilesPath
	return c
}
