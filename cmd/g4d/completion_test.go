package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompletionCommand(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		contains string
	}{
		{
			name:     "bash completion generates output",
			shell:    "bash",
			contains: "bash",
		},
		{
			name:     "zsh completion generates output",
			shell:    "zsh",
			contains: "zsh",
		},
		{
			name:     "fish completion generates output",
			shell:    "fish",
			contains: "fish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"completion", tt.shell})

			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("completion %s failed: %v", tt.shell, err)
			}

			output := buf.String()
			if output == "" {
				t.Errorf("completion %s produced empty output", tt.shell)
			}
			if !strings.Contains(strings.ToLower(output), tt.contains) {
				t.Errorf("completion %s output does not contain %q", tt.shell, tt.contains)
			}
		})
	}
}

func TestCompletionCommand_InvalidShell(t *testing.T) {
	rootCmd.SetArgs([]string{"completion", "powershell"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for invalid shell argument, got nil")
	}
}

func TestCompletionCommand_NoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{"completion"})

	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for missing shell argument, got nil")
	}
}
