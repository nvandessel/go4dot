package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/stow"
)

func TestOverlayConflictContent_TildePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home dir: %v", err)
	}

	tests := []struct {
		name         string
		targetPath   string
		expectsTilde bool
	}{
		{
			name:         "path inside home gets ~/ prefix",
			targetPath:   filepath.Join(home, ".bashrc"),
			expectsTilde: true,
		},
		{
			name:         "path outside home keeps absolute path",
			targetPath:   "/etc/hosts",
			expectsTilde: false,
		},
		{
			name:         "nested path inside home gets ~/ prefix",
			targetPath:   filepath.Join(home, ".config", "nvim", "init.vim"),
			expectsTilde: true,
		},
		{
			name:         "path with similar prefix but outside home",
			targetPath:   home + "-other/file.txt",
			expectsTilde: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &ConflictView{
				conflicts: []stow.ConflictFile{
					{ConfigName: "test", TargetPath: tt.targetPath},
				},
				byConfig: map[string][]stow.ConflictFile{
					"test": {{ConfigName: "test", TargetPath: tt.targetPath}},
				},
				configNames: []string{"test"},
				width:       80,
				height:      24,
			}

			result := overlayConflictContent(v)

			if tt.expectsTilde {
				if !strings.Contains(result, "~/") {
					t.Errorf("expected ~/ prefix for path %q inside home, but not found in output", tt.targetPath)
				}
				// Should not contain the full absolute home path
				relPath, _ := filepath.Rel(home, tt.targetPath)
				expected := "~/" + relPath
				if !strings.Contains(result, expected) {
					t.Errorf("expected %q in output, not found", expected)
				}
			} else {
				// Should contain the absolute path, not ~/
				if strings.Contains(result, "~/") {
					t.Errorf("did not expect ~/ prefix for path %q outside home, but found in output", tt.targetPath)
				}
				if !strings.Contains(result, tt.targetPath) {
					t.Errorf("expected absolute path %q in output, not found", tt.targetPath)
				}
			}
		})
	}
}

func TestOverlayConflictContent_MultipleConfigs(t *testing.T) {
	home, _ := os.UserHomeDir()

	v := &ConflictView{
		conflicts: []stow.ConflictFile{
			{ConfigName: "vim", TargetPath: filepath.Join(home, ".vimrc")},
			{ConfigName: "zsh", TargetPath: filepath.Join(home, ".zshrc")},
		},
		byConfig: map[string][]stow.ConflictFile{
			"vim": {{ConfigName: "vim", TargetPath: filepath.Join(home, ".vimrc")}},
			"zsh": {{ConfigName: "zsh", TargetPath: filepath.Join(home, ".zshrc")}},
		},
		configNames: []string{"vim", "zsh"},
		width:       80,
		height:      24,
	}

	result := overlayConflictContent(v)

	if !strings.Contains(result, "vim:") {
		t.Error("expected 'vim:' config name in output")
	}
	if !strings.Contains(result, "zsh:") {
		t.Error("expected 'zsh:' config name in output")
	}
	if !strings.Contains(result, "2 conflicting") {
		t.Error("expected '2 conflicting' count in output")
	}
}

func TestOverlayConflictContent_ButtonSelection(t *testing.T) {
	tests := []struct {
		name        string
		selectedIdx int
	}{
		{name: "backup selected", selectedIdx: 0},
		{name: "delete selected", selectedIdx: 1},
		{name: "cancel selected", selectedIdx: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &ConflictView{
				conflicts: []stow.ConflictFile{
					{ConfigName: "test", TargetPath: "/tmp/test"},
				},
				byConfig: map[string][]stow.ConflictFile{
					"test": {{ConfigName: "test", TargetPath: "/tmp/test"}},
				},
				configNames: []string{"test"},
				width:       80,
				height:      24,
				selectedIdx: tt.selectedIdx,
			}

			result := overlayConflictContent(v)

			if !strings.Contains(result, "Backup") {
				t.Error("expected 'Backup' button in output")
			}
			if !strings.Contains(result, "Delete") {
				t.Error("expected 'Delete' button in output")
			}
			if !strings.Contains(result, "Cancel") {
				t.Error("expected 'Cancel' button in output")
			}
		})
	}
}

func TestOverlayConflictContent_ManyFiles(t *testing.T) {
	home, _ := os.UserHomeDir()

	var conflicts []stow.ConflictFile
	for i := 0; i < 12; i++ {
		conflicts = append(conflicts, stow.ConflictFile{
			ConfigName: "big",
			TargetPath: filepath.Join(home, ".config", "file"+string(rune('a'+i))),
		})
	}

	v := &ConflictView{
		conflicts: conflicts,
		byConfig: map[string][]stow.ConflictFile{
			"big": conflicts,
		},
		configNames: []string{"big"},
		width:       80,
		height:      24,
	}

	result := overlayConflictContent(v)

	// Should show "and X more" for truncated file list
	if !strings.Contains(result, "more") {
		t.Error("expected truncation indicator for many files")
	}
}

func TestOverlayHelpContent(t *testing.T) {
	h := Help{width: 80}
	result := overlayHelpContent(h)

	expectedSections := []string{
		"Navigation",
		"Actions",
		"Selection & Filter",
		"Other",
	}

	for _, section := range expectedSections {
		if !strings.Contains(result, section) {
			t.Errorf("expected help content to contain section %q", section)
		}
	}

	expectedKeys := []string{"enter", "space", "q / esc"}
	for _, k := range expectedKeys {
		if !strings.Contains(result, k) {
			t.Errorf("expected help content to contain key %q", k)
		}
	}
}

func TestOverlayHelpContent_NarrowWidth(t *testing.T) {
	h := Help{width: 30}
	result := overlayHelpContent(h)

	if result == "" {
		t.Error("expected non-empty help content even at narrow width")
	}
}

func TestOverlayHelpContent_ZeroWidth(t *testing.T) {
	h := Help{width: 0}
	result := overlayHelpContent(h)

	if result == "" {
		t.Error("expected non-empty help content at zero width")
	}
}

func TestOverlayConfirmContent(t *testing.T) {
	tests := []struct {
		name     string
		confirm  *Confirm
		contains []string
	}{
		{
			name: "yes selected",
			confirm: &Confirm{
				title:       "Confirm Action",
				description: "Are you sure?",
				affirmative: "Yes",
				negative:    "No",
				selected:    0,
				width:       80,
			},
			contains: []string{"Confirm Action", "Are you sure?", "Yes", "No"},
		},
		{
			name: "no selected",
			confirm: &Confirm{
				title:       "Delete files?",
				description: "This cannot be undone",
				affirmative: "Delete",
				negative:    "Keep",
				selected:    1,
				width:       80,
			},
			contains: []string{"Delete files?", "This cannot be undone", "Delete", "Keep"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := overlayConfirmContent(tt.confirm)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected confirm content to contain %q", expected)
				}
			}
		})
	}
}

func TestOverlayConfirmContent_NarrowWidth(t *testing.T) {
	c := &Confirm{
		title:       "Confirm",
		description: "Sure?",
		affirmative: "Yes",
		negative:    "No",
		selected:    0,
		width:       40,
	}

	result := overlayConfirmContent(c)
	if result == "" {
		t.Error("expected non-empty confirm content at narrow width")
	}
}
