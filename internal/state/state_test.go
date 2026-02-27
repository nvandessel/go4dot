package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew(t *testing.T) {
	s := New()

	if s.Version != StateVersion {
		t.Errorf("Version = %s, want %s", s.Version, StateVersion)
	}

	if s.InstalledAt.IsZero() {
		t.Error("InstalledAt should not be zero")
	}

	if s.MachineConfig == nil {
		t.Error("MachineConfig should be initialized")
	}

	if s.ExternalDeps == nil {
		t.Error("ExternalDeps should be initialized")
	}
}

func TestStateAddRemoveConfig(t *testing.T) {
	s := New()

	// Add config
	s.AddConfig("git", "git", true)

	if !s.HasConfig("git") {
		t.Error("HasConfig('git') should be true")
	}

	if len(s.Configs) != 1 {
		t.Errorf("len(Configs) = %d, want 1", len(s.Configs))
	}

	// Add same config again (should update, not duplicate)
	s.AddConfig("git", "git-updated", false)

	if len(s.Configs) != 1 {
		t.Errorf("len(Configs) = %d, want 1 (no duplicates)", len(s.Configs))
	}

	if s.Configs[0].Path != "git-updated" {
		t.Errorf("Path = %s, want 'git-updated'", s.Configs[0].Path)
	}

	// Add another config
	s.AddConfig("nvim", "nvim", true)

	if len(s.Configs) != 2 {
		t.Errorf("len(Configs) = %d, want 2", len(s.Configs))
	}

	// Remove config
	s.RemoveConfig("git")

	if s.HasConfig("git") {
		t.Error("HasConfig('git') should be false after removal")
	}

	if len(s.Configs) != 1 {
		t.Errorf("len(Configs) = %d, want 1", len(s.Configs))
	}
}

func TestStateGetConfigNames(t *testing.T) {
	s := New()

	s.AddConfig("git", "git", true)
	s.AddConfig("nvim", "nvim", true)
	s.AddConfig("tmux", "tmux", false)

	names := s.GetConfigNames()

	if len(names) != 3 {
		t.Errorf("len(names) = %d, want 3", len(names))
	}

	// Check all names are present
	expected := map[string]bool{"git": false, "nvim": false, "tmux": false}
	for _, name := range names {
		if _, ok := expected[name]; !ok {
			t.Errorf("Unexpected name: %s", name)
		}
		expected[name] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("Missing name: %s", name)
		}
	}
}

func TestStateExternalDeps(t *testing.T) {
	s := New()

	// Set external dep
	s.SetExternalDep("pure", "~/.zsh/pure", true)

	if dep, ok := s.ExternalDeps["pure"]; !ok {
		t.Error("ExternalDeps['pure'] should exist")
	} else {
		if !dep.Installed {
			t.Error("Installed should be true")
		}
		if dep.Path != "~/.zsh/pure" {
			t.Errorf("Path = %s, want '~/.zsh/pure'", dep.Path)
		}
	}

	// Remove
	s.RemoveExternalDep("pure")

	if _, ok := s.ExternalDeps["pure"]; ok {
		t.Error("ExternalDeps['pure'] should not exist after removal")
	}
}

func TestStateMachineConfig(t *testing.T) {
	s := New()

	// Set machine config
	s.SetMachineConfig("git", "~/.gitconfig.local", true, false)

	if mc, ok := s.MachineConfig["git"]; !ok {
		t.Error("MachineConfig['git'] should exist")
	} else {
		if mc.ConfigPath != "~/.gitconfig.local" {
			t.Errorf("ConfigPath = %s, want '~/.gitconfig.local'", mc.ConfigPath)
		}
		if !mc.HasGPG {
			t.Error("HasGPG should be true")
		}
		if mc.HasSSH {
			t.Error("HasSSH should be false")
		}
	}

	// Remove
	s.RemoveMachineConfig("git")

	if _, ok := s.MachineConfig["git"]; ok {
		t.Error("MachineConfig['git'] should not exist after removal")
	}
}

func TestStateSaveLoad(t *testing.T) {
	// Create temp directory for state
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".config", "go4dot")
	statePath := filepath.Join(stateDir, "state.json")

	// Override home for testing
	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	// Create and save state
	s := New()
	s.DotfilesPath = "/home/user/dotfiles"
	s.Platform = PlatformState{
		OS:             "linux",
		Distro:         "fedora",
		DistroVersion:  "43",
		PackageManager: "dnf",
	}
	s.AddConfig("git", "git", true)
	s.SetExternalDep("pure", "~/.zsh/pure", true)
	s.SetMachineConfig("git", "~/.gitconfig.local", true, false)

	err := s.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("State file should exist: %v", err)
	}

	// Load state
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loaded == nil {
		t.Fatal("Load() returned nil")
	}

	// Verify loaded state
	if loaded.DotfilesPath != s.DotfilesPath {
		t.Errorf("DotfilesPath = %s, want %s", loaded.DotfilesPath, s.DotfilesPath)
	}

	if loaded.Platform.OS != "linux" {
		t.Errorf("Platform.OS = %s, want 'linux'", loaded.Platform.OS)
	}

	if !loaded.HasConfig("git") {
		t.Error("Should have 'git' config")
	}

	if _, ok := loaded.ExternalDeps["pure"]; !ok {
		t.Error("Should have 'pure' external dep")
	}

	if _, ok := loaded.MachineConfig["git"]; !ok {
		t.Error("Should have 'git' machine config")
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Create temp directory with no state
	tmpDir := t.TempDir()

	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	// Load should return nil, nil for non-existent
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() should not error for non-existent: %v", err)
	}

	if loaded != nil {
		t.Error("Load() should return nil for non-existent state")
	}
}

func TestStateSavePermissions(t *testing.T) {
	tmpDir := t.TempDir()

	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	s := New()
	s.DotfilesPath = "/home/user/dotfiles"

	err := s.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Check state directory permissions (should be 0700)
	stateDir := filepath.Join(tmpDir, ".config", "go4dot")
	dirInfo, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("Failed to stat state directory: %v", err)
	}
	dirPerm := dirInfo.Mode().Perm()
	if dirPerm != 0700 {
		t.Errorf("State directory permissions = %04o, want 0700", dirPerm)
	}

	// Check state file permissions (should be 0600)
	statePath := filepath.Join(stateDir, "state.json")
	fileInfo, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Failed to stat state file: %v", err)
	}
	filePerm := fileInfo.Mode().Perm()
	if filePerm != 0600 {
		t.Errorf("State file permissions = %04o, want 0600", filePerm)
	}
}

func TestExists(t *testing.T) {
	tmpDir := t.TempDir()

	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpDir)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	// Initially should not exist
	if Exists() {
		t.Error("Exists() should be false initially")
	}

	// Create and save state
	s := New()
	if err := s.Save(); err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Now should exist
	if !Exists() {
		t.Error("Exists() should be true after Save()")
	}

	// Delete
	if err := Delete(); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	// Should not exist again
	if Exists() {
		t.Error("Exists() should be false after Delete()")
	}
}
