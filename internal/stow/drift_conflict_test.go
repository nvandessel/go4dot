package stow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

func TestDetectConflicts_DirectoryFolding_DataLoss(t *testing.T) {
	type testCase struct {
		name              string
		setupFunc         func(t *testing.T, dotfilesDir, homeDir string)
		expectedConflicts int
		description       string
	}

	tests := []testCase{
		{
			name: "Directory Folding (Correctly Linked)",
			setupFunc: func(t *testing.T, dotfilesDir, homeDir string) {
				// dotfiles/testpkg/config/settings.txt
				srcDir := filepath.Join(dotfilesDir, "testpkg", "config")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				srcFile := filepath.Join(srcDir, "settings.txt")
				if err := os.WriteFile(srcFile, []byte("important data"), 0644); err != nil {
					t.Fatal(err)
				}

				// ~/config -> dotfiles/testpkg/config
				homeConfigDir := filepath.Join(homeDir, "config")
				if err := os.MkdirAll(homeDir, 0755); err != nil {
					t.Fatal(err)
				}

				// Create the directory symlink
				if err := os.Symlink(srcDir, homeConfigDir); err != nil {
					t.Fatal(err)
				}
			},
			expectedConflicts: 0,
			description:       "Symlinked directory (fold) should not be reported as conflict",
		},
		{
			name: "Real Conflict (Regular File)",
			setupFunc: func(t *testing.T, dotfilesDir, homeDir string) {
				// dotfiles/testpkg/file.txt
				srcDir := filepath.Join(dotfilesDir, "testpkg")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(srcDir, "file.txt"), []byte("source"), 0644); err != nil {
					t.Fatal(err)
				}

				// ~/file.txt (regular file)
				if err := os.MkdirAll(homeDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(homeDir, "file.txt"), []byte("conflict"), 0644); err != nil {
					t.Fatal(err)
				}
			},
			expectedConflicts: 1,
			description:       "Regular file in place of symlink should be a conflict",
		},
		{
			name: "Correct Direct Symlink",
			setupFunc: func(t *testing.T, dotfilesDir, homeDir string) {
				// dotfiles/testpkg/link.txt
				srcDir := filepath.Join(dotfilesDir, "testpkg")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					t.Fatal(err)
				}
				srcFile := filepath.Join(srcDir, "link.txt")
				if err := os.WriteFile(srcFile, []byte("source"), 0644); err != nil {
					t.Fatal(err)
				}

				// ~/link.txt -> dotfiles/testpkg/link.txt
				if err := os.MkdirAll(homeDir, 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(srcFile, filepath.Join(homeDir, "link.txt")); err != nil {
					t.Fatal(err)
				}
			},
			expectedConflicts: 0,
			description:       "Correct direct symlink should not be a conflict",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup isolated directories for each test case
			tmpDir := t.TempDir()
			dotfilesDir := filepath.Join(tmpDir, "dotfiles")
			homeDir := filepath.Join(tmpDir, "home")

			// Very important: Set HOME to temp dir for isolation
			t.Setenv("HOME", homeDir)

			// Common setup
			if err := os.MkdirAll(dotfilesDir, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(homeDir, 0755); err != nil {
				t.Fatal(err)
			}

			// Run scenario-specific setup
			tc.setupFunc(t, dotfilesDir, homeDir)

			// Config setup (matches the paths used in setupFuncs)
			// Note: "testpkg" is hardcoded in the setupFuncs as the package name
			cfg := &config.Config{
				Configs: config.ConfigGroups{
					Core: []config.ConfigItem{
						{Name: "testpkg", Path: "testpkg"},
					},
				},
			}

			// Run detection
			conflicts, err := DetectConflicts(cfg, dotfilesDir)
			if err != nil {
				t.Fatalf("DetectConflicts failed: %v", err)
			}

			// Verify count
			if len(conflicts) != tc.expectedConflicts {
				t.Errorf("Expected %d conflicts, got %d. (%s)", tc.expectedConflicts, len(conflicts), tc.description)
			}

			// For the "Directory Folding" case specifically, we want to ensure data safety
			// If it HAD failed (returned conflicts), we verify that deleting them works (for the conflict case)
			// or warns us (for the bug case).
			if tc.name == "Directory Folding (Correctly Linked)" && len(conflicts) > 0 {
				// This block only runs if the bug reappears
				t.Errorf("BUG REPRODUCED: Found conflicts for directory fold")

				// Prove data loss would happen
				conflict := conflicts[0]
				if err := RemoveConflict(conflict); err != nil {
					t.Fatalf("Failed to remove conflict: %v", err)
				}

				// Check source file
				srcFile := filepath.Join(dotfilesDir, "testpkg", "config", "settings.txt")
				if _, err := os.Stat(srcFile); os.IsNotExist(err) {
					t.Errorf("CRITICAL: Source file was deleted! Data loss occurred.")
				}
			}
		})
	}
}
