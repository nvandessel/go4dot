//go:build e2e

package helpers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// VHSConfig configures VHS tape execution
type VHSConfig struct {
	TapePath     string
	OutputPath   string
	GoldenPath   string
	UpdateGolden bool
}

// RunVHSTape executes a VHS tape file and captures output
func RunVHSTape(t *testing.T, cfg VHSConfig) error {
	t.Helper()

	// Check if VHS is installed
	if !IsVHSInstalled() {
		t.Skip("VHS not installed, skipping visual test. Run 'make install-vhs' to install.")
	}

	// Ensure tape file exists
	if _, err := os.Stat(cfg.TapePath); os.IsNotExist(err) {
		return fmt.Errorf("tape file not found: %s", cfg.TapePath)
	}

	// Create output directory if it doesn't exist
	if cfg.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.OutputPath), 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	// Run VHS tape
	cmd := exec.Command("vhs", cfg.TapePath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vhs execution failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// If no golden file comparison needed, we're done
	if cfg.GoldenPath == "" {
		return nil
	}

	// Read output file
	if cfg.OutputPath == "" {
		return fmt.Errorf("output path required for golden file comparison")
	}

	output, err := os.ReadFile(cfg.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	// Update golden file if requested
	if cfg.UpdateGolden {
		if err := os.MkdirAll(filepath.Dir(cfg.GoldenPath), 0755); err != nil {
			return fmt.Errorf("failed to create golden directory: %w", err)
		}
		if err := os.WriteFile(cfg.GoldenPath, output, 0644); err != nil {
			return fmt.Errorf("failed to write golden file: %w", err)
		}
		t.Logf("Updated golden file: %s", cfg.GoldenPath)
		return nil
	}

	// Compare with golden file
	return CompareWithGolden(t, output, cfg.GoldenPath)
}

// CompareWithGolden compares output with golden file
func CompareWithGolden(t *testing.T, actual []byte, goldenPath string) error {
	t.Helper()

	// Read golden file
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("golden file not found: %s (run with UPDATE_GOLDEN=1 to create)", goldenPath)
		}
		return fmt.Errorf("failed to read golden file: %w", err)
	}

	// Normalize line endings for comparison
	actualNormalized := normalizeOutput(string(actual))
	goldenNormalized := normalizeOutput(string(golden))

	if actualNormalized != goldenNormalized {
		// Generate diff for debugging
		diffPath := strings.TrimSuffix(goldenPath, filepath.Ext(goldenPath)) + "_diff.txt"
		diff := generateDiff(goldenNormalized, actualNormalized)
		if err := os.WriteFile(diffPath, []byte(diff), 0644); err == nil {
			t.Logf("Diff written to: %s", diffPath)
		}
		return fmt.Errorf("output does not match golden file\nExpected: %s\nDiff: %s", goldenPath, diffPath)
	}

	return nil
}

// IsVHSInstalled checks if VHS is available in PATH
func IsVHSInstalled() bool {
	_, err := exec.LookPath("vhs")
	return err == nil
}

// normalizeOutput normalizes line endings and trailing whitespace
func normalizeOutput(s string) string {
	// Convert CRLF to LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Trim trailing whitespace from each line
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	// Join lines and trim trailing newlines for robust comparison
	result := strings.Join(lines, "\n")
	return strings.TrimRight(result, "\n")
}

// generateDiff creates a simple diff between expected and actual output
func generateDiff(expected, actual string) string {
	var diff strings.Builder
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	diff.WriteString("=== DIFF ===\n")
	diff.WriteString(fmt.Sprintf("Expected lines: %d\n", len(expectedLines)))
	diff.WriteString(fmt.Sprintf("Actual lines: %d\n\n", len(actualLines)))

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	for i := 0; i < maxLines; i++ {
		var expectedLine, actualLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actualLine = actualLines[i]
		}

		if expectedLine != actualLine {
			diff.WriteString(fmt.Sprintf("Line %d:\n", i+1))
			diff.WriteString(fmt.Sprintf("  - Expected: %q\n", expectedLine))
			diff.WriteString(fmt.Sprintf("  + Actual:   %q\n\n", actualLine))
		}
	}

	return diff.String()
}

// CleanupVHSOutputs removes VHS output files for a test
func CleanupVHSOutputs(paths ...string) {
	for _, path := range paths {
		_ = os.Remove(path) // Ignore errors during cleanup
	}
}

// RunVHSTapeInContainer executes a VHS tape inside a Docker container
// This provides complete isolation and ensures consistent VHS execution
func RunVHSTapeInContainer(t *testing.T, container *DockerTestContainer, cfg VHSConfig) error {
	t.Helper()

	// Ensure tape file exists locally
	if _, err := os.Stat(cfg.TapePath); os.IsNotExist(err) {
		return fmt.Errorf("tape file not found: %s", cfg.TapePath)
	}

	// Run VHS in container
	vhsCfg := VHSTapeConfig{
		TapePath:   cfg.TapePath,
		OutputPath: cfg.OutputPath,
	}

	if err := container.RunVHSTape(vhsCfg); err != nil {
		return fmt.Errorf("failed to run VHS tape in container: %w", err)
	}

	// If no golden file comparison needed, we're done
	if cfg.GoldenPath == "" {
		return nil
	}

	// Read output file
	if cfg.OutputPath == "" {
		return fmt.Errorf("output path required for golden file comparison")
	}

	output, err := os.ReadFile(cfg.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	// Update golden file if requested
	if cfg.UpdateGolden {
		if err := os.MkdirAll(filepath.Dir(cfg.GoldenPath), 0755); err != nil {
			return fmt.Errorf("failed to create golden directory: %w", err)
		}
		if err := os.WriteFile(cfg.GoldenPath, output, 0644); err != nil {
			return fmt.Errorf("failed to write golden file: %w", err)
		}
		t.Logf("Updated golden file: %s", cfg.GoldenPath)
		return nil
	}

	// Compare with golden file
	return CompareWithGolden(t, output, cfg.GoldenPath)
}
