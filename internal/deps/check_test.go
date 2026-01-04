package deps

import (
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

func TestSummary(t *testing.T) {
	result := &CheckResult{
		Critical: []DependencyCheck{
			{Item: config.DependencyItem{Name: "dep1"}, Status: StatusInstalled},
			{Item: config.DependencyItem{Name: "dep2"}, Status: StatusMissing},
		},
		Core: []DependencyCheck{
			{Item: config.DependencyItem{Name: "dep3"}, Status: StatusInstalled},
		},
	}

	summary := result.Summary()

	// Should be "2 installed, 1 missing"
	if summary != "2 installed, 1 missing" {
		t.Errorf("Summary() = %s, want '2 installed, 1 missing'", summary)
	}
}
