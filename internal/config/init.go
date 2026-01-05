package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"gopkg.in/yaml.v3"
)

var (
	subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
)

// InitConfig scans the directory and interactively generates a configuration
// using standard input/output
func InitConfig(path string) error {
	return InitConfigWithIO(path, os.Stdin, os.Stdout)
}

// InitConfigWithIO allows specifying input/output for testing
func InitConfigWithIO(path string, in io.Reader, out io.Writer) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}

	configFile := filepath.Join(absPath, ConfigFileName)
	if _, err := os.Stat(configFile); err == nil {
		var overwrite bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("%s already exists. Overwrite?", ConfigFileName)).
					Value(&overwrite),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}
		if !overwrite {
			fmt.Fprintln(out, "Aborted.")
			return nil
		}
	}

	fmt.Fprintf(out, "🔍 Scanning %s for dotfiles...\n", absPath)
	configs, err := scanDirectory(absPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Found %d potential config directories.\n\n", len(configs))

	// Collect Metadata
	meta := Metadata{
		Version: "1.0.0",
	}

	defaultName := filepath.Base(absPath)
	defaultAuthor := os.Getenv("USER")

	// Metadata Form
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Project Name").
				Value(&meta.Name).
				Placeholder(defaultName),
			huh.NewInput().
				Title("Author").
				Value(&meta.Author).
				Placeholder(defaultAuthor),
			huh.NewInput().
				Title("Description").
				Value(&meta.Description).
				Placeholder("My personal dotfiles"),
			huh.NewInput().
				Title("Repository URL").
				Value(&meta.Repository),
		),
	).WithInput(in).WithOutput(out).Run()

	if err != nil {
		return err
	}

	// Apply defaults if empty
	if meta.Name == "" {
		meta.Name = defaultName
	}
	if meta.Author == "" {
		meta.Author = defaultAuthor
	}
	if meta.Description == "" {
		meta.Description = "My personal dotfiles"
	}

	// Filter Configs using MultiSelect
	var selectedConfigs []ConfigItem
	if len(configs) > 0 {
		var selectedNames []string
		var options []huh.Option[string]

		for _, c := range configs {
			options = append(options, huh.NewOption(c.Name, c.Name).Selected(true))
		}

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewMultiSelect[string]().
					Title("Select configurations to manage").
					Options(options...).
					Value(&selectedNames),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}

		configMap := make(map[string]ConfigItem)
		for _, c := range configs {
			configMap[c.Name] = c
		}

		for _, name := range selectedNames {
			if c, ok := configMap[name]; ok {
				selectedConfigs = append(selectedConfigs, c)
			}
		}
	}

	// External Dependencies
	var externalDeps []ExternalDep
	var addExternal bool

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to add external dependencies (e.g. plugins, themes)?").
				Value(&addExternal),
		),
	).WithInput(in).WithOutput(out).Run()

	if err != nil {
		return err
	}

	for addExternal {
		var name, url, dest, method, strategy string

		// Default values
		method = "clone"
		strategy = "overwrite"

		// Ask for URL first to auto-detect name
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Git Repository URL").
					Description("Enter the full clone URL").
					Placeholder("https://github.com/example/plugin").
					Value(&url),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}

		// Auto-detect name from URL
		if url != "" {
			cleanUrl := strings.TrimRight(url, "/")
			cleanUrl = strings.TrimSuffix(cleanUrl, ".git")

			parts := strings.Split(cleanUrl, "/")
			if len(parts) >= 2 {
				repo := parts[len(parts)-1]
				owner := parts[len(parts)-2]

				// Basic check to see if 'owner' is likely a user/org and not a domain or protocol
				if !strings.Contains(owner, ".") && !strings.Contains(owner, ":") && owner != "" {
					name = fmt.Sprintf("%s %s", owner, repo)
				} else {
					name = repo
				}
			} else if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}

		// Generate default destination placeholder
		defaultDest := "@repoRoot/plugins/" + slugify(name)
		if name == "" {
			defaultDest = "@repoRoot/plugins/my-plugin"
		}

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Name").
					Placeholder("My Plugin").
					Value(&name),
				huh.NewInput().
					Title("Destination").
					Description("Use @repoRoot/path to clone inside dotfiles").
					Placeholder(defaultDest).
					Value(&dest),
				huh.NewSelect[string]().
					Title("Method").
					Options(
						huh.NewOption(fmt.Sprintf("Clone\n%s", subtle.Render("Standard git clone to destination.")), "clone"),
						huh.NewOption(fmt.Sprintf("Copy\n%s", subtle.Render("Clones to temp dir, then copies to destination.")), "copy"),
					).
					Value(&method),
				huh.NewSelect[string]().
					Title("Merge Strategy").
					Description("Only applies if files conflict").
					Options(
						huh.NewOption(fmt.Sprintf("Overwrite\n%s", subtle.Render("Hard Reset: Overwrites YOUR files with theirs.")), "overwrite"),
						huh.NewOption(fmt.Sprintf("Keep Existing\n%s", subtle.Render("Safe Merge: Keeps YOUR files, adds missing ones.")), "keep_existing"),
					).
					Value(&strategy),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}

		if name != "" && url != "" && dest != "" {
			ext := ExternalDep{
				Name:          name,
				ID:            slugify(name),
				URL:           url,
				Destination:   dest,
				Method:        method,
				MergeStrategy: strategy,
			}
			externalDeps = append(externalDeps, ext)
		}

		// Ask to add another
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add another external dependency?").
					Value(&addExternal),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}
	}

	// System Dependencies
	var systemDeps []DependencyItem
	var addSystemDep bool

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to add system dependencies (e.g. neovim, tmux)?").
				Value(&addSystemDep),
		),
	).WithInput(in).WithOutput(out).Run()

	if err != nil {
		return err
	}

	for addSystemDep {
		var name, binary, version string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Dependency Name").Placeholder("neovim").Value(&name),
				huh.NewInput().Title("Binary Name").Placeholder("nvim").Value(&binary),
				huh.NewInput().Title("Required Version (optional)").Placeholder("0.11+").Value(&version),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}

		if name != "" {
			if binary == "" {
				binary = name
			}
			systemDeps = append(systemDeps, DependencyItem{
				Name:    name,
				Binary:  binary,
				Version: version,
			})
		}

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title("Add another system dependency?").Value(&addSystemDep),
			),
		).WithInput(in).WithOutput(out).Run()
		if err != nil {
			return err
		}
	}

	// Machine Specific Configs
	var machineConfigs []MachinePrompt
	var addMachineConfig bool

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to add machine-specific configurations (e.g. git signing)?").
				Value(&addMachineConfig),
		),
	).WithInput(in).WithOutput(out).Run()

	if err != nil {
		return err
	}

	for addMachineConfig {
		var choice string
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select a machine config preset or create custom").
					Options(
						huh.NewOption("Git Signing (Name, Email, GPG Key)", "git-signing"),
						huh.NewOption("Custom", "custom"),
					).
					Value(&choice),
			),
		).WithInput(in).WithOutput(out).Run()

		if err != nil {
			return err
		}

		if choice == "git-signing" {
			machineConfigs = append(machineConfigs, MachinePrompt{
				ID:          "git-local",
				Description: "Local Git Configuration",
				Destination: "~/.gitconfig.local",
				Prompts: []PromptField{
					{ID: "name", Prompt: "Git User Name", Type: "text", Required: true},
					{ID: "email", Prompt: "Git User Email", Type: "text", Required: true},
					{ID: "signingkey", Prompt: "GPG Signing Key ID (optional)", Type: "text"},
				},
				Template: "[user]\n  name = {{.name}}\n  email = {{.email}}\n{{if .signingkey}}  signingkey = {{.signingkey}}\n[commit]\n  gpgsign = true{{end}}\n",
			})
		} else {
			var id, desc, dest, tmpl string
			err = huh.NewForm(
				huh.NewGroup(
					huh.NewInput().Title("Config ID").Placeholder("my-config").Value(&id),
					huh.NewInput().Title("Description").Placeholder("My custom config").Value(&desc),
					huh.NewInput().Title("Destination Path").Placeholder("~/.myconfig").Value(&dest),
					huh.NewInput().Title("Template").Placeholder("key = {{.value}}").Value(&tmpl),
				),
			).WithInput(in).WithOutput(out).Run()

			if err != nil {
				return err
			}

			if id != "" {
				machineConfigs = append(machineConfigs, MachinePrompt{
					ID:          id,
					Description: desc,
					Destination: dest,
					Template:    tmpl,
					Prompts: []PromptField{
						{ID: "value", Prompt: "Enter value", Type: "text", Required: true},
					},
				})
			}
		}

		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title("Add another machine config?").Value(&addMachineConfig),
			),
		).WithInput(in).WithOutput(out).Run()
		if err != nil {
			return err
		}
	}

	// Create Config
	cfg := Config{
		SchemaVersion: "1.0",
		Metadata:      meta,
		Dependencies: Dependencies{
			Critical: []DependencyItem{
				{Name: "git", Binary: "git"},
				{Name: "stow", Binary: "stow"},
			},
			Core: systemDeps,
		},
		Configs: ConfigGroups{
			Core: selectedConfigs,
		},
		External:      externalDeps,
		MachineConfig: machineConfigs,
	}

	// Generate YAML
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	// Add comments to the top
	content := fmt.Sprintf("# Generated by go4dot\n# Edit this file to customize your dotfiles management\n\n%s", string(data))

	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Fprintf(out, "\n✅ Successfully created %s\n", configFile)
	fmt.Fprintln(out, "run 'g4d install' to set up your dotfiles!")

	return nil
}

func scanDirectory(root string) ([]ConfigItem, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var items []ConfigItem

	// Directories to always ignore (not dotfiles-related)
	ignored := map[string]bool{
		// Version control
		".git":    true,
		".github": true,
		".gitlab": true,
		".svn":    true,

		// IDE/Editor
		".idea":   true,
		".vscode": true,
		".vim":    false, // This IS a dotfile config
		".nvim":   false, // This IS a dotfile config

		// Build/Output
		"bin":          true,
		"build":        true,
		"dist":         true,
		"node_modules": true,
		"vendor":       true,
		"target":       true,
		"__pycache__":  true,
		".cache":       true,

		// Project files (not dotfiles)
		ConfigFileName: true,
		"README.md":    true,
		"LICENSE":      true,
		"Makefile":     true,
		"go.mod":       true,
		"go.sum":       true,
		"package.json": true,
		"Cargo.toml":   true,

		// go4dot internal
		"test":    true,
		"sandbox": true,
	}

	for _, entry := range entries {
		name := entry.Name()

		// Check explicit ignore list
		if ignored[name] {
			continue
		}

		// Only include directories (dotfiles are usually directories for stow)
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories that start with . unless they look like dotfile configs
		// (e.g., .config is OK, .cache is not)
		if len(name) > 1 && name[0] == '.' {
			// Common hidden dotfile configs to include
			validHiddenDirs := map[string]bool{
				".config":      true,
				".local":       true,
				".vim":         true,
				".nvim":        true,
				".emacs.d":     true,
				".tmux":        true,
				".ssh":         true,
				".gnupg":       true,
				".fonts":       true,
				".themes":      true,
				".icons":       true,
				".mozilla":     true,
				".thunderbird": true,
			}
			if !validHiddenDirs[name] {
				continue
			}
		}

		items = append(items, ConfigItem{
			Name:        name,
			Path:        name,
			Description: fmt.Sprintf("%s configuration", name),
			Platforms:   []string{"linux", "macos"},
		})
	}

	return items, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	// Replace non-alphanumeric chars with hyphens
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")
	// Trim hyphens
	s = strings.Trim(s, "-")
	return s
}
