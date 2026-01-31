package dashboard

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
)

func TestDetails_View(t *testing.T) {
	tests := []struct {
		name     string
		input    struct {
			configs     []config.ConfigItem
			selectedIdx int
			width       int
			height      int
		}
		expected struct {
			containsName bool
			containsDesc bool
		}
	}{
		{
			name: "renders config name and description",
			input: struct {
				configs     []config.ConfigItem
				selectedIdx int
				width       int
				height      int
			}{
				configs: []config.ConfigItem{
					{
						Name:        "test-config",
						Description: "A test configuration.",
					},
				},
				selectedIdx: 0,
				width:       80,
				height:      24,
			},
			expected: struct {
				containsName bool
				containsDesc bool
			}{
				containsName: true,
				containsDesc: true,
			},
		},
		{
			name: "handles empty config list",
			input: struct {
				configs     []config.ConfigItem
				selectedIdx int
				width       int
				height      int
			}{
				configs:     []config.ConfigItem{},
				selectedIdx: 0,
				width:       80,
				height:      24,
			},
			expected: struct {
				containsName bool
				containsDesc bool
			}{
				containsName: false,
				containsDesc: false,
			},
		},
		{
			name: "handles small terminal size",
			input: struct {
				configs     []config.ConfigItem
				selectedIdx int
				width       int
				height      int
			}{
				configs: []config.ConfigItem{
					{
						Name:        "vim",
						Description: "Vim configuration",
					},
				},
				selectedIdx: 0,
				width:       40,
				height:      10,
			},
			expected: struct {
				containsName bool
				containsDesc bool
			}{
				containsName: true,
				containsDesc: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := State{
				Configs: tt.input.configs,
			}
			d := NewDetails(state)
			d.SetSize(tt.input.width, tt.input.height)
			d.selectedIdx = tt.input.selectedIdx
			d.updateContent()
			view := d.View()

			if len(tt.input.configs) > 0 {
				configName := strings.ToUpper(tt.input.configs[0].Name)
				if tt.expected.containsName && !strings.Contains(view, configName) {
					t.Errorf("expected view to contain %q", configName)
				}
				if tt.expected.containsDesc && !strings.Contains(view, tt.input.configs[0].Description) {
					t.Errorf("expected view to contain %q", tt.input.configs[0].Description)
				}
			}
		})
	}
}
