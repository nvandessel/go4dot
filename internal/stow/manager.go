package stow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
)

// StowResult represents the result of a stow operation across multiple configurations.
type StowResult struct {
	Success []string    // List of successfully stowed config names
	Failed  []StowError // List of configs that failed to stow with their errors
	Skipped []string    // List of configs that were skipped (e.g., directory not found)
}

// StowError represents an error that occurred during a stow operation for a specific config.
type StowError struct {
	ConfigName string // Name of the configuration that failed
	Error      error  // The error that occurred
}

// StowOptions configures behavior for stow operations.
type StowOptions struct {
	DryRun       bool                                 // If true, don't make any changes, just show what would happen
	Force        bool                                 // If true, use --adopt to take over existing files
	ProgressFunc func(current, total int, msg string) // Callback for progress updates
}

// Stow symlinks a config directory using GNU stow.
// It uses default settings and processes the specified config package.
func Stow(dotfilesPath string, configName string, opts StowOptions) error {
	return StowWithCount(dotfilesPath, configName, 1, 1, opts)
}

// StowWithCount symlinks a config directory using GNU stow with progress tracking.
// It allows specifying the current and total item counts for progress reporting.
func StowWithCount(dotfilesPath string, configName string, current, total int, opts StowOptions) error {
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("Stowing %s...", configName))
	}

	// Build stow command
	args := []string{"-v"} // Verbose

	if opts.DryRun {
		args = append(args, "-n") // No-op/dry-run
	}

	if opts.Force {
		args = append(args, "--adopt") // Adopt existing files
	}

	args = append(args, "-t", os.Getenv("HOME")) // Target home directory
	args = append(args, "-d", dotfilesPath)      // Directory containing packages
	args = append(args, configName)              // Package to stow

	cmd := exec.Command("stow", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("stow failed: %w\nOutput: %s", err, string(output))
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("✓ Stowed %s", configName))
	}

	return nil
}

// Unstow removes symlinks for a config using GNU stow.
// It effectively reverses a previous stow operation for the specified package.
func Unstow(dotfilesPath string, configName string, opts StowOptions) error {
	return UnstowWithCount(dotfilesPath, configName, 1, 1, opts)
}

// UnstowWithCount removes symlinks for a config with progress tracking.
// It uses the -D flag of GNU stow to remove the symlinks created for a package.
func UnstowWithCount(dotfilesPath string, configName string, current, total int, opts StowOptions) error {
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("Unstowing %s...", configName))
	}

	args := []string{"-v", "-D"} // Delete/unstow

	if opts.DryRun {
		args = append(args, "-n")
	}

	args = append(args, "-t", os.Getenv("HOME"))
	args = append(args, "-d", dotfilesPath)
	args = append(args, configName)

	cmd := exec.Command("stow", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("unstow failed: %w\nOutput: %s", err, string(output))
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("✓ Unstowed %s", configName))
	}

	return nil
}

// Restow refreshes symlinks for a config (unstow + stow).
// It's useful when files have been added to or removed from the source config directory.
func Restow(dotfilesPath string, configName string, opts StowOptions) error {
	return RestowWithCount(dotfilesPath, configName, 1, 1, opts)
}

// RestowWithCount refreshes symlinks for a config with progress tracking.
// It uses the -R flag of GNU stow to rebuild the symlink tree.
func RestowWithCount(dotfilesPath string, configName string, current, total int, opts StowOptions) error {
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("Restowing %s...", configName))
	}

	args := []string{"-v", "-R"} // Restow

	if opts.DryRun {
		args = append(args, "-n")
	}

	if opts.Force {
		args = append(args, "--adopt")
	}

	args = append(args, "-t", os.Getenv("HOME"))
	args = append(args, "-d", dotfilesPath)
	args = append(args, configName)

	cmd := exec.Command("stow", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("restow failed: %w\nOutput: %s", err, string(output))
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("✓ Restowed %s", configName))
	}

	return nil
}

// StowConfigs stows multiple configurations in sequence.
// It returns a comprehensive result object detailing successes, failures, and skips.
func StowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult {
	result := &StowResult{}
	total := len(configs)

	for i, cfg := range configs {
		current := i + 1

		// Check if config directory exists
		configPath := filepath.Join(dotfilesPath, cfg.Path)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			result.Skipped = append(result.Skipped, cfg.Name)
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("⊘ Skipped %s (directory not found)", cfg.Name))
			}
			continue
		}

		// Stow it
		err := StowWithCount(dotfilesPath, cfg.Path, current, total, opts)
		if err != nil {
			result.Failed = append(result.Failed, StowError{
				ConfigName: cfg.Name,
				Error:      err,
			})
		} else {
			result.Success = append(result.Success, cfg.Name)
		}
	}

	return result
}

// UnstowConfigs unstows multiple configurations in sequence.
// It uses GNU stow -D for each configuration.
func UnstowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult {
	result := &StowResult{}
	total := len(configs)

	for i, cfg := range configs {
		current := i + 1
		err := UnstowWithCount(dotfilesPath, cfg.Path, current, total, opts)
		if err != nil {
			result.Failed = append(result.Failed, StowError{
				ConfigName: cfg.Name,
				Error:      err,
			})
		} else {
			result.Success = append(result.Success, cfg.Name)
		}
	}

	return result
}

// RestowConfigs restows multiple configurations in sequence.
// It uses GNU stow -R for each configuration.
func RestowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult {
	result := &StowResult{}
	total := len(configs)

	for i, cfg := range configs {
		current := i + 1
		configPath := filepath.Join(dotfilesPath, cfg.Path)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			result.Skipped = append(result.Skipped, cfg.Name)
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("⊘ Skipped %s (directory not found)", cfg.Name))
			}
			continue
		}

		err := RestowWithCount(dotfilesPath, cfg.Path, current, total, opts)
		if err != nil {
			result.Failed = append(result.Failed, StowError{
				ConfigName: cfg.Name,
				Error:      err,
			})
		} else {
			result.Success = append(result.Success, cfg.Name)
		}
	}

	return result
}

// IsStowInstalled checks if the GNU stow executable is available in the system PATH.
func IsStowInstalled() bool {
	_, err := exec.LookPath("stow")
	return err == nil
}

// ValidateStow checks if GNU stow is installed and correctly reports its version.
// It ensures that the 'stow' command is available and identifies as GNU Stow.
func ValidateStow() error {
	if !IsStowInstalled() {
		return fmt.Errorf("GNU stow is not installed")
	}

	// Try to get stow version
	cmd := exec.Command("stow", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("stow command failed: %w", err)
	}

	// Check if it's actually GNU stow
	if !strings.Contains(string(output), "GNU Stow") && !strings.Contains(string(output), "stow") {
		return fmt.Errorf("unexpected stow version output: %s", string(output))
	}

	return nil
}
