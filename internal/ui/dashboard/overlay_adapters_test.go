package dashboard

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/stow"
)

func newTestForm() *huh.Form {
	name := ""
	return huh.NewForm(huh.NewGroup(huh.NewInput().Title("Name").Value(&name)))
}

func TestOverlayHelpContent(t *testing.T) {
	content := overlayHelpContent(Help{width: 48})
	if !strings.Contains(content, "Keyboard Shortcuts") {
		t.Fatalf("expected help content to include keyboard shortcuts, got %q", content)
	}
	if !strings.Contains(content, "Navigation") {
		t.Fatalf("expected help content to include navigation section, got %q", content)
	}
}

func TestOverlayMenuContent(t *testing.T) {
	m := NewMenu()
	m.SetSize(100, 40)

	content := overlayMenuContent(&m)

	if content == "" {
		t.Fatal("expected non-empty menu overlay content")
	}

	// Should contain the "More Commands" title from the list
	if !strings.Contains(content, "More Commands") {
		t.Errorf("expected menu content to include 'More Commands' title")
	}

	// Should contain the ESC hint
	if !strings.Contains(content, "ESC to close") {
		t.Errorf("expected menu content to include 'ESC to close' hint")
	}

	// Should contain menu items
	if !strings.Contains(content, "List Configs") {
		t.Errorf("expected menu content to include 'List Configs' item")
	}
	if !strings.Contains(content, "External Dependencies") {
		t.Errorf("expected menu content to include 'External Dependencies' item")
	}
	if !strings.Contains(content, "Uninstall") {
		t.Errorf("expected menu content to include 'Uninstall' item")
	}
}

func TestOverlayMenuContent_CompactWidth(t *testing.T) {
	m := NewMenu()
	m.SetSize(200, 80) // large terminal

	content := overlayMenuContent(&m)
	lines := strings.Split(content, "\n")

	// The content should be constrained to menuMaxWidth, not full terminal width.
	// Check that no line exceeds menuMaxWidth (with some tolerance for ANSI codes).
	for _, line := range lines {
		// Use lipgloss.Width which strips ANSI for visual width measurement.
		// But since we can't easily import lipgloss in tests, just check
		// that the plain text is reasonable.
		plain := strings.TrimRight(line, " ")
		if len(plain) > 0 && len(stripAnsiForTest(plain)) > menuMaxWidth+10 {
			t.Errorf("line exceeds compact width: visual width of %q is %d (max %d)",
				plain, len(stripAnsiForTest(plain)), menuMaxWidth)
		}
	}
}

func TestOverlayConfirmContent(t *testing.T) {
	confirm := &Confirm{
		title:       "Confirm Action",
		description: "Are you sure?",
		affirmative: "Yes",
		negative:    "No",
		selected:    0,
	}
	content := overlayConfirmContent(confirm)
	if !strings.Contains(content, "Confirm Action") {
		t.Fatalf("expected confirm content to include title, got %q", content)
	}
	if !strings.Contains(content, "Are you sure?") {
		t.Fatalf("expected confirm content to include description, got %q", content)
	}
}

func TestOverlayOnboardingContent_Steps(t *testing.T) {
	tests := []struct {
		name        string
		step        OnboardingStep
		expectText  string
		withForm    bool
		seedContent func(o *Onboarding)
	}{
		{name: "scanning", step: stepScanning, expectText: "Initializing go4dot"},
		{name: "writing", step: stepWriting, expectText: "Writing .go4dot.yaml"},
		{name: "complete", step: stepComplete, expectText: "Configuration Created"},
		{name: "metadata", step: stepMetadata, expectText: "Project Information", withForm: true},
		{name: "configs", step: stepConfigs, expectText: "Select Configurations", withForm: true},
		{
			name:       "external with count",
			step:       stepExternal,
			expectText: "External Dependencies (1 added)",
			withForm:   true,
			seedContent: func(o *Onboarding) {
				o.externalDeps = []config.ExternalDep{{Name: "plugin", URL: "https://example.com/repo"}}
			},
		},
		{name: "external details", step: stepExternalDetails, expectText: "Add External Dependency", withForm: true},
		{
			name:       "dependencies with count",
			step:       stepDependencies,
			expectText: "System Dependencies (1 added)",
			withForm:   true,
			seedContent: func(o *Onboarding) {
				o.systemDeps = []config.DependencyItem{{Name: "git", Binary: "git"}}
			},
		},
		{name: "dependencies details", step: stepDependenciesDetails, expectText: "Add System Dependency", withForm: true},
		{
			name:       "machine with count",
			step:       stepMachine,
			expectText: "Machine Configuration (1 added)",
			withForm:   true,
			seedContent: func(o *Onboarding) {
				o.machineConfigs = []config.MachinePrompt{{ID: "git", Description: "Git signing"}}
			},
		},
		{name: "machine details", step: stepMachineDetails, expectText: "Configure Machine Setting", withForm: true},
		{
			name:       "confirm",
			step:       stepConfirm,
			expectText: "Review Configuration",
			withForm:   true,
			seedContent: func(o *Onboarding) {
				o.metadata = config.Metadata{Name: "Dotfiles"}
				o.selectedConfigs = []string{"vim"}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewOnboarding("/tmp/dotfiles")
			o.step = tt.step
			if tt.withForm {
				o.form = newTestForm()
			}
			if tt.seedContent != nil {
				tt.seedContent(&o)
			}

			content := overlayOnboardingContent(&o)
			if !strings.Contains(content, tt.expectText) {
				t.Fatalf("expected onboarding content to include %q, got %q", tt.expectText, content)
			}
		})
	}
}

func TestOverlayConfigListContent(t *testing.T) {
	empty := overlayConfigListContent(&ConfigListView{})
	if empty != "" {
		t.Fatalf("expected empty content when config list is not ready, got %q", empty)
	}

	view := &ConfigListView{
		ready:    true,
		viewport: viewport.New(20, 5),
	}
	view.viewport.SetContent("vim")
	content := overlayConfigListContent(view)
	if !strings.Contains(content, "Configuration List") {
		t.Fatalf("expected config list content to include title, got %q", content)
	}
	if !strings.Contains(content, "vim") {
		t.Fatalf("expected config list content to include viewport content, got %q", content)
	}
}

func TestOverlayExternalContent(t *testing.T) {
	s := spinner.New()
	s.Spinner = spinner.Dot

	view := &ExternalView{
		ready:   true,
		loading: true,
		spinner: s,
	}
	content := overlayExternalContent(view)
	if !strings.Contains(content, "Loading status") {
		t.Fatalf("expected loading content to include status text, got %q", content)
	}

	view.loading = false
	view.viewport = viewport.New(20, 5)
	view.viewport.SetContent("external dep")
	content = overlayExternalContent(view)
	if !strings.Contains(content, "external dep") {
		t.Fatalf("expected external content to include viewport text, got %q", content)
	}
}

func TestOverlayMachineContent(t *testing.T) {
	view := &MachineView{
		ready:    true,
		viewport: viewport.New(20, 5),
	}
	view.viewport.SetContent("machine configs")
	content := overlayMachineContent(view)
	if !strings.Contains(content, "Machine Configuration") {
		t.Fatalf("expected machine content to include default title, got %q", content)
	}

	view.currentForm = newTestForm()
	view.currentConfig = &config.MachinePrompt{Description: "Git Signing"}
	content = overlayMachineContent(view)
	if !strings.Contains(content, "Git Signing") {
		t.Fatalf("expected machine form content to include config description, got %q", content)
	}
}

func TestOverlayConflictContent_DisplayPath(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	insideHome := filepath.Join(homeDir, ".vimrc")
	outsideHome := filepath.Join(t.TempDir(), "outside", "file.txt")

	conflicts := []stow.ConflictFile{
		{ConfigName: "vim", TargetPath: insideHome},
		{ConfigName: "vim", TargetPath: outsideHome},
	}
	view := NewConflictView(conflicts)
	view.width = 100

	content := overlayConflictContent(view)
	relPath, _ := filepath.Rel(homeDir, insideHome)
	expectedHome := filepath.ToSlash("~/" + relPath)
	if !strings.Contains(strings.ReplaceAll(content, "\\", "/"), expectedHome) {
		t.Fatalf("expected conflict content to include home-relative path %q, got %q", expectedHome, content)
	}
	if strings.Contains(content, "~/..") {
		t.Fatalf("expected conflict content to avoid home-relative path for outside file, got %q", content)
	}
	if !strings.Contains(content, outsideHome) {
		t.Fatalf("expected conflict content to include absolute outside path, got %q", content)
	}
}
