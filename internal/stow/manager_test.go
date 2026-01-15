package stow

import (
	"os"
	"path/filepath"
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
