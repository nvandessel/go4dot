package stow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

func TestCountFiles(t *testing.T) {
	// Create temp directory with some files
	tmpDir := t.TempDir()

	// Create some files
	testFiles := []string{
		"file1.txt",
		"subdir/file2.txt",
		"subdir/nested/file3.txt",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	count, err := countFiles(tmpDir)
	if err != nil {
		t.Fatalf("countFiles failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 files, got %d", count)
	}
}

func TestCountFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	count, err := countFiles(tmpDir)
	if err != nil {
		t.Fatalf("countFiles failed: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 files, got %d", count)
	}
}

func TestCountFilesNotExist(t *testing.T) {
	_, err := countFiles("/nonexistent/path")
	if err == nil {
		t.Error("Expected error for non-existent path")
	}
}

func TestQuickDriftCheck(t *testing.T) {
	// Create temp dotfiles directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "nvim")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create some files
	for i := 0; i < 3; i++ {
		if err := os.WriteFile(filepath.Join(configDir, "file"+string(rune('0'+i))+".lua"), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "nvim", Path: "nvim"},
			},
		},
	}

	// Create state with different count (simulate drift)
	st := state.New()
	st.SetSymlinkCount("nvim", 2) // Was 2, now 3

	summary, err := QuickDriftCheck(cfg, tmpDir, st)
	if err != nil {
		t.Fatalf("QuickDriftCheck failed: %v", err)
	}

	if !summary.HasDrift() {
		t.Error("Expected drift to be detected")
	}

	if summary.DriftedConfigs != 1 {
		t.Errorf("Expected 1 drifted config, got %d", summary.DriftedConfigs)
	}

	if len(summary.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(summary.Results))
	}

	result := summary.Results[0]
	if result.CurrentCount != 3 {
		t.Errorf("Expected current count 3, got %d", result.CurrentCount)
	}
	if result.StoredCount != 2 {
		t.Errorf("Expected stored count 2, got %d", result.StoredCount)
	}
}

func TestQuickDriftCheckNoDrift(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "zsh")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create 2 files
	for i := 0; i < 2; i++ {
		if err := os.WriteFile(filepath.Join(configDir, "file"+string(rune('0'+i))), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "zsh", Path: "zsh"},
			},
		},
	}

	// State matches
	st := state.New()
	st.SetSymlinkCount("zsh", 2)

	summary, err := QuickDriftCheck(cfg, tmpDir, st)
	if err != nil {
		t.Fatalf("QuickDriftCheck failed: %v", err)
	}

	if summary.HasDrift() {
		t.Error("Expected no drift")
	}
}

func TestFullDriftCheck(t *testing.T) {
	// Create temp dotfiles directory
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "test")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file in dotfiles
	if err := os.WriteFile(filepath.Join(configDir, ".testrc"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "test", Path: "test"},
			},
		},
	}

	results, err := FullDriftCheck(cfg, tmpDir)
	if err != nil {
		t.Fatalf("FullDriftCheck failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	result := results[0]
	if result.CurrentCount != 1 {
		t.Errorf("Expected 1 file, got %d", result.CurrentCount)
	}

	// Should detect as new file since no symlink exists in home
	if len(result.NewFiles) != 1 {
		t.Errorf("Expected 1 new file, got %d", len(result.NewFiles))
	}
}

func TestGetDriftedConfigs(t *testing.T) {
	results := []DriftResult{
		{ConfigName: "a", HasDrift: true},
		{ConfigName: "b", HasDrift: false},
		{ConfigName: "c", HasDrift: true},
	}

	drifted := GetDriftedConfigs(results)

	if len(drifted) != 2 {
		t.Errorf("Expected 2 drifted configs, got %d", len(drifted))
	}
}

func TestDriftSummaryHasDrift(t *testing.T) {
	tests := []struct {
		name     string
		summary  DriftSummary
		expected bool
	}{
		{
			name:     "No drifted configs",
			summary:  DriftSummary{DriftedConfigs: 0},
			expected: false,
		},
		{
			name:     "Has drifted configs",
			summary:  DriftSummary{DriftedConfigs: 2},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.summary.HasDrift(); got != tt.expected {
				t.Errorf("HasDrift() = %v, want %v", got, tt.expected)
			}
		})
	}
}
