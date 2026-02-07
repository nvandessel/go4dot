package deps

import (
	"fmt"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

// InstallResult represents the result of installing dependencies
type InstallResult struct {
	Installed     []config.DependencyItem
	Failed        []InstallError
	Skipped       []config.DependencyItem
	ManualSkipped []config.DependencyItem // Manual deps skipped from automated install
}

// InstallError represents a failed installation
type InstallError struct {
	Item  config.DependencyItem
	Error error
}

// InstallOptions configures the installation behavior
type InstallOptions struct {
	SkipPrompts  bool                                 // If true, install without asking
	OnlyMissing  bool                                 // Only install missing deps
	DryRun       bool                                 // Don't actually install, just report
	ProgressFunc func(current, total int, msg string) // Called for progress updates with item counts
}

// Install installs missing dependencies
func Install(cfg *config.Config, p *platform.Platform, opts InstallOptions) (*InstallResult, error) {
	result := &InstallResult{}

	// Check current status
	checkResult, err := Check(cfg, p)
	if err != nil {
		return nil, fmt.Errorf("failed to check dependencies: %w", err)
	}

	// Report manual dependencies that must be installed by the user
	manualMissing := checkResult.GetManualMissing()
	for _, depCheck := range manualMissing {
		dep := depCheck.Item
		result.ManualSkipped = append(result.ManualSkipped, dep)
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(0, 0, fmt.Sprintf("Skipping manual dependency: %s (install manually)", dep.Name))
		}
	}

	// Get missing dependencies (excludes manual deps)
	missing := checkResult.GetMissing()
	if len(missing) == 0 {
		return result, nil // Nothing to do
	}

	// Get package manager
	pkgMgr, err := platform.GetPackageManager(p)
	if err != nil {
		return nil, fmt.Errorf("failed to get package manager: %w", err)
	}

	if !pkgMgr.IsAvailable() {
		return nil, fmt.Errorf("package manager %s is not available", pkgMgr.Name())
	}

	// Update package cache first
	total := len(missing)
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, total, "Updating package cache...")
	}

	if !opts.DryRun {
		if err := pkgMgr.Update(); err != nil {
			// Don't fail on update errors, just warn
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(0, total, fmt.Sprintf("Warning: failed to update package cache: %v", err))
			}
		}
	}

	// Install each missing dependency
	for i, depCheck := range missing {
		dep := depCheck.Item
		current := i + 1

		if opts.ProgressFunc != nil {
			opts.ProgressFunc(current, total, fmt.Sprintf("Installing %s...", dep.Name))
		}

		if opts.DryRun {
			result.Installed = append(result.Installed, dep)
			continue
		}

		// Get package name for this platform
		pkgName := getPackageNameForPlatform(dep, pkgMgr.Name())
		if pkgName == "" {
			pkgName = dep.Name
		}

		// Try to install
		err := pkgMgr.Install(pkgName)
		if err != nil {
			result.Failed = append(result.Failed, InstallError{
				Item:  dep,
				Error: err,
			})
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("Failed to install %s: %v", dep.Name, err))
			}
		} else {
			result.Installed = append(result.Installed, dep)
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("Installed %s", dep.Name))
			}
		}
	}

	return result, nil
}

// getPackageNameForPlatform returns the platform-specific package name
func getPackageNameForPlatform(dep config.DependencyItem, manager string) string {
	if dep.Package != nil {
		if pkgName, ok := dep.Package[manager]; ok {
			return pkgName
		}
	}
	return ""
}

// InstallMissing is a convenience function that installs only missing dependencies
func InstallMissing(cfg *config.Config, p *platform.Platform) (*InstallResult, error) {
	return Install(cfg, p, InstallOptions{
		OnlyMissing: true,
	})
}
