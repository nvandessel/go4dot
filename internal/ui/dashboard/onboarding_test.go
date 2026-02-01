package dashboard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
)

func TestOnboarding_New(t *testing.T) {
	o := NewOnboarding("/tmp/test")

	if o.path != "/tmp/test" {
		t.Errorf("expected path '/tmp/test', got '%s'", o.path)
	}

	if o.step != stepScanning {
		t.Errorf("expected initial step to be stepScanning, got %d", o.step)
	}

	if o.metadata.Version != "1.0.0" {
		t.Errorf("expected metadata version '1.0.0', got '%s'", o.metadata.Version)
	}
}

func TestOnboarding_View_Scanning(t *testing.T) {
	o := NewOnboarding("/tmp/test")
	o.width = 80
	o.height = 24

	view := o.View()

	if !strings.Contains(view, "Scanning") {
		t.Error("expected view to contain 'Scanning'")
	}
}

func TestOnboarding_ScanDirectory(t *testing.T) {
	// Create a temp directory with some test directories
	tmpDir, err := os.MkdirTemp("", "onboarding-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create some directories
	if err := os.Mkdir(filepath.Join(tmpDir, "vim"), 0755); err != nil {
		t.Fatalf("failed to create vim dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "zsh"), 0755); err != nil {
		t.Fatalf("failed to create zsh dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755); err != nil {
		t.Fatalf("failed to create .git dir: %v", err)
	}

	configs, err := scanDirectoryForConfigs(tmpDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Should find vim and zsh, not .git
	if len(configs) != 2 {
		t.Errorf("expected 2 configs, got %d", len(configs))
	}

	// Check that .git is not in the list
	for _, c := range configs {
		if c.Name == ".git" {
			t.Error(".git should be ignored")
		}
	}
}

func TestOnboarding_Slugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"my-plugin", "my-plugin"},
		{"My Plugin 123", "my-plugin-123"},
		{"  spaces  ", "spaces"},
		{"UPPERCASE", "uppercase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			if result != tt.expected {
				t.Errorf("slugify(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOnboarding_WindowResize(t *testing.T) {
	o := NewOnboarding("/tmp/test")

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	model, _ := o.Update(msg)
	updated := model.(*Onboarding)

	if updated.width != 100 {
		t.Errorf("expected width 100, got %d", updated.width)
	}

	if updated.height != 50 {
		t.Errorf("expected height 50, got %d", updated.height)
	}
}

func TestOnboarding_ScannedConfigsMsg(t *testing.T) {
	o := NewOnboarding("/tmp/test")
	o.width = 80
	o.height = 24

	msg := scannedConfigsMsg{
		configs: []config.ConfigItem{
			{Name: "vim"},
			{Name: "zsh"},
		},
	}

	model, _ := o.Update(msg)
	updated := model.(*Onboarding)

	if updated.step != stepMetadata {
		t.Errorf("expected step to be stepMetadata after scan, got %d", updated.step)
	}

	if len(updated.scannedConfigs) != 2 {
		t.Errorf("expected 2 scanned configs, got %d", len(updated.scannedConfigs))
	}
}
