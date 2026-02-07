package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/machine"
)

func TestCheckStatusIsError(t *testing.T) {
	tests := []struct {
		status   CheckStatus
		expected bool
	}{
		{StatusOK, false},
		{StatusWarning, false},
		{StatusError, true},
		{StatusSkipped, false},
	}

	for _, tt := range tests {
		if got := tt.status.isError(); got != tt.expected {
			t.Errorf("CheckStatus(%v).isError() = %v, want %v", tt.status, got, tt.expected)
		}
	}
}

func TestCheckResultIsHealthy(t *testing.T) {
	tests := []struct {
		name    string
		checks  []Check
		healthy bool
	}{
		{
			name: "All OK",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusOK},
			},
			healthy: true,
		},
		{
			name: "With warnings",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusWarning},
			},
			healthy: true, // Warnings don't make it unhealthy
		},
		{
			name: "With error",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusError},
			},
			healthy: false,
		},
		{
			name: "With skipped",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusSkipped},
			},
			healthy: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CheckResult{Checks: tt.checks}
			if got := result.IsHealthy(); got != tt.healthy {
				t.Errorf("IsHealthy() = %v, want %v", got, tt.healthy)
			}
		})
	}
}

func TestCheckResultHasWarnings(t *testing.T) {
	tests := []struct {
		name        string
		checks      []Check
		hasWarnings bool
	}{
		{
			name: "No warnings",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusError},
			},
			hasWarnings: false,
		},
		{
			name: "Has warnings",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusWarning},
			},
			hasWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CheckResult{Checks: tt.checks}
			if got := result.HasWarnings(); got != tt.hasWarnings {
				t.Errorf("HasWarnings() = %v, want %v", got, tt.hasWarnings)
			}
		})
	}
}

func TestCheckResultCountByStatus(t *testing.T) {
	result := &CheckResult{
		Checks: []Check{
			{Status: StatusOK},
			{Status: StatusOK},
			{Status: StatusWarning},
			{Status: StatusError},
			{Status: StatusSkipped},
		},
	}

	ok, warnings, errors, skipped := result.CountByStatus()

	if ok != 2 {
		t.Errorf("ok = %d, want 2", ok)
	}
	if warnings != 1 {
		t.Errorf("warnings = %d, want 1", warnings)
	}
	if errors != 1 {
		t.Errorf("errors = %d, want 1", errors)
	}
	if skipped != 1 {
		t.Errorf("skipped = %d, want 1", skipped)
	}
}

func TestCheckStow(t *testing.T) {
	check := checkStow()

	// The check should complete without error
	if check.Name != "GNU Stow" {
		t.Errorf("Name = %q, want 'GNU Stow'", check.Name)
	}

	// Log the result (may vary by system)
	t.Logf("Stow check: status=%v, message=%s", check.Status, check.Message)
}

func TestCheckGit(t *testing.T) {
	check := checkGit()

	if check.Name != "Git" {
		t.Errorf("Name = %q, want 'Git'", check.Name)
	}

	// Git should be installed on most development systems
	if check.Status == StatusOK {
		if check.Message == "" {
			t.Error("Expected message to show git path")
		}
	}

	t.Logf("Git check: status=%v, message=%s", check.Status, check.Message)
}

func TestSummarizeDepsCheck(t *testing.T) {
	tests := []struct {
		name           string
		result         *deps.CheckResult
		expectedStatus CheckStatus
	}{
		{
			name: "All installed",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "git"}, Status: deps.StatusInstalled},
				},
				Core: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "stow"}, Status: deps.StatusInstalled},
				},
			},
			expectedStatus: StatusOK,
		},
		{
			name: "Critical missing",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "git"}, Status: deps.StatusMissing},
				},
			},
			expectedStatus: StatusError,
		},
		{
			name: "Optional missing",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "git"}, Status: deps.StatusInstalled},
				},
				Optional: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "fzf"}, Status: deps.StatusMissing},
				},
			},
			expectedStatus: StatusWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := summarizeDepsCheck(tt.result)
			if check.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.expectedStatus)
			}
		})
	}
}

func TestSummarizeExternalCheck(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []deps.ExternalStatus
		expectedStatus CheckStatus
	}{
		{
			name: "All installed",
			statuses: []deps.ExternalStatus{
				{Status: "installed"},
				{Status: "installed"},
			},
			expectedStatus: StatusOK,
		},
		{
			name: "Some missing",
			statuses: []deps.ExternalStatus{
				{Status: "installed"},
				{Status: "missing"},
			},
			expectedStatus: StatusWarning,
		},
		{
			name: "All skipped",
			statuses: []deps.ExternalStatus{
				{Status: "skipped"},
			},
			expectedStatus: StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := summarizeExternalCheck(tt.statuses)
			if check.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.expectedStatus)
			}
		})
	}
}

func TestSummarizeMachineCheck(t *testing.T) {
	tests := []struct {
		name           string
		statuses       []machine.MachineConfigStatus
		expectedStatus CheckStatus
	}{
		{
			name: "All configured",
			statuses: []machine.MachineConfigStatus{
				{Status: "configured"},
				{Status: "configured"},
			},
			expectedStatus: StatusOK,
		},
		{
			name: "Some missing",
			statuses: []machine.MachineConfigStatus{
				{Status: "configured"},
				{Status: "missing"},
			},
			expectedStatus: StatusWarning,
		},
		{
			name: "Has error",
			statuses: []machine.MachineConfigStatus{
				{Status: "configured"},
				{Status: "error"},
			},
			expectedStatus: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := summarizeMachineCheck(tt.statuses)
			if check.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.expectedStatus)
			}
		})
	}
}

func TestSummarizeSymlinkCheck(t *testing.T) {
	tests := []struct {
		name           string
		checks         []SymlinkCheck
		expectedStatus CheckStatus
	}{
		{
			name: "All OK",
			checks: []SymlinkCheck{
				{Status: StatusOK},
				{Status: StatusOK},
			},
			expectedStatus: StatusOK,
		},
		{
			name: "Some warnings",
			checks: []SymlinkCheck{
				{Status: StatusOK},
				{Status: StatusWarning},
			},
			expectedStatus: StatusWarning,
		},
		{
			name: "Has errors",
			checks: []SymlinkCheck{
				{Status: StatusOK},
				{Status: StatusError},
			},
			expectedStatus: StatusError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := summarizeSymlinkCheck(tt.checks)
			if check.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.expectedStatus)
			}
		})
	}
}

func TestCheckSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	home := os.Getenv("HOME")

	// Create a fake dotfiles structure
	configDir := filepath.Join(tmpDir, "testconfig", ".config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Create a test file in the dotfiles
	testFile := filepath.Join(configDir, "test.conf")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		Configs: config.ConfigGroups{
			Core: []config.ConfigItem{
				{Name: "testconfig", Path: "testconfig"},
			},
		},
	}

	checks := checkSymlinks(cfg, tmpDir)

	// Should have at least one check
	if len(checks) == 0 {
		t.Error("Expected at least one symlink check")
	}

	// Log results
	for _, c := range checks {
		t.Logf("Symlink check: config=%s, target=%s, status=%v, msg=%s",
			c.Config, c.TargetPath, c.Status, c.Message)
	}

	// The symlink should be "missing" since we haven't created it
	found := false
	for _, c := range checks {
		if c.Config == "testconfig" {
			found = true
			// It should be warning (missing) since the symlink doesn't exist
			if c.Status != StatusWarning && c.Status != StatusOK {
				// It might be OK if there's already a symlink in the home directory
				t.Logf("Expected warning or ok status for missing symlink, got %v", c.Status)
			}
		}
	}

	if !found {
		t.Error("Expected to find check for 'testconfig'")
	}

	_ = home // Used implicitly by checkSymlinks via $HOME env var
}

func TestRunChecks(t *testing.T) {
	cfg := &config.Config{
		SchemaVersion: "1.0",
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"}, // Should exist
			},
		},
	}

	var progressMessages []string
	opts := CheckOptions{
		ProgressFunc: func(current, total int, msg string) {
			progressMessages = append(progressMessages, msg)
		},
	}

	result, err := RunChecks(cfg, opts)
	if err != nil {
		t.Fatalf("RunChecks failed: %v", err)
	}

	if result.Platform == nil {
		t.Error("Platform should be detected")
	}

	if len(result.Checks) == 0 {
		t.Error("Expected at least one check")
	}

	if len(progressMessages) == 0 {
		t.Error("Expected progress messages")
	}

	// Log the checks
	for _, check := range result.Checks {
		t.Logf("Check: %s - %v - %s", check.Name, check.Status, check.Message)
	}
}

func TestProgress(t *testing.T) {
	var received string
	opts := CheckOptions{
		ProgressFunc: func(current, total int, msg string) {
			received = msg
		},
	}

	progress(opts, "test message")

	if received != "test message" {
		t.Errorf("Expected 'test message', got %q", received)
	}
}

func TestProgressNoCallback(t *testing.T) {
	opts := CheckOptions{}

	// Should not panic with nil callback
	progress(opts, "test message")
}

func TestSummarizeDepsCheckWithManual(t *testing.T) {
	tests := []struct {
		name           string
		result         *deps.CheckResult
		expectedStatus CheckStatus
		msgContains    string
	}{
		{
			name: "Only manual missing",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "git"}, Status: deps.StatusInstalled},
				},
				Core: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "op", Manual: true}, Status: deps.StatusManualMissing},
				},
			},
			expectedStatus: StatusWarning,
			msgContains:    "manual",
		},
		{
			name: "Manual and optional missing",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "git"}, Status: deps.StatusInstalled},
				},
				Core: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "op", Manual: true}, Status: deps.StatusManualMissing},
				},
				Optional: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "fzf"}, Status: deps.StatusMissing},
				},
			},
			expectedStatus: StatusWarning,
			msgContains:    "manual",
		},
		{
			name: "All installed including manual",
			result: &deps.CheckResult{
				Critical: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "git"}, Status: deps.StatusInstalled},
				},
				Core: []deps.DependencyCheck{
					{Item: config.DependencyItem{Name: "op", Manual: true}, Status: deps.StatusInstalled},
				},
			},
			expectedStatus: StatusOK,
			msgContains:    "installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := summarizeDepsCheck(tt.result)
			if check.Status != tt.expectedStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.expectedStatus)
			}
			if !strings.Contains(strings.ToLower(check.Message), strings.ToLower(tt.msgContains)) {
				t.Errorf("Message = %q, expected to contain %q", check.Message, tt.msgContains)
			}
		})
	}
}
