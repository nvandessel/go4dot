package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/ui"
	"gopkg.in/yaml.v3"
)

// OnboardingStep represents the current step in the onboarding wizard
type OnboardingStep int

const (
	stepScanning OnboardingStep = iota
	stepMetadata
	stepConfigs
	stepExternal
	stepDependencies
	stepMachine
	stepConfirm
	stepWriting
	stepComplete
)

// OnboardingCompleteMsg is sent when onboarding finishes
type OnboardingCompleteMsg struct {
	ConfigPath string
	Config     *config.Config
	Error      error
}

// scannedConfigsMsg is sent when directory scanning completes
type scannedConfigsMsg struct {
	configs []config.ConfigItem
	err     error
}

// configWrittenMsg is sent when config file is written
type configWrittenMsg struct {
	path string
	err  error
}

// Onboarding is the model for the multi-step onboarding wizard
type Onboarding struct {
	width    int
	height   int
	step     OnboardingStep
	spinner  spinner.Model
	form     *huh.Form
	path     string
	quitting bool

	// Collected data
	scannedConfigs  []config.ConfigItem
	selectedConfigs []string
	metadata        config.Metadata
	externalDeps    []config.ExternalDep
	systemDeps      []config.DependencyItem
	machineConfigs  []config.MachinePrompt

	// Current external/dep being added
	currentExternal config.ExternalDep
	currentDep      config.DependencyItem

	// Flags for looping forms
	addMoreExternal bool
	addMoreDeps     bool
	addMoreMachine  bool

	// Error tracking
	lastError error
}

// NewOnboarding creates a new onboarding wizard
func NewOnboarding(path string) Onboarding {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	return Onboarding{
		path:     path,
		step:     stepScanning,
		spinner:  s,
		metadata: config.Metadata{Version: "1.0.0"},
	}
}

func (o Onboarding) Init() tea.Cmd {
	return tea.Batch(
		o.spinner.Tick,
		o.scanDirectory,
	)
}

func (o *Onboarding) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.width = msg.Width
		o.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c", "esc"))):
			if o.step == stepComplete || o.step == stepWriting {
				// Don't cancel during these steps
			} else {
				o.quitting = true
				return o, func() tea.Msg {
					return OnboardingCompleteMsg{Error: fmt.Errorf("cancelled")}
				}
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		o.spinner, cmd = o.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case scannedConfigsMsg:
		if msg.err != nil {
			o.lastError = msg.err
			return o, func() tea.Msg {
				return OnboardingCompleteMsg{Error: msg.err}
			}
		}
		o.scannedConfigs = msg.configs
		o.step = stepMetadata
		o.form = o.createMetadataForm()
		cmds = append(cmds, o.form.Init())

	case configWrittenMsg:
		if msg.err != nil {
			o.lastError = msg.err
			return o, func() tea.Msg {
				return OnboardingCompleteMsg{Error: msg.err}
			}
		}
		o.step = stepComplete
		return o, func() tea.Msg {
			return OnboardingCompleteMsg{
				ConfigPath: msg.path,
				Config:     o.buildConfig(),
			}
		}
	}

	// Handle form updates
	if o.form != nil {
		form, cmd := o.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			o.form = f
		}
		cmds = append(cmds, cmd)

		// Check for form completion
		if o.form.State == huh.StateCompleted {
			return o.handleFormComplete()
		}

		// Check for form abort
		if o.form.State == huh.StateAborted {
			o.quitting = true
			return o, func() tea.Msg {
				return OnboardingCompleteMsg{Error: fmt.Errorf("cancelled")}
			}
		}
	}

	return o, tea.Batch(cmds...)
}

func (o *Onboarding) handleFormComplete() (tea.Model, tea.Cmd) {
	switch o.step {
	case stepMetadata:
		// Apply defaults
		if o.metadata.Name == "" {
			o.metadata.Name = filepath.Base(o.path)
		}
		if o.metadata.Author == "" {
			o.metadata.Author = os.Getenv("USER")
		}
		if o.metadata.Description == "" {
			o.metadata.Description = "My personal dotfiles"
		}

		if len(o.scannedConfigs) > 0 {
			o.step = stepConfigs
			o.form = o.createConfigsForm()
			return o, o.form.Init()
		}
		// Skip to external deps if no configs found
		o.step = stepExternal
		o.form = o.createExternalPromptForm()
		return o, o.form.Init()

	case stepConfigs:
		o.step = stepExternal
		o.form = o.createExternalPromptForm()
		return o, o.form.Init()

	case stepExternal:
		if o.addMoreExternal {
			// User wants to add an external dep
			o.form = o.createExternalDetailsForm()
			return o, o.form.Init()
		}
		// Done with external, move to system deps
		o.step = stepDependencies
		o.form = o.createDepsPromptForm()
		return o, o.form.Init()

	case stepDependencies:
		if o.addMoreDeps {
			// User wants to add a system dep
			o.form = o.createDepsDetailsForm()
			return o, o.form.Init()
		}
		// Done with deps, move to machine config
		o.step = stepMachine
		o.form = o.createMachinePromptForm()
		return o, o.form.Init()

	case stepMachine:
		if o.addMoreMachine {
			// User wants to add machine config
			o.form = o.createMachineDetailsForm()
			return o, o.form.Init()
		}
		// Done with machine, move to confirm
		o.step = stepConfirm
		o.form = o.createConfirmForm()
		return o, o.form.Init()

	case stepConfirm:
		// Write config
		o.step = stepWriting
		return o, o.writeConfig
	}

	return o, nil
}

func (o Onboarding) View() string {
	if o.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(1, 0)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	var content string

	switch o.step {
	case stepScanning:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("🔍 Initializing go4dot"),
			"",
			o.spinner.View()+" Scanning for dotfiles...",
		)

	case stepWriting:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("✍️ Creating Configuration"),
			"",
			o.spinner.View()+" Writing .go4dot.yaml...",
		)

	case stepComplete:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("✅ Configuration Created"),
			"",
			ui.SuccessStyle.Render("Your .go4dot.yaml has been created!"),
			"",
			subtitleStyle.Render("Run 'g4d install' to set up your dotfiles."),
		)

	case stepMetadata:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("📝 Project Information"),
			subtitleStyle.Render(fmt.Sprintf("Found %d potential configs", len(o.scannedConfigs))),
			"",
			o.form.View(),
		)

	case stepConfigs:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("📦 Select Configurations"),
			subtitleStyle.Render("Choose which configs to manage"),
			"",
			o.form.View(),
		)

	case stepExternal:
		title := "🔗 External Dependencies"
		if len(o.externalDeps) > 0 {
			title = fmt.Sprintf("🔗 External Dependencies (%d added)", len(o.externalDeps))
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			subtitleStyle.Render("Git repos for plugins, themes, etc."),
			"",
			o.form.View(),
		)

	case stepDependencies:
		title := "⚙️ System Dependencies"
		if len(o.systemDeps) > 0 {
			title = fmt.Sprintf("⚙️ System Dependencies (%d added)", len(o.systemDeps))
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			subtitleStyle.Render("Required packages (neovim, tmux, etc.)"),
			"",
			o.form.View(),
		)

	case stepMachine:
		title := "🖥️ Machine Configuration"
		if len(o.machineConfigs) > 0 {
			title = fmt.Sprintf("🖥️ Machine Configuration (%d added)", len(o.machineConfigs))
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			subtitleStyle.Render("Machine-specific settings (git signing, etc.)"),
			"",
			o.form.View(),
		)

	case stepConfirm:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("📋 Review Configuration"),
			"",
			o.renderSummary(),
			"",
			o.form.View(),
		)
	}

	// Center content in available space
	return lipgloss.Place(
		o.width,
		o.height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// Form creation methods

func (o *Onboarding) createMetadataForm() *huh.Form {
	defaultName := filepath.Base(o.path)
	defaultAuthor := os.Getenv("USER")

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project Name").
				Value(&o.metadata.Name).
				Placeholder(defaultName),
			huh.NewInput().
				Title("Author").
				Value(&o.metadata.Author).
				Placeholder(defaultAuthor),
			huh.NewInput().
				Title("Description").
				Value(&o.metadata.Description).
				Placeholder("My personal dotfiles"),
			huh.NewInput().
				Title("Repository URL").
				Value(&o.metadata.Repository),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createConfigsForm() *huh.Form {
	var options []huh.Option[string]
	for _, c := range o.scannedConfigs {
		options = append(options, huh.NewOption(c.Name, c.Name).Selected(true))
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select configurations to manage").
				Options(options...).
				Value(&o.selectedConfigs),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createExternalPromptForm() *huh.Form {
	prompt := "Would you like to add external dependencies?"
	if len(o.externalDeps) > 0 {
		prompt = "Add another external dependency?"
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(prompt).
				Description("Git repos for plugins, themes, etc.").
				Value(&o.addMoreExternal),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createExternalDetailsForm() *huh.Form {
	o.currentExternal = config.ExternalDep{
		Method:        "clone",
		MergeStrategy: "overwrite",
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Git Repository URL").
				Placeholder("https://github.com/example/plugin").
				Value(&o.currentExternal.URL).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("URL is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Name").
				Placeholder("My Plugin").
				Value(&o.currentExternal.Name),
			huh.NewInput().
				Title("Destination").
				Description("Use @repoRoot/path to clone inside dotfiles").
				Placeholder("@repoRoot/plugins/my-plugin").
				Value(&o.currentExternal.Destination),
			huh.NewSelect[string]().
				Title("Method").
				Options(
					huh.NewOption("Clone (standard git clone)", "clone"),
					huh.NewOption("Copy (clone to temp, copy to dest)", "copy"),
				).
				Value(&o.currentExternal.Method),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createDepsPromptForm() *huh.Form {
	prompt := "Would you like to add system dependencies?"
	if len(o.systemDeps) > 0 {
		prompt = "Add another system dependency?"
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(prompt).
				Description("Required packages (neovim, tmux, etc.)").
				Value(&o.addMoreDeps),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createDepsDetailsForm() *huh.Form {
	o.currentDep = config.DependencyItem{}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Dependency Name").
				Placeholder("neovim").
				Value(&o.currentDep.Name).
				Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("name is required")
					}
					return nil
				}),
			huh.NewInput().
				Title("Binary Name").
				Placeholder("nvim").
				Value(&o.currentDep.Binary),
			huh.NewInput().
				Title("Required Version (optional)").
				Placeholder("0.11+").
				Value(&o.currentDep.Version),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createMachinePromptForm() *huh.Form {
	prompt := "Would you like to add machine-specific configurations?"
	if len(o.machineConfigs) > 0 {
		prompt = "Add another machine configuration?"
	}
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(prompt).
				Description("Machine-specific settings (git signing, etc.)").
				Value(&o.addMoreMachine),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createMachineDetailsForm() *huh.Form {
	var choice string
	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a preset or create custom").
				Options(
					huh.NewOption("Git Signing (Name, Email, GPG Key)", "git-signing"),
					huh.NewOption("Custom", "custom"),
				).
				Value(&choice),
		),
	).WithWidth(60).WithShowHelp(false)
}

func (o *Onboarding) createConfirmForm() *huh.Form {
	var confirm bool
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create configuration file?").
				Description("This will write .go4dot.yaml to your dotfiles directory").
				Affirmative("Yes, create").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithWidth(60).WithShowHelp(false)
}

// Helper methods

func (o *Onboarding) scanDirectory() tea.Msg {
	absPath, err := filepath.Abs(o.path)
	if err != nil {
		return scannedConfigsMsg{err: fmt.Errorf("failed to resolve path: %w", err)}
	}

	configs, err := scanDirectoryForConfigs(absPath)
	if err != nil {
		return scannedConfigsMsg{err: err}
	}

	return scannedConfigsMsg{configs: configs}
}

func (o *Onboarding) writeConfig() tea.Msg {
	cfg := o.buildConfig()

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return configWrittenMsg{err: fmt.Errorf("failed to generate YAML: %w", err)}
	}

	content := fmt.Sprintf("# Generated by go4dot\n# Edit this file to customize your dotfiles management\n\n%s", string(data))

	configFile := filepath.Join(o.path, config.ConfigFileName)
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		return configWrittenMsg{err: fmt.Errorf("failed to write config file: %w", err)}
	}

	return configWrittenMsg{path: configFile}
}

func (o *Onboarding) buildConfig() *config.Config {
	// Build selected configs list
	var selectedConfigItems []config.ConfigItem
	configMap := make(map[string]config.ConfigItem)
	for _, c := range o.scannedConfigs {
		configMap[c.Name] = c
	}
	for _, name := range o.selectedConfigs {
		if c, ok := configMap[name]; ok {
			selectedConfigItems = append(selectedConfigItems, c)
		}
	}

	return &config.Config{
		SchemaVersion: "1.0",
		Metadata:      o.metadata,
		Dependencies: config.Dependencies{
			Critical: []config.DependencyItem{
				{Name: "git", Binary: "git"},
				{Name: "stow", Binary: "stow"},
			},
			Core: o.systemDeps,
		},
		Configs: config.ConfigGroups{
			Core: selectedConfigItems,
		},
		External:      o.externalDeps,
		MachineConfig: o.machineConfigs,
	}
}

func (o *Onboarding) renderSummary() string {
	var lines []string
	labelStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(ui.TextColor)

	lines = append(lines, labelStyle.Render("Project: ")+valueStyle.Render(o.metadata.Name))
	if o.metadata.Author != "" {
		lines = append(lines, labelStyle.Render("Author: ")+valueStyle.Render(o.metadata.Author))
	}
	lines = append(lines, labelStyle.Render("Configs: ")+valueStyle.Render(fmt.Sprintf("%d selected", len(o.selectedConfigs))))
	lines = append(lines, labelStyle.Render("External: ")+valueStyle.Render(fmt.Sprintf("%d dependencies", len(o.externalDeps))))
	lines = append(lines, labelStyle.Render("System deps: ")+valueStyle.Render(fmt.Sprintf("%d packages", len(o.systemDeps))))
	lines = append(lines, labelStyle.Render("Machine configs: ")+valueStyle.Render(fmt.Sprintf("%d templates", len(o.machineConfigs))))

	return strings.Join(lines, "\n")
}

// scanDirectoryForConfigs scans a directory for potential dotfile configs
func scanDirectoryForConfigs(root string) ([]config.ConfigItem, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var items []config.ConfigItem

	// Directories to always ignore
	ignored := map[string]bool{
		".git": true, ".github": true, ".gitlab": true, ".svn": true,
		".idea": true, ".vscode": true,
		"bin": true, "build": true, "dist": true, "node_modules": true,
		"vendor": true, "target": true, "__pycache__": true, ".cache": true,
		config.ConfigFileName: true, "README.md": true, "LICENSE": true,
		"Makefile": true, "go.mod": true, "go.sum": true,
		"package.json": true, "Cargo.toml": true,
		"test": true, "sandbox": true,
	}

	// Valid hidden directories that are likely dotfile configs
	validHiddenDirs := map[string]bool{
		".config": true, ".local": true, ".vim": true, ".nvim": true,
		".emacs.d": true, ".tmux": true, ".ssh": true, ".gnupg": true,
		".fonts": true, ".themes": true, ".icons": true,
	}

	for _, entry := range entries {
		name := entry.Name()

		if ignored[name] {
			continue
		}

		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories unless they're valid dotfile configs
		if len(name) > 1 && name[0] == '.' {
			if !validHiddenDirs[name] {
				continue
			}
		}

		items = append(items, config.ConfigItem{
			Name:        name,
			Path:        name,
			Description: fmt.Sprintf("%s configuration", name),
			Platforms:   []string{"linux", "macos"},
		})
	}

	return items, nil
}

// slugify converts a string to a URL-friendly slug
func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
