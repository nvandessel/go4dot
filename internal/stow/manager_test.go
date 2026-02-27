package stow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

func TestIsStowInstalled(t *testing.T) {
	// This test depends on system state, but we can at least check it runs
	installed := IsStowInstalled()
	t.Logf("Stow installed: %v", installed)
}

func TestValidateStow(t *testing.T) {
	// Skip if stow is not installed
	if !IsStowInstalled() {
		t.Skip("Stow is not installed, skipping validation test")
	}

	err := ValidateStow()
	if err != nil {
		t.Errorf("ValidateStow() failed: %v", err)
	}
}

func TestStowConfigs(t *testing.T) {
	// Skip if stow is not installed
	if !IsStowInstalled() {
		t.Skip("Stow is not installed, skipping stow test")
	}

	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create a fake config directory
	testConfigDir := filepath.Join(tmpDir, "testconfig")
	err := os.MkdirAll(filepath.Join(testConfigDir, ".config"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a test file
	testFile := filepath.Join(testConfigDir, ".config", "test.conf")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create config items
	configs := []config.ConfigItem{
		{
			Name: "testconfig",
			Path: "testconfig",
		},
		{
			Name: "nonexistent",
			Path: "nonexistent",
		},
	}

	opts := StowOptions{
		DryRun: true, // Don't actually create symlinks in tests
	}

	// Test stowing
	result := StowConfigs(tmpDir, configs, opts)

	// Should have one skipped (nonexistent) and one attempt
	if len(result.Skipped) != 1 {
		t.Errorf("Expected 1 skipped config, got %d", len(result.Skipped))
	}

	if result.Skipped[0] != "nonexistent" {
		t.Errorf("Expected 'nonexistent' to be skipped, got %s", result.Skipped[0])
	}
}

func TestUnstowConfigs(t *testing.T) {
	// Skip if stow is not installed
	if !IsStowInstalled() {
		t.Skip("Stow is not installed, skipping unstow test")
	}

	tmpDir := t.TempDir()

	configs := []config.ConfigItem{
		{
			Name: "testconfig",
			Path: "testconfig",
		},
	}

	opts := StowOptions{
		DryRun: true,
	}

	result := UnstowConfigs(tmpDir, configs, opts)

	// Unstowing non-existent config shouldn't cause test failure
	// It should either succeed or fail gracefully
	t.Logf("Unstow result: success=%d, failed=%d", len(result.Success), len(result.Failed))
}

func TestRestowConfigs(t *testing.T) {
	// Skip if stow is not installed
	if !IsStowInstalled() {
		t.Skip("Stow is not installed, skipping restow test")
	}

	tmpDir := t.TempDir()

	// Create test config
	testConfigDir := filepath.Join(tmpDir, "testconfig")
	err := os.MkdirAll(filepath.Join(testConfigDir, ".config"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	configs := []config.ConfigItem{
		{
			Name: "testconfig",
			Path: "testconfig",
		},
		{
			Name: "missing",
			Path: "missing",
		},
	}

	opts := StowOptions{
		DryRun: true,
	}

	result := RestowConfigs(tmpDir, configs, opts)

	// Should have one skipped (missing)
	if len(result.Skipped) != 1 {
		t.Errorf("Expected 1 skipped config, got %d", len(result.Skipped))
	}
}

func TestStowResult(t *testing.T) {
	result := &StowResult{
		Success: []string{"config1", "config2"},
		Failed: []StowError{
			{ConfigName: "config3", Error: os.ErrNotExist},
		},
		Skipped: []string{"config4"},
	}

	if len(result.Success) != 2 {
		t.Errorf("Expected 2 successful configs, got %d", len(result.Success))
	}

	if len(result.Failed) != 1 {
		t.Errorf("Expected 1 failed config, got %d", len(result.Failed))
	}

	if len(result.Skipped) != 1 {
		t.Errorf("Expected 1 skipped config, got %d", len(result.Skipped))
	}
}

func TestStowOptionsProgressCallback(t *testing.T) {
	// Skip if stow is not installed
	if !IsStowInstalled() {
		t.Skip("Stow is not installed, skipping progress callback test")
	}

	tmpDir := t.TempDir()
	testConfigDir := filepath.Join(tmpDir, "testconfig")
	err := os.MkdirAll(filepath.Join(testConfigDir, ".config"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	var progressMessages []string
	opts := StowOptions{
		DryRun: true,
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	_ = Stow(tmpDir, "testconfig", opts)
	// Don't fail on error since we're in dry-run mode with a test directory

	// Should have received at least one progress message
	if len(progressMessages) == 0 {
		t.Error("Expected at least one progress message")
	}

	t.Logf("Received %d progress messages", len(progressMessages))
}

func TestStow_RejectsInjection(t *testing.T) {
	tests := []struct {
		name       string
		configName string
		wantErr    bool
	}{
		{name: "valid name", configName: "vim", wantErr: false},
		{name: "flag injection --target", configName: "--target=/etc", wantErr: true},
		{name: "flag injection -D", configName: "-D", wantErr: true},
		{name: "shell metachar", configName: "vim;rm -rf /", wantErr: true},
		{name: "path traversal", configName: "../etc/passwd", wantErr: true},
		{name: "double dash only", configName: "--", wantErr: true},
	}

	// Save and restore the original commander
	origCommander := CurrentCommander
	defer func() { CurrentCommander = origCommander }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommander{}
			CurrentCommander = mock

			tmpDir := t.TempDir()

			// Create config directory for valid case
			if !tt.wantErr {
				err := os.MkdirAll(filepath.Join(tmpDir, tt.configName, ".config"), 0755)
				if err != nil {
					t.Fatalf("Failed to create test directory: %v", err)
				}
			}

			opts := StowOptions{DryRun: true}

			// Test StowWithCount
			err := StowWithCount(tmpDir, tt.configName, 1, 1, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("StowWithCount(%q) error = %v, wantErr %v", tt.configName, err, tt.wantErr)
			}

			// Test UnstowWithCount
			err = UnstowWithCount(tmpDir, tt.configName, 1, 1, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnstowWithCount(%q) error = %v, wantErr %v", tt.configName, err, tt.wantErr)
			}

			// Test RestowWithCount
			err = RestowWithCount(tmpDir, tt.configName, 1, 1, opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestowWithCount(%q) error = %v, wantErr %v", tt.configName, err, tt.wantErr)
			}
		})
	}
}

func TestStow_UsesDoubleDashSeparator(t *testing.T) {
	// Verify that the -- separator appears before the config name in args
	origCommander := CurrentCommander
	defer func() { CurrentCommander = origCommander }()

	mock := &MockCommander{}
	CurrentCommander = mock

	tmpDir := t.TempDir()
	configName := "vim"

	// Create config directory
	err := os.MkdirAll(filepath.Join(tmpDir, configName, ".config"), 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	opts := StowOptions{DryRun: true}

	_ = StowWithCount(tmpDir, configName, 1, 1, opts)

	// Verify -- is the second-to-last arg and configName is the last arg
	args := mock.LastArgs
	if len(args) < 2 {
		t.Fatalf("Expected at least 2 args, got %d: %v", len(args), args)
	}

	lastArg := args[len(args)-1]
	secondToLast := args[len(args)-2]

	if secondToLast != "--" {
		t.Errorf("Expected second-to-last arg to be '--', got %q (args: %v)", secondToLast, args)
	}
	if lastArg != configName {
		t.Errorf("Expected last arg to be %q, got %q (args: %v)", configName, lastArg, args)
	}

	// Also verify no os.Getenv("HOME") leak: -t should have a real path, not empty
	for i, arg := range args {
		if arg == "-t" && i+1 < len(args) {
			if args[i+1] == "" {
				t.Error("Target directory (-t) should not be empty; os.UserHomeDir() should provide a value")
			}
			break
		}
	}
}

func TestUnstow_RejectsInjection(t *testing.T) {
	origCommander := CurrentCommander
	defer func() { CurrentCommander = origCommander }()

	mock := &MockCommander{}
	CurrentCommander = mock

	tmpDir := t.TempDir()
	opts := StowOptions{DryRun: true}

	// Flag injection should be rejected
	err := UnstowWithCount(tmpDir, "--target=/etc", 1, 1, opts)
	if err == nil {
		t.Error("UnstowWithCount should reject --target=/etc")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid config name") {
		t.Errorf("Expected 'invalid config name' error, got: %v", err)
	}
}

func TestRestow_RejectsInjection(t *testing.T) {
	origCommander := CurrentCommander
	defer func() { CurrentCommander = origCommander }()

	mock := &MockCommander{}
	CurrentCommander = mock

	tmpDir := t.TempDir()
	opts := StowOptions{DryRun: true}

	// Flag injection should be rejected
	err := RestowWithCount(tmpDir, "-D", 1, 1, opts)
	if err == nil {
		t.Error("RestowWithCount should reject -D")
	}
	if err != nil && !strings.Contains(err.Error(), "invalid config name") {
		t.Errorf("Expected 'invalid config name' error, got: %v", err)
	}
}
