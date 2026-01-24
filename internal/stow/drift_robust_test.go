package stow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

func TestFullDriftCheck_Robust(t *testing.T) {
	// Setup:
	// tmp/dotfiles/pkg1/.config/test.conf
	// tmp/dotfiles/pkg1/.bashrc
	// tmp/home/ (target)

	tmpDir := t.TempDir()
	dotfilesPath := filepath.Join(tmpDir, "dotfiles")
	homeDir := filepath.Join(tmpDir, "home")

	pkg1Path := filepath.Join(dotfilesPath, "pkg1")
	err := os.MkdirAll(filepath.Join(pkg1Path, ".config"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(homeDir, 0755)
	if err != nil {
		t.Fatal(err)
	}

	testConfSource := filepath.Join(pkg1Path, ".config/test.conf")
	if err := os.WriteFile(testConfSource, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}
	bashrcSource := filepath.Join(pkg1Path, ".bashrc")
	if err := os.WriteFile(bashrcSource, []byte("bashrc"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "pkg1", Path: "pkg1"},
			},
		},
	}

	t.Run("Initially everything is new", func(t *testing.T) {
		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, homeDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if summary.DriftedConfigs != 1 {
			t.Errorf("expected 1 drifted config, got %d", summary.DriftedConfigs)
		}
		res := summary.Results[0]
		if len(res.NewFiles) != 2 {
			t.Errorf("expected 2 new files, got %d", len(res.NewFiles))
		}
	})

	t.Run("With correct symlinks", func(t *testing.T) {
		// Create correct symlinks
		err = os.MkdirAll(filepath.Join(homeDir, ".config"), 0755)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(testConfSource, filepath.Join(homeDir, ".config/test.conf")); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(bashrcSource, filepath.Join(homeDir, ".bashrc")); err != nil {
			t.Fatal(err)
		}

		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, homeDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if summary.DriftedConfigs != 0 {
			t.Errorf("expected 0 drifted configs, got %d", summary.DriftedConfigs)
		}
	})

	t.Run("With conflict (regular file)", func(t *testing.T) {
		// Replace one symlink with a regular file
		targetBashrc := filepath.Join(homeDir, ".bashrc")
		if err := os.Remove(targetBashrc); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(targetBashrc, []byte("conflict"), 0644); err != nil {
			t.Fatal(err)
		}

		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, homeDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		res := summary.Results[0]
		if len(res.ConflictFiles) != 1 || res.ConflictFiles[0] != ".bashrc" {
			t.Errorf("expected .bashrc conflict, got %v", res.ConflictFiles)
		}
	})

	t.Run("With missing file (orphaned symlink)", func(t *testing.T) {
		// Restore bashrc symlink
		targetBashrc := filepath.Join(homeDir, ".bashrc")
		if err := os.Remove(targetBashrc); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(bashrcSource, targetBashrc); err != nil {
			t.Fatal(err)
		}

		// Create an orphaned symlink: symlink exists in home but source is gone from dotfiles
		orphanSource := filepath.Join(pkg1Path, "deleted.txt")
		// (We don't need the source to exist to create a symlink)
		targetOrphan := filepath.Join(homeDir, "deleted.txt")
		if err := os.Symlink(orphanSource, targetOrphan); err != nil {
			t.Fatal(err)
		}

		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, homeDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		res := summary.Results[0]
		if len(res.MissingFiles) != 1 {
			t.Errorf("expected 1 missing file, got %d: %v", len(res.MissingFiles), res.MissingFiles)
		} else if res.MissingFiles[0] != "deleted.txt" {
			t.Errorf("expected deleted.txt as missing, got %s", res.MissingFiles[0])
		}
	})

	t.Run("Directory folding detection", func(t *testing.T) {
		// Clear home and do directory folding
		if err := os.RemoveAll(homeDir); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(homeDir, 0755); err != nil {
			t.Fatal(err)
		}

		// Symlink the whole .config directory
		if err := os.Symlink(filepath.Join(pkg1Path, ".config"), filepath.Join(homeDir, ".config")); err != nil {
			t.Fatal(err)
		}
		// .bashrc still separate symlink
		if err := os.Symlink(bashrcSource, filepath.Join(homeDir, ".bashrc")); err != nil {
			t.Fatal(err)
		}

		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, homeDir, nil)
		if err != nil {
			t.Fatal(err)
		}
		if summary.DriftedConfigs != 0 {
			t.Errorf("expected 0 drift with directory folding, got %d", summary.DriftedConfigs)
			for _, r := range summary.Results {
				t.Logf("Config %s: New=%v, Conflict=%v, Missing=%v", r.ConfigName, r.NewFiles, r.ConflictFiles, r.MissingFiles)
			}
		}
	})

	t.Run("Removed configs (in state but not in config)", func(t *testing.T) {
		st := state.New()
		st.AddConfig("oldpkg", "oldpkg", false)

		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, homeDir, st)
		if err != nil {
			t.Fatal(err)
		}

		if len(summary.RemovedConfigs) != 1 || summary.RemovedConfigs[0] != "oldpkg" {
			t.Errorf("expected oldpkg in removed configs, got %v", summary.RemovedConfigs)
		}
		if summary.HasDrift() == false {
			t.Error("expected HasDrift() to be true")
		}
	})
}
