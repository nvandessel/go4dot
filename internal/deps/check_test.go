package deps

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

func TestCheck(t *testing.T) {
	// Create a simple test config
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"}, // sh exists on all systems
			},
			Core: []config.DependencyItem{
				{Name: "definitely-does-not-exist-12345", Binary: "definitely-does-not-exist-12345"},
			},
		},
	}

	// Detect platform
	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	// Check dependencies
	result, err := Check(cfg, p)
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}

	// Verify results
	if len(result.Critical) != 1 {
		t.Errorf("len(Critical) = %d, want 1", len(result.Critical))
	}

	if len(result.Core) != 1 {
		t.Errorf("len(Core) = %d, want 1", len(result.Core))
	}

	// sh should be installed
	if result.Critical[0].Status != StatusInstalled {
		t.Errorf("Critical[0].Status = %v, want %v", result.Critical[0].Status, StatusInstalled)
	}

	// fake package should be missing
	if result.Core[0].Status != StatusMissing {
		t.Errorf("Core[0].Status = %v, want %v", result.Core[0].Status, StatusMissing)
	}
}

func TestCheckDependency(t *testing.T) {
	tests := []struct {
		name       string
		dep        config.DependencyItem
		wantStatus DepStatus
	}{
		{
			name: "Existing binary",
			dep: config.DependencyItem{
				Name:   "sh",
				Binary: "sh",
			},
			wantStatus: StatusInstalled,
		},
		{
			name: "Non-existent binary",
			dep: config.DependencyItem{
				Name:   "fake-binary-xyz",
				Binary: "fake-binary-xyz",
			},
			wantStatus: StatusMissing,
		},
		{
			name: "Binary defaults to name",
			dep: config.DependencyItem{
				Name: "sh",
			},
			wantStatus: StatusInstalled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := checkDependency(tt.dep)

			if check.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.wantStatus)
			}

			if check.Status == StatusInstalled && check.InstalledPath == "" {
				t.Error("InstalledPath should not be empty for installed dependency")
			}
		})
	}
}

func TestGetMissing(t *testing.T) {
	result := &CheckResult{
		Critical: []DependencyCheck{
			{Item: config.DependencyItem{Name: "installed1"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "missing1"}, Status: StatusMissing},
		},
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "installed2"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "missing2"}, Status: StatusMissing},
		},
		Optional: []DependencyCheck{
			{Item: config.DependencyItem{Name: "missing3"}, Status: StatusMissing},
		},
	}

	missing := result.GetMissing()

	if len(missing) != 3 {
		t.Errorf("len(GetMissing()) = %d, want 3", len(missing))
	}

	// Check that we got the right ones
	names := make(map[string]bool)
	for _, dep := range missing {
		names[dep.Item.Name] = true
	}

	expectedMissing := []string{"missing1", "missing2", "missing3"}
	for _, name := range expectedMissing {
		if !names[name] {
			t.Errorf("Expected %s to be in missing list", name)
		}
	}
}

func TestGetMissingCritical(t *testing.T) {
	result := &CheckResult{
		Critical: []DependencyCheck{
			{Item: config.DependencyItem{Name: "installed1"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "missing1"}, Status: StatusMissing},
		},
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "missing2"}, Status: StatusMissing},
		},
	}

	missing := result.GetMissingCritical()

	if len(missing) != 1 {
		t.Errorf("len(GetMissingCritical()) = %d, want 1", len(missing))
	}

	if missing[0].Item.Name != "missing1" {
		t.Errorf("GetMissingCritical()[0].Name = %s, want missing1", missing[0].Item.Name)
	}
}

func TestAllInstalled(t *testing.T) {
	tests := []struct {
		name   string
		result *CheckResult
		want   bool
	}{
		{
			name: "All installed",
			result: &CheckResult{
				Critical: []DependencyCheck{
					{Item: config.DependencyItem{Name: "dep1"}, Status: StatusInstalled},
				},
				Core: []DependencyCheck{
					{Item: config.DependencyItem{Name: "dep2"}, Status: StatusInstalled},
				},
			},
			want: true,
		},
		{
			name: "Some missing",
			result: &CheckResult{
				Critical: []DependencyCheck{
					{Item: config.DependencyItem{Name: "dep1"}, Status: StatusInstalled},
					{Item: config.DependencyItem{Name: "dep2"}, Status: StatusMissing},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.AllInstalled(); got != tt.want {
				t.Errorf("AllInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		installed string
		required  string
		want      bool
	}{
		{"0.10.1", "0.10.1", true},
		{"0.11.0", "0.10.1+", true},
		{"0.10.1", "0.10.1+", true},
		{"0.9.5", "0.10.1+", false},
		{"1.0.0", "0.10+", true},
		{"v0.11.0", "0.11+", true},
		{"0.11.0-dev", "0.11+", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%s", tt.installed, tt.required), func(t *testing.T) {
			if got := compareVersions(tt.installed, tt.required); got != tt.want {
				t.Errorf("compareVersions(%s, %s) = %v, want %v", tt.installed, tt.required, got, tt.want)
			}
		})
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  []int
	}{
		{"1.2.3", []int{1, 2, 3}},
		{"v1.2.3", []int{1, 2, 3}},
		{"0.10.1-dev", []int{0, 10, 1}},
		{"1.2", []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersion(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("parseVersion(%s) length = %d, want %d", tt.input, len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseVersion(%s)[%d] = %d, want %d", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCheckDependencyManual(t *testing.T) {
	tests := []struct {
		name       string
		dep        config.DependencyItem
		wantStatus DepStatus
	}{
		{
			name:       "Manual dep installed",
			dep:        config.DependencyItem{Name: "sh", Binary: "sh", Manual: true},
			wantStatus: StatusInstalled,
		},
		{
			name:       "Manual dep missing",
			dep:        config.DependencyItem{Name: "fake-manual-dep-xyz", Binary: "fake-manual-dep-xyz", Manual: true},
			wantStatus: StatusManualMissing,
		},
		{
			name:       "Non-manual dep missing",
			dep:        config.DependencyItem{Name: "fake-dep-xyz", Binary: "fake-dep-xyz", Manual: false},
			wantStatus: StatusMissing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := checkDependency(tt.dep)
			if check.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", check.Status, tt.wantStatus)
			}
		})
	}
}

func TestCheckDependencyManualVersionMismatch(t *testing.T) {
	// Use a binary that exists and supports --version, with a version that won't match.
	// "sh" is universally available and supports --version on most systems.
	dep := config.DependencyItem{
		Name:       "sh",
		Binary:     "sh",
		Version:    "999.999.999",
		VersionCmd: "--version",
		Manual:     true,
	}

	check := checkDependency(dep)
	// sh --version may either return a parseable version (mismatch) or fail.
	// Either StatusVersionMismatch or StatusCheckFailed is acceptable here,
	// as long as it's not StatusInstalled (which would mean version matched).
	if check.Status == StatusInstalled {
		t.Fatalf("expected status to NOT be %v for impossible version requirement", StatusInstalled)
	}
}

func TestCheckDependencyManualVersionError(t *testing.T) {
	dep := config.DependencyItem{
		Name:    "false",
		Binary:  "false",
		Version: "2.0.0",
		Manual:  true,
	}

	check := checkDependency(dep)
	if check.Status != StatusCheckFailed {
		t.Fatalf("expected status %v, got %v", StatusCheckFailed, check.Status)
	}
	if check.Error == nil {
		t.Fatal("expected version error to be set for manual dependency")
	}
}

func TestGetManualMissing(t *testing.T) {
	result := &CheckResult{
		Critical: []DependencyCheck{
			{Item: config.DependencyItem{Name: "installed1"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "manual1", Manual: true}, Status: StatusManualMissing},
		},
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "missing1"}, Status: StatusMissing},
			{Item: config.DependencyItem{Name: "manual2", Manual: true}, Status: StatusManualMissing},
		},
	}

	manual := result.GetManualMissing()
	if len(manual) != 2 {
		t.Errorf("len(GetManualMissing()) = %d, want 2", len(manual))
	}
}

func TestGetMissingExcludesManual(t *testing.T) {
	result := &CheckResult{
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "missing1"}, Status: StatusMissing},
			{Item: config.DependencyItem{Name: "manual1", Manual: true}, Status: StatusManualMissing},
		},
	}

	missing := result.GetMissing()
	if len(missing) != 1 {
		t.Errorf("len(GetMissing()) = %d, want 1 (should exclude manual)", len(missing))
	}
}

func TestGetMissingExcludesManualVersionMismatch(t *testing.T) {
	result := &CheckResult{
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "missing1"}, Status: StatusMissing},
			{Item: config.DependencyItem{Name: "manual1", Manual: true}, Status: StatusVersionMismatch},
		},
	}

	missing := result.GetMissing()
	if len(missing) != 1 {
		t.Fatalf("len(GetMissing()) = %d, want 1 (should exclude manual mismatch)", len(missing))
	}
	if missing[0].Item.Name != "missing1" {
		t.Errorf("GetMissing()[0].Name = %s, want missing1", missing[0].Item.Name)
	}
}

func TestAllInstalledIgnoresManual(t *testing.T) {
	result := &CheckResult{
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "installed1"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "manual1", Manual: true}, Status: StatusManualMissing},
		},
	}

	if !result.AllInstalled() {
		t.Error("AllInstalled() should return true when only manual deps are missing")
	}
}

func TestSummaryIncludesManual(t *testing.T) {
	result := &CheckResult{
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "installed1"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "manual1", Manual: true}, Status: StatusManualMissing},
		},
	}

	summary := result.Summary()
	if !strings.Contains(summary, "1 manual") {
		t.Errorf("Summary() = %q, expected to contain '1 manual'", summary)
	}
}

func TestCheckWithManualDeps(t *testing.T) {
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "sh", Binary: "sh"},
			},
			Core: []config.DependencyItem{
				{Name: "fake-manual-dep-xyz", Binary: "fake-manual-dep-xyz", Manual: true},
			},
		},
	}

	p, err := platform.Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	result, err := Check(cfg, p)
	if err != nil {
		t.Fatalf("Check() failed: %v", err)
	}

	if result.Critical[0].Status != StatusInstalled {
		t.Errorf("Critical[0].Status = %v, want %v", result.Critical[0].Status, StatusInstalled)
	}

	if result.Core[0].Status != StatusManualMissing {
		t.Errorf("Core[0].Status = %v, want %v", result.Core[0].Status, StatusManualMissing)
	}

	missing := result.GetMissing()
	for _, dep := range missing {
		if dep.Item.Name == "fake-manual-dep-xyz" {
			t.Error("GetMissing() should not include manual deps")
		}
	}

	manualMissing := result.GetManualMissing()
	if len(manualMissing) != 1 {
		t.Errorf("len(GetManualMissing()) = %d, want 1", len(manualMissing))
	}
}

func TestGetVersion_RejectsInjection(t *testing.T) {
	tests := []struct {
		name    string
		binary  string
		cmd     string
		wantErr bool
		errText string // expected substring in error message
	}{
		{
			name:    "malicious binary",
			binary:  "rm",
			cmd:     "-rf /",
			wantErr: true,
			errText: "invalid version command",
		},
		{
			name:    "flag injection binary",
			binary:  "--help",
			cmd:     "--version",
			wantErr: true,
			errText: "invalid binary name",
		},
		{
			name:    "path traversal binary",
			binary:  "../../../bin/sh",
			cmd:     "--version",
			wantErr: true,
			errText: "invalid binary name",
		},
		{
			name:    "shell metachar binary",
			binary:  "cmd;evil",
			cmd:     "--version",
			wantErr: true,
			errText: "invalid binary name",
		},
		{
			name:    "invalid version cmd",
			binary:  "git",
			cmd:     "--exec=malicious",
			wantErr: true,
			errText: "invalid version command",
		},
		{
			name:    "valid version check",
			binary:  "git",
			cmd:     "--version",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getVersion(tt.binary, tt.cmd)

			if tt.wantErr {
				if err == nil {
					t.Errorf("getVersion(%q, %q) expected error, got nil", tt.binary, tt.cmd)
					return
				}
				if tt.errText != "" && !strings.Contains(err.Error(), tt.errText) {
					t.Errorf("getVersion(%q, %q) error = %q, expected to contain %q",
						tt.binary, tt.cmd, err.Error(), tt.errText)
				}
			} else {
				// For valid inputs, the validation should pass. The command
				// itself may fail (e.g., binary not in PATH), but it should
				// NOT fail with a validation error.
				if err != nil {
					errMsg := err.Error()
					if strings.Contains(errMsg, "invalid binary name") ||
						strings.Contains(errMsg, "invalid version command") {
						t.Errorf("getVersion(%q, %q) returned validation error: %v",
							tt.binary, tt.cmd, err)
					}
					// Other errors (binary not found, etc.) are acceptable
				}
			}
		})
	}
}
