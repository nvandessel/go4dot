package stow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

func TestDetectConflicts_DirectoryFolding_DataLoss(t *testing.T) {
	// Setup directories
	tmpDir := t.TempDir()
	dotfilesDir := filepath.Join(tmpDir, "dotfiles")
	homeDir := filepath.Join(tmpDir, "home")

	t.Setenv("HOME", homeDir)

	if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a config package "pkg" with a file
	pkgDir := filepath.Join(dotfilesDir, "pkg")
	if err := os.Mkdir(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(pkgDir, "file.txt")
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory symlink in home (directory folding)
	// ~/.config/pkg -> ~/dotfiles/pkg
	// Note: stow usually folds deeper, but let's simulate a direct dir link
	// effectively making ~/file.txt -> dotfiles/pkg/file.txt via ~/pkg/file.txt

	// Actually, let's match stow structure closer.
	// pkg/file.txt
	// target: ~/file.txt (if stowed at top level)

	// Create the "conflict" situation:
	// We manually create a symlink for the DIRECTORY 'pkg' in home to point to 'dotfiles/pkg'
	// But go4dot config expects to manage 'pkg/file.txt'.

	// Wait, DetectConflicts iterates files.
	// Config: Name: "pkg", Path: "pkg"
	// Files: "file.txt"
	// Target: ~/file.txt

	// If I have ~/file.txt as a symlink to .../dotfiles/pkg/file.txt, that's normal stowing.

	// The directory folding case is:
	// Config: Name "nvim", Path "nvim"
	// Files: ".config/nvim/init.lua"
	// In dotfiles: nvim/.config/nvim/init.lua
	// In home: ~/.config/nvim -> dotfiles/nvim/.config/nvim (The directory is linked)

	// Let's set up that structure.

	// 1. Config setup
	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "testpkg", Path: "testpkg"},
			},
		},
	}

	// 2. Filesystem setup
	// dotfiles/testpkg/config/settings.txt
	srcDir := filepath.Join(dotfilesDir, "testpkg", "config")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(srcDir, "settings.txt")
	if err := os.WriteFile(srcFile, []byte("important data"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. Create directory fold in home
	// ~/config -> dotfiles/testpkg/config
	homeConfigDir := filepath.Join(homeDir, "config")
	// Make sure parent exists
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create the symlink
	if err := os.Symlink(srcDir, homeConfigDir); err != nil {
		t.Fatal(err)
	}

	// Verify access via symlink
	targetFile := filepath.Join(homeConfigDir, "settings.txt")
	if _, err := os.Stat(targetFile); err != nil {
		t.Fatalf("Failed to access file via directory symlink: %v", err)
	}

	// 4. Run DetectConflicts
	conflicts, err := DetectConflicts(cfg, dotfilesDir)
	if err != nil {
		t.Fatalf("DetectConflicts failed: %v", err)
	}

	// 5. Assertions

	// Current BUG: It detects a conflict
	if len(conflicts) > 0 {
		t.Errorf("BUG REPRODUCED: Found %d conflicts for correctly folded directory", len(conflicts))

		// Simulate "Delete" action on the conflict
		// THIS IS THE DANGEROUS PART
		conflict := conflicts[0]

		// Verify target is what we think it is
		if conflict.TargetPath != targetFile {
			t.Errorf("Unexpected conflict path: %s", conflict.TargetPath)
		}

		// Try to remove it (simulate user choosing 'delete')
		if err := RemoveConflict(conflict); err != nil {
			t.Fatalf("RemoveConflict failed: %v", err)
		}

		// Check if SOURCE file is gone
		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			t.Errorf("CRITICAL: Source file was deleted! Data loss occurred.")
		} else {
			t.Log("Source file survived (unexpected given the hypothesis, or lucky OS behavior)")
		}
	} else {
		t.Log("No conflicts found (Behavior is correct)")
	}
}
