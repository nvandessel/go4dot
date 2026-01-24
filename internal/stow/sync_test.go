package stow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

func setupSyncTestEnv(t *testing.T) (dotfilesPath, homeDir string, cleanup func()) {
	t.Helper()

	// Use mock commander for all stow operations during tests
	origCommander := CurrentCommander
	CurrentCommander = &MockCommander{}

	tmpDir := t.TempDir()
	dotfilesPath = filepath.Join(tmpDir, "dotfiles")
	homeDir = filepath.Join(tmpDir, "home")

	origHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", homeDir); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(homeDir, 0755); err != nil {
		t.Fatal(err)
	}

	cleanup = func() {
		CurrentCommander = origCommander
		if err := os.Setenv("HOME", origHome); err != nil {
			t.Errorf("failed to restore HOME: %v", err)
		}
	}

	return dotfilesPath, homeDir, cleanup
}

func TestSyncAll_OrphanedCleanup(t *testing.T) {
	dotfilesPath, homeDir, cleanup := setupSyncTestEnv(t)
	defer cleanup()

	pkg1Path := filepath.Join(dotfilesPath, "pkg1")
	if err := os.MkdirAll(pkg1Path, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in dotfiles
	testFile := filepath.Join(pkg1Path, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
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

	// 1. Sync for the first time
	_, err := SyncAll(dotfilesPath, cfg, st, false, StowOptions{})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// Verify symlink exists
	targetFile := filepath.Join(homeDir, "test.txt")
	if _, err := os.Lstat(targetFile); err != nil {
		t.Errorf("Expected symlink at %s, got error: %v", targetFile, err)
	}

	// 2. Create an orphaned symlink
	orphanTarget := filepath.Join(homeDir, "orphan.txt")
	if err := os.Symlink(filepath.Join(pkg1Path, "nonexistent.txt"), orphanTarget); err != nil {
		t.Fatal(err)
	}

	// 3. Sync again and verify orphan is removed
	_, err = SyncAll(dotfilesPath, cfg, st, false, StowOptions{})
	if err != nil {
		t.Fatalf("SyncAll second time failed: %v", err)
	}

	if _, err := os.Lstat(orphanTarget); !os.IsNotExist(err) {
		t.Errorf("Expected orphaned symlink %s to be removed, but it still exists", orphanTarget)
	}
}

func TestSyncAll_RemovedConfig(t *testing.T) {
	dotfilesPath, homeDir, cleanup := setupSyncTestEnv(t)
	defer cleanup()

	// Setup pkg1 and pkg2
	for _, pkg := range []string{"pkg1", "pkg2"} {
		pkgPath := filepath.Join(dotfilesPath, pkg)
		if err := os.MkdirAll(pkgPath, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pkgPath, pkg+".txt"), []byte(pkg), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "pkg1", Path: "pkg1"},
				{Name: "pkg2", Path: "pkg2"},
			},
		},
	}

	st := state.New()

	// 1. Initial sync
	_, err := SyncAll(dotfilesPath, cfg, st, false, StowOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Verify both exist
	if _, err := os.Lstat(filepath.Join(homeDir, "pkg1.txt")); err != nil {
		t.Error("pkg1.txt missing")
	}
	if _, err := os.Lstat(filepath.Join(homeDir, "pkg2.txt")); err != nil {
		t.Error("pkg2.txt missing")
	}

	// 2. Remove pkg2 from config but it's still in state
	cfg.Configs.Core = []config.ConfigItem{
		{Name: "pkg1", Path: "pkg1"},
	}

	// 3. Sync and verify pkg2 is unstowed and removed from state
	_, err = SyncAll(dotfilesPath, cfg, st, false, StowOptions{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Lstat(filepath.Join(homeDir, "pkg2.txt")); !os.IsNotExist(err) {
		t.Error("pkg2.txt should have been unstowed")
	}

	found := false
	for _, sc := range st.Configs {
		if sc.Name == "pkg2" {
			found = true
			break
		}
	}
	if found {
		t.Error("pkg2 should have been removed from state")
	}
}

func TestSyncSingle(t *testing.T) {
	dotfilesPath, homeDir, cleanup := setupSyncTestEnv(t)
	defer cleanup()

	pkg1Path := filepath.Join(dotfilesPath, "pkg1")
	if err := os.MkdirAll(pkg1Path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg1Path, "f1.txt"), []byte("f1"), 0644); err != nil {
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

	// Create orphan for pkg1
	orphanTarget := filepath.Join(homeDir, "orphan.txt")
	if err := os.Symlink(filepath.Join(pkg1Path, "gone.txt"), orphanTarget); err != nil {
		t.Fatal(err)
	}

	err := SyncSingle(dotfilesPath, "pkg1", cfg, st, StowOptions{})
	if err != nil {
		t.Fatal(err)
	}

	// Verify f1.txt is symlinked
	if _, err := os.Lstat(filepath.Join(homeDir, "f1.txt")); err != nil {
		t.Error("f1.txt missing")
	}

	// Verify orphan is removed
	if _, err := os.Lstat(orphanTarget); !os.IsNotExist(err) {
		t.Error("orphan should have been removed")
	}
}
