package dashboard

import (
	"errors"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestInstallResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   *InstallResult
		expected bool
	}{
		{
			name:     "No errors",
			result:   &InstallResult{},
			expected: false,
		},
		{
			name: "With deps failed",
			result: &InstallResult{
				DepsFailed: []deps.InstallError{{Error: errors.New("test")}},
			},
			expected: true,
		},
		{
			name: "With configs failed",
			result: &InstallResult{
				ConfigsFailed: []stow.StowError{{Error: errors.New("test")}},
			},
			expected: true,
		},
		{
			name: "With external failed",
			result: &InstallResult{
				ExternalFailed: []deps.ExternalError{{Error: errors.New("test")}},
			},
			expected: true,
		},
		{
			name: "With general errors",
			result: &InstallResult{
				Errors: []error{errors.New("test")},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestInstallResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   *InstallResult
		contains []string
	}{
		{
			name:     "Empty result",
			result:   &InstallResult{},
			contains: []string{},
		},
		{
			name: "With platform",
			result: &InstallResult{
				Platform: &platform.Platform{OS: "linux", PackageManager: "apt"},
			},
			contains: []string{"Platform: linux"},
		},
		{
			name: "With platform and distro",
			result: &InstallResult{
				Platform: &platform.Platform{OS: "linux", Distro: "ubuntu", PackageManager: "apt"},
			},
			contains: []string{"Platform: linux", "(ubuntu)"},
		},
		{
			name: "With dependencies",
			result: &InstallResult{
				DepsInstalled: []config.DependencyItem{{Name: "git"}, {Name: "vim"}},
				DepsFailed:    []deps.InstallError{{Error: errors.New("test")}},
			},
			contains: []string{"Dependencies:"},
		},
		{
			name: "With configs stowed",
			result: &InstallResult{
				ConfigsStowed: []string{"vim", "zsh"},
			},
			contains: []string{"Configs: 2 stowed"},
		},
		{
			name: "With configs stowed and adopted",
			result: &InstallResult{
				ConfigsStowed:  []string{"vim"},
				ConfigsAdopted: []string{"zsh"},
			},
			contains: []string{"1 stowed", "1 adopted"},
		},
		{
			name: "With external cloned",
			result: &InstallResult{
				ExternalCloned: []config.ExternalDep{{Name: "repo1"}, {Name: "repo2"}},
			},
			contains: []string{"External: 2 cloned"},
		},
		{
			name: "With external failed",
			result: &InstallResult{
				ExternalCloned: []config.ExternalDep{{Name: "repo1"}},
				ExternalFailed: []deps.ExternalError{{Dep: config.ExternalDep{Name: "repo2"}, Error: errors.New("test")}},
			},
			contains: []string{"External: 1 cloned, 1 failed"},
		},
		{
			name: "With configs failed only",
			result: &InstallResult{
				ConfigsFailed: []stow.StowError{{ConfigName: "vim", Error: errors.New("test")}},
			},
			contains: []string{"Configs: 0 stowed, 1 failed"},
		},
		{
			name: "With machine configs",
			result: &InstallResult{
				MachineConfigs: []machine.RenderResult{{ID: "ssh", Destination: "/etc/ssh"}},
			},
			contains: []string{"Machine configs: 1 configured"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := tt.result.Summary()
			for _, want := range tt.contains {
				if !strings.Contains(summary, want) {
					t.Errorf("Summary() = %q, expected to contain %q", summary, want)
				}
			}
		})
	}
}
