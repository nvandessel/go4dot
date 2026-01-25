package doctor

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status CheckStatus
		icon   string
	}{
		{StatusOK, "✓"},
		{StatusWarning, "⚠"},
		{StatusError, "✗"},
		{StatusSkipped, "⊘"},
	}

	for _, tt := range tests {
		if got := statusIcon(tt.status); got != tt.icon {
			t.Errorf("statusIcon(%v) = %q, want %q", tt.status, got, tt.icon)
		}
	}
}

func TestCheckResultReport(t *testing.T) {
	result := &CheckResult{
		Platform: &platform.Platform{
			OS:             "linux",
			Distro:         "fedora",
			PackageManager: "dnf",
		},
		Checks: []Check{
			{
				Name:    "Test Check 1",
				Status:  StatusOK,
				Message: "All good",
			},
			{
				Name:    "Test Check 2",
				Status:  StatusWarning,
				Message: "Something might be wrong",
				Fix:     "Run some command",
			},
		},
	}

	report := result.Report()

	// Check that report contains expected elements
	expectedStrings := []string{
		"go4dot Health Check",
		"linux",
		"fedora",
		"dnf",
		"Test Check 1",
		"All good",
		"Test Check 2",
		"Something might be wrong",
		"Run some command",
		"Summary",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(report, expected) {
			t.Errorf("Report missing expected string: %q", expected)
		}
	}

	t.Logf("Report:\n%s", report)
}

func TestCheckResultDetailedReport(t *testing.T) {
	result := &CheckResult{
		Platform: &platform.Platform{
			OS:             "linux",
			PackageManager: "dnf",
		},
		Checks: []Check{
			{
				Name:    "Test Check",
				Status:  StatusOK,
				Message: "OK",
			},
		},
		DriftSummary: &stow.DriftSummary{
			TotalConfigs:   1,
			DriftedConfigs: 1,
			Results: []stow.DriftResult{
				{
					ConfigName:   "git",
					HasDrift:     true,
					MissingFiles: []string{".gitconfig"},
				},
			},
		},
	}

	report := result.DetailedReport()

	// Should contain symlink details since there's an issue
	if !strings.Contains(report, "Symlink Drift Details") {
		t.Error("DetailedReport should contain symlink drift details for issues")
	}

	if !strings.Contains(report, "git") {
		t.Error("DetailedReport should mention the git config")
	}

	t.Logf("Detailed report:\n%s", report)
}

func TestCheckResultQuickReport(t *testing.T) {
	tests := []struct {
		name     string
		checks   []Check
		contains string
	}{
		{
			name: "All OK",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusOK},
			},
			contains: "2 checks passed",
		},
		{
			name: "Has warnings",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusWarning},
			},
			contains: "1 warnings",
		},
		{
			name: "Has errors",
			checks: []Check{
				{Status: StatusOK},
				{Status: StatusError},
			},
			contains: "1 errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CheckResult{Checks: tt.checks}
			quick := result.QuickReport()

			if !strings.Contains(quick, tt.contains) {
				t.Errorf("QuickReport() = %q, want to contain %q", quick, tt.contains)
			}
		})
	}
}

func TestCheckResultGetFixes(t *testing.T) {
	result := &CheckResult{
		Checks: []Check{
			{
				Status: StatusOK,
				Fix:    "This should not appear", // OK status, no fix needed
			},
			{
				Status: StatusError,
				Fix:    "Fix 1",
			},
			{
				Status: StatusWarning,
				Fix:    "Fix 2",
			},
			{
				Status: StatusError,
				Fix:    "Fix 1", // Duplicate, should not appear twice
			},
		},
	}

	fixes := result.GetFixes()

	if len(fixes) != 2 {
		t.Errorf("len(GetFixes()) = %d, want 2", len(fixes))
	}

	// Check that the right fixes are included
	hasF1, hasF2 := false, false
	for _, fix := range fixes {
		if fix == "Fix 1" {
			hasF1 = true
		}
		if fix == "Fix 2" {
			hasF2 = true
		}
	}

	if !hasF1 || !hasF2 {
		t.Errorf("GetFixes() = %v, expected to contain 'Fix 1' and 'Fix 2'", fixes)
	}
}

func TestCheckResultFixReport(t *testing.T) {
	tests := []struct {
		name     string
		checks   []Check
		contains string
	}{
		{
			name: "No fixes needed",
			checks: []Check{
				{Status: StatusOK},
			},
			contains: "No fixes needed",
		},
		{
			name: "Has fixes",
			checks: []Check{
				{Status: StatusError, Fix: "Run this command"},
			},
			contains: "Run this command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &CheckResult{Checks: tt.checks}
			report := result.FixReport()

			if !strings.Contains(report, tt.contains) {
				t.Errorf("FixReport() = %q, want to contain %q", report, tt.contains)
			}
		})
	}
}

func TestReportWithEmptyPlatform(t *testing.T) {
	result := &CheckResult{
		Checks: []Check{
			{
				Name:    "Test",
				Status:  StatusOK,
				Message: "OK",
			},
		},
	}

	// Should not panic with nil platform
	report := result.Report()

	if !strings.Contains(report, "go4dot Health Check") {
		t.Error("Report should contain header even without platform")
	}
}
