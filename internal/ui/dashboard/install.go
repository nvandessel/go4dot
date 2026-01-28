package dashboard

import (
	"fmt"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
)

// InstallOptions configures the dashboard installation behavior
type InstallOptions struct {
	Auto         bool // Non-interactive, use defaults
	Minimal      bool // Only core configs, skip optional
	SkipDeps     bool // Skip dependency installation
	SkipExternal bool // Skip external dependency cloning
	SkipMachine  bool // Skip machine-specific configuration
	SkipStow     bool // Skip stowing configs
	Overwrite    bool // Overwrite existing files
}

// InstallResult holds the result of an installation
type InstallResult struct {
	Platform       *platform.Platform
	DepsInstalled  []config.DependencyItem
	DepsFailed     []deps.InstallError
	ConfigsStowed  []string
	ConfigsAdopted []string
	ConfigsFailed  []stow.StowError
	ExternalCloned []config.ExternalDep
	ExternalFailed []deps.ExternalError
	MachineConfigs []machine.RenderResult
	Errors         []error
}

// HasErrors returns true if any errors occurred
func (r *InstallResult) HasErrors() bool {
	return len(r.DepsFailed) > 0 || len(r.ConfigsFailed) > 0 ||
		len(r.ExternalFailed) > 0 || len(r.Errors) > 0
}

// Summary returns a summary string of the installation
func (r *InstallResult) Summary() string {
	var summary string
	if r.Platform != nil {
		summary = fmt.Sprintf("Platform: %s", r.Platform.OS)
		if r.Platform.Distro != "" {
			summary += fmt.Sprintf(" (%s)", r.Platform.Distro)
		}
		summary += "\n"
	}

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

// RunInstallOperation runs the install operation within the dashboard
func RunInstallOperation(runner *OperationRunner, cfg *config.Config, dotfilesPath string, opts InstallOptions) (*InstallResult, error) {
	result := &InstallResult{}

	// Step 0: Detect platform
	runner.Progress(0, "Detecting OS and package manager...")
	p, err := platform.Detect()
	if err != nil {
		runner.StepComplete(0, StepError, err.Error())
		return nil, fmt.Errorf("failed to detect platform: %w", err)
	}
	result.Platform = p
	runner.StepComplete(0, StepSuccess, fmt.Sprintf("%s (%s)", p.OS, p.PackageManager))

	// Step 1: Install dependencies
	if !opts.SkipDeps {
		if err := runDependencyInstall(runner, cfg, p, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		runner.StepComplete(1, StepSkipped, "Skipped")
	}

	// Step 2: Stow configs
	if !opts.SkipStow {
		if err := runStowConfigs(runner, cfg, dotfilesPath, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		runner.StepComplete(2, StepSkipped, "Skipped")
	}

	// Step 3: Clone external dependencies
	if !opts.SkipExternal {
		if err := runCloneExternal(runner, cfg, dotfilesPath, p, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		runner.StepComplete(3, StepSkipped, "Skipped")
	}

	// Step 4: Configure machine settings
	if !opts.SkipMachine {
		if err := runMachineConfig(runner, cfg, opts, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	} else {
		runner.StepComplete(4, StepSkipped, "Skipped")
	}

	// Save state
	if err := saveInstallState(cfg, dotfilesPath, result); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to save state: %v", err))
	}

	// Report completion
	if result.HasErrors() {
		runner.Done(false, result.Summary(), fmt.Errorf("installation completed with errors"))
	} else {
		runner.Done(true, result.Summary(), nil)
	}

	return result, nil
}

func runDependencyInstall(runner *OperationRunner, cfg *config.Config, p *platform.Platform, opts InstallOptions, result *InstallResult) error {
	runner.Progress(1, "Checking dependencies...")

	checkResult, err := deps.Check(cfg, p)
	if err != nil {
		runner.StepComplete(1, StepError, err.Error())
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	missing := checkResult.GetMissing()
	if len(missing) == 0 {
		runner.StepComplete(1, StepSuccess, "All dependencies installed")
		return nil
	}

	runner.Progress(1, fmt.Sprintf("Installing %d dependencies...", len(missing)))

	installOpts := deps.InstallOptions{
		OnlyMissing: true,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	installResult, err := deps.Install(cfg, p, installOpts)
	if err != nil {
		runner.StepComplete(1, StepError, err.Error())
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	result.DepsInstalled = installResult.Installed
	result.DepsFailed = installResult.Failed

	if len(installResult.Failed) > 0 {
		runner.StepComplete(1, StepWarning, fmt.Sprintf("%d installed, %d failed", len(installResult.Installed), len(installResult.Failed)))
		for _, f := range installResult.Failed {
			runner.Log("error", fmt.Sprintf("Failed: %s - %v", f.Item.Name, f.Error))
		}
	} else {
		runner.StepComplete(1, StepSuccess, fmt.Sprintf("%d dependencies installed", len(installResult.Installed)))
	}

	return nil
}

func runStowConfigs(runner *OperationRunner, cfg *config.Config, dotfilesPath string, opts InstallOptions, result *InstallResult) error {
	runner.Progress(2, "Checking config status...")

	var configs []config.ConfigItem
	if opts.Minimal {
		configs = cfg.Configs.Core
	} else {
		configs = cfg.GetAllConfigs()
	}

	if len(configs) == 0 {
		runner.StepComplete(2, StepSuccess, "No configs to stow")
		return nil
	}

	// Check for existing symlinks
	adoptSummary, err := stow.ScanExistingSymlinks(cfg, dotfilesPath)
	fullyLinkedMap := make(map[string]bool)
	if err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to scan existing symlinks: %v", err))
	} else if adoptSummary != nil {
		for _, ar := range adoptSummary.GetFullyLinked() {
			fullyLinkedMap[ar.ConfigName] = true
			result.ConfigsAdopted = append(result.ConfigsAdopted, ar.ConfigName)
		}
	}

	if len(result.ConfigsAdopted) > 0 {
		runner.Log("info", fmt.Sprintf("Found %d config(s) already symlinked", len(result.ConfigsAdopted)))
	}

	// Filter out fully-linked configs
	var configsToStow []config.ConfigItem
	for _, c := range configs {
		if !fullyLinkedMap[c.Name] {
			configsToStow = append(configsToStow, c)
		}
	}

	if len(configsToStow) == 0 {
		runner.StepComplete(2, StepSuccess, "All configs already linked")
		return nil
	}

	runner.Progress(2, fmt.Sprintf("Stowing %d configs...", len(configsToStow)))

	stowOpts := stow.StowOptions{
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	stowResult := stow.StowConfigs(dotfilesPath, configsToStow, stowOpts)

	result.ConfigsStowed = stowResult.Success
	result.ConfigsFailed = stowResult.Failed

	if len(stowResult.Failed) > 0 {
		runner.StepComplete(2, StepWarning, fmt.Sprintf("%d stowed, %d failed", len(stowResult.Success), len(stowResult.Failed)))
		for _, f := range stowResult.Failed {
			runner.Log("error", fmt.Sprintf("Failed: %s - %v", f.ConfigName, f.Error))
		}
	} else {
		runner.StepComplete(2, StepSuccess, fmt.Sprintf("%d configs stowed", len(stowResult.Success)))
	}

	return nil
}

func runCloneExternal(runner *OperationRunner, cfg *config.Config, dotfilesPath string, p *platform.Platform, opts InstallOptions, result *InstallResult) error {
	if len(cfg.External) == 0 {
		runner.StepComplete(3, StepSuccess, "No external dependencies")
		return nil
	}

	runner.Progress(3, fmt.Sprintf("Cloning %d external dependencies...", len(cfg.External)))

	extOpts := deps.ExternalOptions{
		RepoRoot: dotfilesPath,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	extResult, err := deps.CloneExternal(cfg, p, extOpts)
	if err != nil {
		runner.StepComplete(3, StepError, err.Error())
		return fmt.Errorf("failed to clone external dependencies: %w", err)
	}

	result.ExternalCloned = extResult.Cloned
	result.ExternalFailed = extResult.Failed

	if len(extResult.Failed) > 0 {
		runner.StepComplete(3, StepWarning, fmt.Sprintf("%d cloned, %d failed", len(extResult.Cloned), len(extResult.Failed)))
		for _, f := range extResult.Failed {
			runner.Log("error", fmt.Sprintf("Failed: %s - %v", f.Dep.Name, f.Error))
		}
	} else if len(extResult.Cloned) > 0 || len(extResult.Skipped) > 0 {
		runner.StepComplete(3, StepSuccess, fmt.Sprintf("%d cloned, %d already present", len(extResult.Cloned), len(extResult.Skipped)))
	} else {
		runner.StepComplete(3, StepSuccess, "All external dependencies already present")
	}

	return nil
}

func runMachineConfig(runner *OperationRunner, cfg *config.Config, opts InstallOptions, result *InstallResult) error {
	if len(cfg.MachineConfig) == 0 {
		runner.StepComplete(4, StepSuccess, "No machine configs")
		return nil
	}

	runner.Progress(4, "Checking machine configuration...")

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
		runner.StepComplete(4, StepSuccess, "All machine configs present")
		return nil
	}

	runner.Progress(4, fmt.Sprintf("Configuring %d machine settings...", len(needsConfig)))

	promptOpts := machine.PromptOptions{
		SkipPrompts: opts.Auto,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	renderOpts := machine.RenderOptions{
		Overwrite: opts.Overwrite,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	for _, mc := range needsConfig {
		promptResult, err := machine.CollectSingleConfig(cfg, mc.ID, promptOpts)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to collect %s: %w", mc.ID, err))
			runner.Log("error", fmt.Sprintf("Failed to collect %s: %v", mc.ID, err))
			continue
		}

		renderResult, err := machine.RenderAndWrite(&mc, promptResult.Values, renderOpts)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to write %s: %w", mc.ID, err))
			runner.Log("error", fmt.Sprintf("Failed to write %s: %v", mc.ID, err))
			continue
		}

		result.MachineConfigs = append(result.MachineConfigs, *renderResult)
	}

	if len(result.MachineConfigs) > 0 {
		runner.StepComplete(4, StepSuccess, fmt.Sprintf("%d machine settings configured", len(result.MachineConfigs)))
	} else if len(needsConfig) > 0 {
		runner.StepComplete(4, StepWarning, "Some machine configs failed")
	}

	return nil
}

func saveInstallState(cfg *config.Config, dotfilesPath string, result *InstallResult) error {
	st, err := state.Load()
	if err != nil || st == nil {
		st = state.New()
	}
	st.DotfilesPath = dotfilesPath

	if result.Platform != nil {
		st.Platform = state.PlatformState{
			OS:             result.Platform.OS,
			Distro:         result.Platform.Distro,
			DistroVersion:  result.Platform.DistroVersion,
			PackageManager: result.Platform.PackageManager,
		}
	}

	allConfigs := append(result.ConfigsStowed, result.ConfigsAdopted...)
	for _, configName := range allConfigs {
		item := cfg.GetConfigByName(configName)
		isCore := false
		if item != nil {
			for _, c := range cfg.Configs.Core {
				if c.Name == configName {
					isCore = true
					break
				}
			}
		}
		st.AddConfig(configName, configName, isCore)
	}

	for _, ext := range result.ExternalCloned {
		st.SetExternalDep(ext.ID, ext.Destination, true)
	}

	for _, mc := range result.MachineConfigs {
		st.SetMachineConfig(mc.ID, mc.Destination, false, false)
	}

	if err := stow.UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
		return fmt.Errorf("failed to update symlink counts: %w", err)
	}

	if err := st.Save(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}
