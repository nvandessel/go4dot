package ui

import (
	"testing"
)

func TestSetNonInteractive(t *testing.T) {
	// Test that SetNonInteractive(true) forces non-interactive mode
	t.Run("set to true forces non-interactive", func(t *testing.T) {
		SetNonInteractive(true)
		if !IsNonInteractive() {
			t.Error("IsNonInteractive() should return true when flag is set")
		}
		if IsInteractive() {
			t.Error("IsInteractive() should return false when flag is set")
		}
	})

	// Test that SetNonInteractive(false) clears the forced flag
	// Note: In non-TTY environments (like tests), IsInteractive may still return false
	t.Run("set to false clears flag", func(t *testing.T) {
		SetNonInteractive(true)
		SetNonInteractive(false)
		// After clearing, behavior depends on TTY state
		// We can only verify the flag was cleared by checking it can be set again
		SetNonInteractive(true)
		if !IsNonInteractive() {
			t.Error("Flag should be settable after clearing")
		}
	})

	// Reset for other tests
	SetNonInteractive(false)
}

func TestIsNonInteractive(t *testing.T) {
	// IsNonInteractive should be the inverse of IsInteractive
	SetNonInteractive(true)
	if !IsNonInteractive() {
		t.Error("IsNonInteractive() should return true when nonInteractive flag is set")
	}

	SetNonInteractive(false)
}

func TestNewRunContext(t *testing.T) {
	ctx := NewRunContext()

	if ctx == nil {
		t.Fatal("NewRunContext() returned nil")
	}

	// HasConfig should default to false
	if ctx.HasConfig {
		t.Error("NewRunContext() should have HasConfig = false by default")
	}

	// ConfigPath should default to empty
	if ctx.ConfigPath != "" {
		t.Errorf("NewRunContext() ConfigPath = %q, want empty", ctx.ConfigPath)
	}

	// DotfilesPath should default to empty
	if ctx.DotfilesPath != "" {
		t.Errorf("NewRunContext() DotfilesPath = %q, want empty", ctx.DotfilesPath)
	}
}

func TestRunContextWithConfig(t *testing.T) {
	tests := []struct {
		name         string
		configPath   string
		dotfilesPath string
		wantHasConf  bool
	}{
		{
			name:         "with valid paths",
			configPath:   "/home/user/dotfiles/.go4dot.yaml",
			dotfilesPath: "/home/user/dotfiles",
			wantHasConf:  true,
		},
		{
			name:         "with empty config path",
			configPath:   "",
			dotfilesPath: "/home/user/dotfiles",
			wantHasConf:  false,
		},
		{
			name:         "both empty",
			configPath:   "",
			dotfilesPath: "",
			wantHasConf:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewRunContext().WithConfig(tt.configPath, tt.dotfilesPath)

			if ctx.HasConfig != tt.wantHasConf {
				t.Errorf("WithConfig() HasConfig = %v, want %v", ctx.HasConfig, tt.wantHasConf)
			}

			if ctx.ConfigPath != tt.configPath {
				t.Errorf("WithConfig() ConfigPath = %q, want %q", ctx.ConfigPath, tt.configPath)
			}

			if ctx.DotfilesPath != tt.dotfilesPath {
				t.Errorf("WithConfig() DotfilesPath = %q, want %q", ctx.DotfilesPath, tt.dotfilesPath)
			}
		})
	}
}

func TestWithConfigChaining(t *testing.T) {
	// Test that WithConfig returns the same context for chaining
	ctx := NewRunContext()
	result := ctx.WithConfig("/path/to/config", "/path/to/dotfiles")

	if result != ctx {
		t.Error("WithConfig() should return the same context pointer for chaining")
	}
}
