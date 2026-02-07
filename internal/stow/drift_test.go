package stow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
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

	summary, err := FullDriftCheck(cfg, tmpDir)
	if err != nil {
		t.Fatalf("FullDriftCheck failed: %v", err)
	}

	if summary.TotalConfigs != 1 {
		t.Errorf("Expected 1 total config, got %d", summary.TotalConfigs)
	}

	if len(summary.Results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(summary.Results))
	}

	result := summary.Results[0]
	if result.CurrentCount != 1 {
		t.Errorf("Expected 1 file, got %d", result.CurrentCount)
	}

	// Should detect as new file since no symlink exists in home
	if len(result.NewFiles) != 1 {
		t.Errorf("Expected 1 new file, got %d", len(result.NewFiles))
	}

	if summary.DriftedConfigs != 1 {
		t.Errorf("Expected 1 drifted config, got %d", summary.DriftedConfigs)
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

func TestDriftSummaryResultsMap(t *testing.T) {
	tests := []struct {
		name    string
		summary *DriftSummary
	}{
		{
			name:    "nil summary returns nil",
			summary: nil,
		},
		{
			name: "empty results returns empty map",
			summary: &DriftSummary{
				Results: []DriftResult{},
			},
		},
		{
			name: "single result",
			summary: &DriftSummary{
				Results: []DriftResult{
					{ConfigName: "vim", HasDrift: true},
				},
			},
		},
		{
			name: "multiple results",
			summary: &DriftSummary{
				Results: []DriftResult{
					{ConfigName: "vim", HasDrift: true},
					{ConfigName: "zsh", HasDrift: false},
					{ConfigName: "nvim", HasDrift: true},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.ResultsMap()

			if tt.summary == nil {
				if got != nil {
					t.Errorf("ResultsMap() = %v, want nil", got)
				}
				return
			}

			if len(got) != len(tt.summary.Results) {
				t.Errorf("ResultsMap() returned %d entries, want %d", len(got), len(tt.summary.Results))
			}

			// Verify each result is accessible by ConfigName and points to the original slice element
			for i := range tt.summary.Results {
				r := &tt.summary.Results[i]
				if mapResult, ok := got[r.ConfigName]; !ok {
					t.Errorf("ResultsMap() missing key %q", r.ConfigName)
				} else if mapResult != r {
					t.Errorf("ResultsMap()[%q] points to different memory address", r.ConfigName)
				}
			}
		})
	}
}

func TestDriftSummaryResultByName(t *testing.T) {
	tests := []struct {
		name       string
		summary    *DriftSummary
		configName string
		wantNil    bool
		wantName   string
	}{
		{
			name:       "nil summary returns nil",
			summary:    nil,
			configName: "vim",
			wantNil:    true,
		},
		{
			name: "empty results returns nil",
			summary: &DriftSummary{
				Results: []DriftResult{},
			},
			configName: "vim",
			wantNil:    true,
		},
		{
			name: "config not found returns nil",
			summary: &DriftSummary{
				Results: []DriftResult{
					{ConfigName: "zsh", HasDrift: false},
				},
			},
			configName: "vim",
			wantNil:    true,
		},
		{
			name: "finds existing config",
			summary: &DriftSummary{
				Results: []DriftResult{
					{ConfigName: "vim", HasDrift: true},
					{ConfigName: "zsh", HasDrift: false},
				},
			},
			configName: "vim",
			wantNil:    false,
			wantName:   "vim",
		},
		{
			name: "finds last config",
			summary: &DriftSummary{
				Results: []DriftResult{
					{ConfigName: "vim", HasDrift: true},
					{ConfigName: "zsh", HasDrift: false},
					{ConfigName: "nvim", HasDrift: true},
				},
			},
			configName: "nvim",
			wantNil:    false,
			wantName:   "nvim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.ResultByName(tt.configName)

			if tt.wantNil {
				if got != nil {
					t.Errorf("ResultByName(%q) = %v, want nil", tt.configName, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("ResultByName(%q) = nil, want non-nil", tt.configName)
			}

			if got.ConfigName != tt.wantName {
				t.Errorf("ResultByName(%q).ConfigName = %q, want %q", tt.configName, got.ConfigName, tt.wantName)
			}

			// Verify it points to the original slice element, not a copy
			for i := range tt.summary.Results {
				if tt.summary.Results[i].ConfigName == tt.configName {
					if got != &tt.summary.Results[i] {
						t.Errorf("ResultByName(%q) returns a copy, not a pointer to the original", tt.configName)
					}
					break
				}
			}
		})
	}
}
