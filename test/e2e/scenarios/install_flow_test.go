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

// TestInstall_BasicFlow tests the basic install workflow
func TestInstall_BasicFlow(t *testing.T) {
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

	// Set up dotfiles repository
	setupCommands := []string{
		"git config --global user.email 'test@example.com'",
		"git config --global user.name 'Test User'",
		"mkdir -p ~/dotfiles",
		"cp -r ~/fixtures/. ~/dotfiles/",
		"cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'",
	}

	for _, cmd := range setupCommands {
		if output, err := container.Exec("bash", "-c", cmd); err != nil {
			t.Fatalf("Failed to set up dotfiles: %v\nCommand: %s\nOutput: %s", err, cmd, output)
		}
	}

	// Run stow add (simpler than full install for testing)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	output, err := container.ExecContext(ctx, "bash", "-c", "cd ~/dotfiles && g4d stow add vim --non-interactive")
	if err != nil {
		t.Logf("Stow output: %s", output)
	}

	t.Logf("Stow add output:\n%s", output)

	// Verify symlink was created
	lsOutput, err := container.Exec("bash", "-c", "ls -la ~/.vimrc")
	if err != nil {
		t.Errorf("Expected .vimrc symlink to exist: %v", err)
	} else {
		if !strings.Contains(lsOutput, "->") {
			t.Errorf("Expected .vimrc to be a symlink, got: %s", lsOutput)
		}
		t.Logf("Symlink created successfully: %s", lsOutput)
	}
}

// TestInstall_MultipleConfigs tests installing multiple configs
func TestInstall_MultipleConfigs(t *testing.T) {
	t.Parallel()

	binaryPath := helpers.BuildTestBinary(t)
	projectRoot := helpers.GetProjectRoot(t)
	fixturesDir := filepath.Join(projectRoot, "test", "e2e", "fixtures", "dotfiles")

	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
	})

	// Setup
	setupCmd := "git config --global user.email 'test@example.com' && git config --global user.name 'Test User' && mkdir -p ~/dotfiles && cp -r ~/fixtures/. ~/dotfiles/ && cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'"
	if output, err := container.Exec("bash", "-c", setupCmd); err != nil {
		t.Fatalf("Failed to set up: %v\nOutput: %s", err, output)
	}

	// Install multiple configs (one at a time)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	for _, cfg := range []string{"vim", "zsh"} {
		output, err := container.ExecContext(ctx, "bash", "-c", "cd ~/dotfiles && g4d stow add "+cfg+" --non-interactive")
		if err != nil {
			t.Logf("Stow output for %s: %s", cfg, output)
		}
		t.Logf("Installed %s:\n%s", cfg, output)
	}

	// Verify both symlinks exist
	configs := map[string]string{
		"vim": "~/.vimrc",
		"zsh": "~/.zshrc",
	}

	for name, path := range configs {
		lsOutput, err := container.Exec("bash", "-c", "ls -la "+path)
		if err != nil {
			t.Errorf("Expected %s symlink (%s) to exist: %v", name, path, err)
		} else if !strings.Contains(lsOutput, "->") {
			t.Errorf("Expected %s to be a symlink, got: %s", path, lsOutput)
		} else {
			t.Logf("✓ %s symlink created: %s", name, path)
		}
	}
}

// TestInstall_SymlinkVerification tests that symlinks are created correctly
func TestInstall_SymlinkVerification(t *testing.T) {
	t.Parallel()

	binaryPath := helpers.BuildTestBinary(t)
	projectRoot := helpers.GetProjectRoot(t)
	fixturesDir := filepath.Join(projectRoot, "test", "e2e", "fixtures", "dotfiles")

	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
	})

	// Setup
	setupCmd := "git config --global user.email 'test@example.com' && git config --global user.name 'Test User' && mkdir -p ~/dotfiles && cp -r ~/fixtures/. ~/dotfiles/ && cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'"
	if output, err := container.Exec("bash", "-c", setupCmd); err != nil {
		t.Fatalf("Failed to set up: %v\nOutput: %s", err, output)
	}

	// Install config
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	stowOutput, err := container.ExecContext(ctx, "bash", "-c", "cd ~/dotfiles && g4d stow add vim --non-interactive")
	t.Logf("Stow add output: %s", stowOutput)
	if err != nil {
		t.Fatalf("Stow failed: %v", err)
	}

	// Verify symlink was created and points to correct target
	linkOutput, err := container.Exec("bash", "-c", "readlink ~/.vimrc")
	if err != nil {
		t.Errorf("Expected .vimrc symlink to exist: %v", err)
	} else {
		t.Logf("Symlink target: %s", linkOutput)
		if !strings.Contains(linkOutput, "dotfiles/vim/.vimrc") {
			t.Errorf("Expected symlink to point to dotfiles/vim/.vimrc, got: %s", linkOutput)
		}
	}

	// Verify symlink content is accessible
	contentOutput, err := container.Exec("bash", "-c", "head -1 ~/.vimrc")
	if err != nil {
		t.Errorf("Expected to read .vimrc content: %v", err)
	} else if !strings.Contains(contentOutput, "Test Vim Configuration") {
		t.Errorf("Expected vim config content, got: %s", contentOutput)
	}
}

// TestInstall_RestowOperation tests restowing configs
func TestInstall_RestowOperation(t *testing.T) {
	t.Parallel()

	binaryPath := helpers.BuildTestBinary(t)
	projectRoot := helpers.GetProjectRoot(t)
	fixturesDir := filepath.Join(projectRoot, "test", "e2e", "fixtures", "dotfiles")

	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
	})

	// Setup
	setupCmd := "git config --global user.email 'test@example.com' && git config --global user.name 'Test User' && mkdir -p ~/dotfiles && cp -r ~/fixtures/. ~/dotfiles/ && cd ~/dotfiles && git init && git add . && git commit -m 'Initial commit'"
	if output, err := container.Exec("bash", "-c", setupCmd); err != nil {
		t.Fatalf("Failed to set up: %v\nOutput: %s", err, output)
	}

	// Initial install
	ctx1, cancel1 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel1()
	initOutput, err := container.ExecContext(ctx1, "bash", "-c", "cd ~/dotfiles && g4d stow add vim --non-interactive")
	if err != nil {
		t.Fatalf("Initial stow failed: %v\nOutput: %s", err, initOutput)
	}
	t.Logf("Initial stow output: %s", initOutput)

	// Restow (refresh all configs)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel2()

	output, err := container.ExecContext(ctx2, "bash", "-c", "cd ~/dotfiles && g4d stow refresh --non-interactive")
	t.Logf("Restow operation output:\n%s", output)
	if err != nil {
		t.Fatalf("Restow failed: %v", err)
	}

	// Verify symlink still exists
	lsOutput, err := container.Exec("bash", "-c", "ls -la ~/.vimrc")
	if err != nil {
		t.Errorf("Expected .vimrc to still exist after restow: %v", err)
	} else if !strings.Contains(lsOutput, "->") {
		t.Errorf("Expected .vimrc to still be a symlink after restow")
	}
}
