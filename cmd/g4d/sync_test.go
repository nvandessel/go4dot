package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/ui"
)

func TestSyncCommands(t *testing.T) {
	tmpDir := t.TempDir()
	dotfilesPath := filepath.Join(tmpDir, "dotfiles")
	homeDir := filepath.Join(tmpDir, "home")

	origHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Setenv("HOME", origHome); err != nil {
			t.Errorf("failed to restore HOME: %v", err)
		}
	}()

	// Ensure non-interactive for tests
	ui.SetNonInteractive(true)

	// Setup pkg1
	pkg1Path := filepath.Join(dotfilesPath, "pkg1")
	if err := os.MkdirAll(pkg1Path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg1Path, "test.txt"), []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "pkg1", Path: "pkg1"},
			},
		},
	}
	st := state.New()

	t.Run("syncAllConfigs", func(t *testing.T) {
		err := syncAllConfigs(cfg, dotfilesPath, st)
		if err != nil {
			t.Fatalf("syncAllConfigs failed: %v", err)
		}

		// Verify symlink
		if _, err := os.Lstat(filepath.Join(homeDir, "test.txt")); err != nil {
			t.Error("test.txt not symlinked")
		}
	})

	t.Run("syncSingleConfig", func(t *testing.T) {
		// Add another file
		if err := os.WriteFile(filepath.Join(pkg1Path, "test2.txt"), []byte("content2"), 0644); err != nil {
			t.Fatal(err)
		}

		err := syncSingleConfig("pkg1", cfg, dotfilesPath, st)
		if err != nil {
			t.Fatalf("syncSingleConfig failed: %v", err)
		}

		// Verify symlink
		if _, err := os.Lstat(filepath.Join(homeDir, "test2.txt")); err != nil {
			t.Error("test2.txt not symlinked")
		}
	})

	t.Run("syncSingleConfig NotFound", func(t *testing.T) {
		err := syncSingleConfig("nonexistent", cfg, dotfilesPath, st)
		if err == nil {
			t.Error("expected error for nonexistent config, got nil")
		}
	})
}
