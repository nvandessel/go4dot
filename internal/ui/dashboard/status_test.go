package dashboard

import (
	"os"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestModel_getConfigStatusInfo(t *testing.T) {
	tests := []struct {
		name          string
		cfg           config.ConfigItem
		linkStatus    *stow.ConfigLinkStatus
		drift         *stow.DriftResult
		linkStatusMap map[string]*stow.ConfigLinkStatus
		expectedIcon  string
		expectedTags  []string
	}{
		{
			name: "Fully linked",
			cfg:  config.ConfigItem{Name: "test"},
			linkStatus: &stow.ConfigLinkStatus{
				LinkedCount: 2,
				TotalCount:  2,
			},
			expectedIcon: "✓",
		},
		{
			name: "Partially linked",
			cfg:  config.ConfigItem{Name: "test"},
			linkStatus: &stow.ConfigLinkStatus{
				LinkedCount: 1,
				TotalCount:  2,
			},
			expectedIcon: "◆",
		},
		{
			name: "Not linked",
			cfg:  config.ConfigItem{Name: "test"},
			linkStatus: &stow.ConfigLinkStatus{
				LinkedCount: 0,
				TotalCount:  2,
			},
			expectedIcon: "✗",
		},
		{
			name: "Conflicts",
			cfg:  config.ConfigItem{Name: "test"},
			linkStatus: &stow.ConfigLinkStatus{
				LinkedCount: 0,
				TotalCount:  2,
				Files: []stow.FileStatus{
					{RelPath: "file1", IsLinked: false, Issue: "file exists (conflict)"},
					{RelPath: "file2", IsLinked: false, Issue: "points elsewhere"},
				},
			},
			expectedIcon: "⚠",
			expectedTags: []string{"conflicts (2)"},
		},
		{
			name: "Missing module dependencies",
			cfg: config.ConfigItem{
				Name:      "test",
				DependsOn: []string{"dep1"},
			},
			linkStatus: &stow.ConfigLinkStatus{
				LinkedCount: 2,
				TotalCount:  2,
			},
			linkStatusMap: map[string]*stow.ConfigLinkStatus{
				"dep1": {LinkedCount: 1, TotalCount: 2}, // Not fully linked
			},
			expectedIcon: "✓",
			expectedTags: []string{"deps"},
		},
		{
			name: "Drift fallback",
			cfg:  config.ConfigItem{Name: "test"},
			drift: &stow.DriftResult{
				HasDrift:     true,
				NewFiles:     []string{"newfile"},
				CurrentCount: 1,
			},
			expectedIcon: "◆",
		},
		{
			name: "Missing external dependencies",
			cfg: config.ConfigItem{
				Name: "test",
				ExternalDeps: []config.ExternalDep{
					{Destination: ".config/missing"},
				},
			},
			linkStatus: &stow.ConfigLinkStatus{
				LinkedCount: 2,
				TotalCount:  2,
			},
			expectedIcon: "✓",
			expectedTags: []string{"external"},
		},
	}

	// Setup temporary HOME for external dep test
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tmpHome)
	defer func() { _ = os.Setenv("HOME", origHome) }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				linkStatus: tt.linkStatusMap,
			}
			if m.linkStatus == nil {
				m.linkStatus = make(map[string]*stow.ConfigLinkStatus)
			}

			info := m.getConfigStatusInfo(tt.cfg, tt.linkStatus, tt.drift)

			if !strings.Contains(info.icon, tt.expectedIcon) {
				t.Errorf("expected icon to contain %q, got %q", tt.expectedIcon, info.icon)
			}

			for _, tag := range tt.expectedTags {
				found := false
				for _, t := range info.statusTags {
					if strings.Contains(t, tag) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected tag %q not found in %v", tag, info.statusTags)
				}
			}
		})
	}
}

func TestModel_renderHelp_StatusLegend(t *testing.T) {
	m := New(&platform.Platform{}, nil, nil, nil, []config.ConfigItem{}, "", "", false)
	help := m.renderHelp()

	expected := []string{
		"Status Indicators",
		"Fully linked",
		"Partially linked",
		"Conflicts detected",
		"Not linked",
		"Status tags",
	}

	for _, s := range expected {
		if !strings.Contains(help, s) {
			t.Errorf("expected help to contain %q", s)
		}
	}
}
