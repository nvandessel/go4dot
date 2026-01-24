package setup

import (
	"fmt"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
)

// InstallOptions configures the installation behavior
type InstallOptions struct {
	Auto         bool                                 // Non-interactive, use defaults
	Minimal      bool                                 // Only core configs, skip optional
	SkipDeps     bool                                 // Skip dependency installation
	SkipExternal bool                                 // Skip external dependency cloning
	SkipMachine  bool                                 // Skip machine-specific configuration
	SkipStow     bool                                 // Skip stowing configs
	Overwrite    bool                                 // Overwrite existing files
	ProgressFunc func(current, total int, msg string) // Called for progress updates with item counts
}

// InstallResult tracks the result of the installation
type InstallResult struct {
	Platform       *platform.Platform
	DepsInstalled  []config.DependencyItem
	DepsFailed     []deps.InstallError
	ConfigsStowed  []string
	ConfigsAdopted []string // Configs that were already linked and adopted
	ConfigsFailed  []stow.StowError
	ExternalCloned []config.ExternalDep
	ExternalFailed []deps.ExternalError
	MachineConfigs []machine.RenderResult
	Errors         []error
}

// HasErrors returns true if any errors occurred during installation
func (r *InstallResult) HasErrors() bool {
	return len(r.DepsFailed) > 0 || len(r.ConfigsFailed) > 0 ||
		len(r.ExternalFailed) > 0 || len(r.Errors) > 0
}

// Install runs the full installation flow
func Install(cfg *config.Config, dotfilesPath string, opts InstallOptions) (*InstallResult, error) {
	result := &InstallResult{}

	// Step 1: Detect platform
	progress(opts, "Detecting platform...")
	p, err := platform.Detect()
	if err != nil {
		return nil, fmt.Errorf("failed to detect platform: %w", err)
	}
	result.Platform = p
	progress(opts, fmt.Sprintf("✓ Platform: %s (%s)", p.OS, p.PackageManager))

	// Step 2: Check and install dependencies
	if !opts.SkipDeps {
		if err := installDependencies(cfg, p, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
			// Don't return - continue with other steps
		}
	} else {
		progress(opts, "⊘ Skipping dependency installation")
	}

	// Step 3: Stow configs
	if !opts.SkipStow {
		if err := stowConfigs(cfg, dotfilesPath, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		progress(opts, "⊘ Skipping config stowing")
	}

	// Step 4: Clone external dependencies
	if !opts.SkipExternal {
		if err := cloneExternal(cfg, dotfilesPath, p, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		progress(opts, "⊘ Skipping external dependencies")
	}

	// Step 5: Configure machine-specific settings
	if !opts.SkipMachine {
		if err := configureMachine(cfg, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		progress(opts, "⊘ Skipping machine configuration")
	}

	return result, nil
}

// installDependencies checks and installs missing dependencies
func installDependencies(cfg *config.Config, p *platform.Platform, opts InstallOptions, result *InstallResult) error {
	progress(opts, "\n── Dependencies ──")

	// Check current status
	checkResult, err := deps.Check(cfg, p)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	missing := checkResult.GetMissing()
	if len(missing) == 0 {
		progress(opts, "✓ All dependencies are installed")
		return nil
	}

	progress(opts, fmt.Sprintf("Installing %d missing dependencies...", len(missing)))

	installOpts := deps.InstallOptions{
		OnlyMissing: true,
		ProgressFunc: func(current, total int, msg string) {
			progressWithCount(opts, current, total, "  "+msg)
		},
	}

	installResult, err := deps.Install(cfg, p, installOpts)
	if err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	result.DepsInstalled = installResult.Installed
	result.DepsFailed = installResult.Failed

	if len(installResult.Failed) > 0 {
		progress(opts, fmt.Sprintf("⚠ %d dependencies failed to install", len(installResult.Failed)))
	} else {
		progress(opts, fmt.Sprintf("✓ Installed %d dependencies", len(installResult.Installed)))
	}

	return nil
}

// stowConfigs stows all or selected configs, adopting existing symlinks where possible
func stowConfigs(cfg *config.Config, dotfilesPath string, opts InstallOptions, result *InstallResult) error {
	progress(opts, "\n── Configs ──")

	// Get configs to stow
	var configs []config.ConfigItem
	if opts.Minimal {
		configs = cfg.Configs.Core
	} else {
		configs = cfg.GetAllConfigs()
	}

	if len(configs) == 0 {
		progress(opts, "No configs to stow")
		return nil
	}

	// Check for existing symlinks first
	adoptSummary, _ := stow.ScanExistingSymlinks(cfg, dotfilesPath)

	// Build a map of fully-linked configs (can be adopted without re-stowing)
	fullyLinkedMap := make(map[string]bool)
	if adoptSummary != nil {
		for _, ar := range adoptSummary.GetFullyLinked() {
			fullyLinkedMap[ar.ConfigName] = true
			result.ConfigsAdopted = append(result.ConfigsAdopted, ar.ConfigName)
		}
	}

	if len(result.ConfigsAdopted) > 0 {
		progress(opts, fmt.Sprintf("✓ Found %d config(s) already symlinked", len(result.ConfigsAdopted)))
	}

	// Filter out fully-linked configs from those to stow
	var configsToStow []config.ConfigItem
	for _, c := range configs {
		if !fullyLinkedMap[c.Name] {
			configsToStow = append(configsToStow, c)
		}
	}

	if len(configsToStow) == 0 {
		progress(opts, "All configs are already linked")
		return nil
	}

	progress(opts, fmt.Sprintf("Stowing %d configs...", len(configsToStow)))

	stowOpts := stow.StowOptions{
		ProgressFunc: func(current, total int, msg string) {
			progressWithCount(opts, current, total, "  "+msg)
		},
	}

	stowResult := stow.StowConfigs(dotfilesPath, configsToStow, stowOpts)

	result.ConfigsStowed = stowResult.Success
	result.ConfigsFailed = stowResult.Failed

	if len(stowResult.Failed) > 0 {
		progress(opts, fmt.Sprintf("⚠ %d configs failed to stow", len(stowResult.Failed)))
	}
	if len(stowResult.Success) > 0 {
		progress(opts, fmt.Sprintf("✓ Stowed %d configs", len(stowResult.Success)))
	}
	if len(stowResult.Skipped) > 0 {
		progress(opts, fmt.Sprintf("⊘ Skipped %d configs (not found)", len(stowResult.Skipped)))
	}

	return nil
}

// cloneExternal clones external dependencies
func cloneExternal(cfg *config.Config, dotfilesPath string, p *platform.Platform, opts InstallOptions, result *InstallResult) error {
	if len(cfg.External) == 0 {
		return nil
	}

	progress(opts, "\n── External Dependencies ──")
	progress(opts, fmt.Sprintf("Cloning %d external dependencies...", len(cfg.External)))

	extOpts := deps.ExternalOptions{
		RepoRoot: dotfilesPath,
		ProgressFunc: func(current, total int, msg string) {
			progressWithCount(opts, current, total, "  "+msg)
		},
	}

	extResult, err := deps.CloneExternal(cfg, p, extOpts)
	if err != nil {
		return fmt.Errorf("failed to clone external dependencies: %w", err)
	}

	result.ExternalCloned = extResult.Cloned
	result.ExternalFailed = extResult.Failed

	if len(extResult.Failed) > 0 {
		progress(opts, fmt.Sprintf("⚠ %d external deps failed", len(extResult.Failed)))
	}
	if len(extResult.Cloned) > 0 {
		progress(opts, fmt.Sprintf("✓ Cloned %d external deps", len(extResult.Cloned)))
	}
	if len(extResult.Skipped) > 0 {
		progress(opts, fmt.Sprintf("⊘ Skipped %d external deps", len(extResult.Skipped)))
	}

	return nil
}

// configureMachine configures machine-specific settings
func configureMachine(cfg *config.Config, opts InstallOptions, result *InstallResult) error {
	if len(cfg.MachineConfig) == 0 {
		return nil
	}

	progress(opts, "\n── Machine Configuration ──")

	// Check which configs are missing
	statuses := machine.CheckMachineConfigStatus(cfg)
	var needsConfig []config.MachinePrompt

	for _, status := range statuses {
		if status.Status == "missing" {
			mc := machine.GetMachineConfigByID(cfg, status.ID)
			if mc != nil {
				needsConfig = append(needsConfig, *mc)
			}
		}
	}

	if len(needsConfig) == 0 {
		progress(opts, "✓ All machine configs are already set up")
		return nil
	}

	progress(opts, fmt.Sprintf("Configuring %d machine settings...", len(needsConfig)))

	promptOpts := machine.PromptOptions{
		SkipPrompts: opts.Auto,
		ProgressFunc: func(current, total int, msg string) {
			progressWithCount(opts, current, total, "  "+msg)
		},
	}

	renderOpts := machine.RenderOptions{
		Overwrite: opts.Overwrite,
		ProgressFunc: func(current, total int, msg string) {
			progressWithCount(opts, current, total, "  "+msg)
		},
	}

	// Collect and render each config
	for _, mc := range needsConfig {
		promptResult, err := machine.CollectSingleConfig(cfg, mc.ID, promptOpts)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to collect %s: %w", mc.ID, err))
			continue
		}

		renderResult, err := machine.RenderAndWrite(&mc, promptResult.Values, renderOpts)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to write %s: %w", mc.ID, err))
			continue
		}

		result.MachineConfigs = append(result.MachineConfigs, *renderResult)
	}

	if len(result.MachineConfigs) > 0 {
		progress(opts, fmt.Sprintf("✓ Configured %d machine settings", len(result.MachineConfigs)))
	}

	return nil
}

// progress sends a progress message if the callback is set
func progress(opts InstallOptions, msg string) {
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, 0, msg)
	}
}

// progressWithCount sends a progress message with item counts
func progressWithCount(opts InstallOptions, current, total int, msg string) {
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, msg)
	}
}

// Summary returns a human-readable summary of the installation result
func (r *InstallResult) Summary() string {
	var summary string

	summary += fmt.Sprintf("Platform: %s", r.Platform.OS)
	if r.Platform.Distro != "" {
		summary += fmt.Sprintf(" (%s)", r.Platform.Distro)
	}
	summary += "\n"

	if len(r.DepsInstalled) > 0 || len(r.DepsFailed) > 0 {
		summary += fmt.Sprintf("Dependencies: %d installed, %d failed\n",
			len(r.DepsInstalled), len(r.DepsFailed))
	}

	if len(r.ConfigsStowed) > 0 || len(r.ConfigsAdopted) > 0 || len(r.ConfigsFailed) > 0 {
		if len(r.ConfigsAdopted) > 0 {
			summary += fmt.Sprintf("Configs: %d stowed, %d adopted, %d failed\n",
				len(r.ConfigsStowed), len(r.ConfigsAdopted), len(r.ConfigsFailed))
		} else {
			summary += fmt.Sprintf("Configs: %d stowed, %d failed\n",
				len(r.ConfigsStowed), len(r.ConfigsFailed))
		}
	}

	if len(r.ExternalCloned) > 0 || len(r.ExternalFailed) > 0 {
		summary += fmt.Sprintf("External: %d cloned, %d failed\n",
			len(r.ExternalCloned), len(r.ExternalFailed))
	}

	if len(r.MachineConfigs) > 0 {
		summary += fmt.Sprintf("Machine configs: %d configured\n", len(r.MachineConfigs))
	}

	return summary
}

// SaveState saves the installation state to the standard location.
func SaveState(cfg *config.Config, dotfilesPath string, result *InstallResult) error {
	st, err := state.Load()
	if err != nil || st == nil {
		st = state.New()
	}
	st.DotfilesPath = dotfilesPath

	// Save platform info
	if result.Platform != nil {
		st.Platform = state.PlatformState{
			OS:             result.Platform.OS,
			Distro:         result.Platform.Distro,
			DistroVersion:  result.Platform.DistroVersion,
			PackageManager: result.Platform.PackageManager,
		}
	}

	// Save installed configs (both stowed and adopted)
	allConfigs := append(result.ConfigsStowed, result.ConfigsAdopted...)
	for _, configName := range allConfigs {
		item := cfg.GetConfigByName(configName)
		isCore := false
		if item != nil {
			// Check if it's a core config
			for _, c := range cfg.Configs.Core {
				if c.Name == configName {
					isCore = true
					break
				}
			}
		}
		st.AddConfig(configName, configName, isCore)
	}

	// Save external deps
	for _, ext := range result.ExternalCloned {
		st.SetExternalDep(ext.ID, ext.Destination, true)
	}

	// Save machine configs
	for _, mc := range result.MachineConfigs {
		st.SetMachineConfig(mc.ID, mc.Destination, false, false)
	}

	// Update symlink counts so dashboard shows correct sync status
	if err := stow.UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
		return fmt.Errorf("failed to update symlink counts: %w", err)
	}

	// Save state
	if err := st.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}
