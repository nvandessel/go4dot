//go:build e2e

package scenarios

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nvandessel/go4dot/test/e2e/helpers"
)

// TestDoctor_NoConfig tests doctor command without any dotfiles installed
func TestDoctor_NoConfig(t *testing.T) {
	t.Parallel() // Run tests in parallel for speed

	// Build binary
	binaryPath := helpers.BuildTestBinary(t)

	// Create container
	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath: binaryPath,
	})

	// Run doctor command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := container.ExecContext(ctx, "g4d", "doctor", "--non-interactive")

	// Without a config file, doctor should exit with an error
	if err == nil {
		t.Errorf("Expected doctor to fail without config file")
	}

	// Should indicate config not found
	if !strings.Contains(output, "could not find .go4dot.yaml") {
		t.Errorf("Expected error message about missing config, got: %s", output)
	}

	t.Logf("Doctor correctly failed without config:\n%s", output)
}

// TestDoctor_WithConfig tests doctor command with dotfiles installed
func TestDoctor_WithConfig(t *testing.T) {
	t.Parallel()

	// Build binary
	binaryPath := helpers.BuildTestBinary(t)

	projectRoot := helpers.GetProjectRoot(t)
	fixturesDir := filepath.Join(projectRoot, "test", "e2e", "fixtures", "dotfiles")

	// Create container with fixtures
	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
	})

	// Set up dotfiles directory with test config
	setupCommands := []string{
		"git config --global user.email 'test@example.com'",
		"git config --global user.name 'Test User'",
		"mkdir -p ~/dotfiles",
		"cp -r ~/fixtures/. ~/dotfiles/",  // Include hidden files
		"cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'",
	}

	for _, cmd := range setupCommands {
		output, err := container.Exec("bash", "-c", cmd)
		if err != nil {
			t.Fatalf("Failed to set up dotfiles: %v\nCommand: %s\nOutput: %s", err, cmd, output)
		}
	}

	// Verify setup
	lsOutput, _ := container.Exec("bash", "-c", "ls -la ~/dotfiles/")
	t.Logf("Dotfiles directory contents:\n%s", lsOutput)

	// Run doctor command
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := container.ExecContext(ctx, "bash", "-c", "cd ~/dotfiles && g4d doctor --non-interactive")
	if err != nil {
		// Doctor may exit with non-zero if issues found
		t.Logf("Doctor exited with error (may be expected): %v", err)
	}

	t.Logf("Doctor output (with config):\n%s", output)

	// Validate output contains expected sections
	expectedSections := []string{
		"Health Report",
		"Platform:",
		"Summary",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Doctor output missing expected section: %s", section)
		}
	}

	// Should have success indicators
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected success indicators (✓) in doctor output")
	}
}

// TestDoctor_AfterInstall tests doctor command after a full installation
func TestDoctor_AfterInstall(t *testing.T) {
	t.Parallel()

	// Build binary
	binaryPath := helpers.BuildTestBinary(t)

	projectRoot := helpers.GetProjectRoot(t)
	fixturesDir := filepath.Join(projectRoot, "test", "e2e", "fixtures", "dotfiles")

	// Create container with fixtures
	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
	})

	// Set up dotfiles directory
	setupCommands := []string{
		"git config --global user.email 'test@example.com'",
		"git config --global user.name 'Test User'",
		"mkdir -p ~/dotfiles",
		"cp -r ~/fixtures/. ~/dotfiles/",  // Include hidden files
		"cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'",
	}

	for _, cmd := range setupCommands {
		output, err := container.Exec("bash", "-c", cmd)
		if err != nil {
			t.Fatalf("Failed to set up dotfiles: %v\nCommand: %s\nOutput: %s", err, cmd, output)
		}
	}

	// Run stow (simpler than full install for this test)
	stowOutput, err := container.Exec("bash", "-c", "cd ~/dotfiles && g4d stow add vim zsh --non-interactive")
	if err != nil {
		t.Logf("Stow output: %s", stowOutput)
		// Continue even if stow has issues - we're testing doctor
	}

	// Run doctor after installation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := container.ExecContext(ctx, "bash", "-c", "cd ~/dotfiles && g4d doctor --non-interactive")
	if err != nil {
		t.Logf("Doctor exited with error: %v", err)
	}

	t.Logf("Doctor output (after install):\n%s", output)

	// Validate that doctor shows health report
	expectedSections := []string{
		"Health Report",
		"Platform:",
		"Summary",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Doctor output missing expected section: %s", section)
		}
	}

	// Should have at least some checks passing
	if !strings.Contains(output, "✓") {
		t.Errorf("Expected some checks to pass (✓) in doctor output")
	}
}

// TestDoctor_Verbose tests doctor command with verbose flag
func TestDoctor_Verbose(t *testing.T) {
	t.Parallel()

	// Build binary
	binaryPath := helpers.BuildTestBinary(t)

	projectRoot := helpers.GetProjectRoot(t)
	fixturesDir := filepath.Join(projectRoot, "test", "e2e", "fixtures", "dotfiles")

	// Create container with fixtures
	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
	})

	// Set up dotfiles
	setupCmd := "git config --global user.email 'test@example.com' && git config --global user.name 'Test User' && mkdir -p ~/dotfiles && cp -r ~/fixtures/. ~/dotfiles/ && cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'"
	if output, err := container.Exec("bash", "-c", setupCmd); err != nil {
		t.Fatalf("Failed to set up dotfiles: %v\nOutput: %s", err, output)
	}

	// Run doctor with verbose flag
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	output, err := container.ExecContext(ctx, "bash", "-c", "cd ~/dotfiles && g4d doctor --verbose --non-interactive")
	if err != nil {
		t.Logf("Doctor exited with error: %v", err)
	}

	t.Logf("Doctor verbose output:\n%s", output)

	// Verbose output should be more detailed than normal
	if len(output) < 100 {
		t.Errorf("Verbose output seems too short: %d bytes", len(output))
	}
}
