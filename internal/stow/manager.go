package stow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/validation"
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

// Commander defines the interface for executing stow commands.
type Commander interface {
	Run(name string, args ...string) ([]byte, error)
}

// ExecCommander is the default implementation that uses os/exec.
type ExecCommander struct{}

// Run executes a command using os/exec.
func (e *ExecCommander) Run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// MockCommander simulates GNU Stow behavior for testing.
type MockCommander struct {
	LastArgs []string
}

// Run parses stow-like arguments and manipulates the filesystem to simulate stow.
func (m *MockCommander) Run(name string, args ...string) ([]byte, error) {
	m.LastArgs = args

	if name != "stow" {
		return nil, fmt.Errorf("unexpected command: %s", name)
	}

	// Handle --version
	for _, arg := range args {
		if arg == "--version" {
			return []byte("stow (GNU Stow) version 2.3.1"), nil
		}
	}

	// Parse arguments
	var targetDir, dotfilesDir string
	var deleteMode, restowMode, dryRun bool
	var packages []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-t":
			i++
			if i < len(args) {
				targetDir = args[i]
			}
		case "-d":
			i++
			if i < len(args) {
				dotfilesDir = args[i]
			}
		case "-D":
			deleteMode = true
		case "-R":
			restowMode = true
		case "-n":
			dryRun = true
		case "-v":
			// ignore
		case "--":
			// Everything after -- is a package name, not a flag
			packages = append(packages, args[i+1:]...)
			// Skip to end since we've consumed all remaining args
			i = len(args) - 1
		default:
			if !strings.HasPrefix(arg, "-") {
				packages = append(packages, arg)
			}
		}
	}

	if targetDir == "" {
		targetDir = os.Getenv("HOME")
	}

	if dryRun {
		return []byte("Dry run: no changes made"), nil
	}

	for _, pkg := range packages {
		pkgPath := filepath.Join(dotfilesDir, pkg)

		if deleteMode || restowMode {
			// Simulate Unstow
			_ = filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(pkgPath, path)
				targetPath := filepath.Join(targetDir, rel)

				// Check if it's a symlink pointing to our source
				if linkInfo, err := os.Lstat(targetPath); err == nil && linkInfo.Mode()&os.ModeSymlink != 0 {
					if dest, err := os.Readlink(targetPath); err == nil {
						absDest := dest
						if !filepath.IsAbs(dest) {
							absDest = filepath.Clean(filepath.Join(filepath.Dir(targetPath), dest))
						}
						if absDest == filepath.Clean(path) {
							_ = os.Remove(targetPath)
						}
					}
				}
				return nil
			})
		}

		if !deleteMode {
			// Simulate Stow
			_ = filepath.Walk(pkgPath, func(path string, info os.FileInfo, err error) error {
				if err != nil || info.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(pkgPath, path)
				targetPath := filepath.Join(targetDir, rel)

				// Ensure parent directory exists
				_ = os.MkdirAll(filepath.Dir(targetPath), 0755)

				// Create symlink (relative path preferred by stow)
				relSrc, _ := filepath.Rel(filepath.Dir(targetPath), path)
				_ = os.Symlink(relSrc, targetPath)
				return nil
			})
		}
	}

	return []byte("Mock stow finished successfully"), nil
}

var (
	// CurrentCommander is the commander instance used for all stow operations.
	// It can be replaced in tests with a mock implementation.
	CurrentCommander Commander = &ExecCommander{}
)

// Stow symlinks a config directory using GNU stow.
// It uses default settings and processes the specified config package.
func Stow(dotfilesPath string, configName string, opts StowOptions) error {
	return StowWithCount(dotfilesPath, configName, 1, 1, opts)
}

// StowWithCount symlinks a config directory using GNU stow with progress tracking.
// It allows specifying the current and total item counts for progress reporting.
func StowWithCount(dotfilesPath string, configName string, current, total int, opts StowOptions) error {
	if err := validation.ValidateConfigName(configName); err != nil {
		return fmt.Errorf("invalid config name: %w", err)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("Stowing %s...", configName))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Build stow command
	args := []string{"-v"} // Verbose

	if opts.DryRun {
		args = append(args, "-n") // No-op/dry-run
	}

	if opts.Force {
		args = append(args, "--adopt") // Adopt existing files
	}

	args = append(args, "-t", homeDir)         // Target home directory
	args = append(args, "-d", dotfilesPath)    // Directory containing packages
	args = append(args, "--", configName)      // Package to stow (-- prevents flag injection)

	output, err := CurrentCommander.Run("stow", args...)

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
	if err := validation.ValidateConfigName(configName); err != nil {
		return fmt.Errorf("invalid config name: %w", err)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("Unstowing %s...", configName))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	args := []string{"-v", "-D"} // Delete/unstow

	if opts.DryRun {
		args = append(args, "-n")
	}

	args = append(args, "-t", homeDir)
	args = append(args, "-d", dotfilesPath)
	args = append(args, "--", configName)

	output, err := CurrentCommander.Run("stow", args...)

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
	if err := validation.ValidateConfigName(configName); err != nil {
		return fmt.Errorf("invalid config name: %w", err)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(current, total, fmt.Sprintf("Restowing %s...", configName))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	args := []string{"-v", "-R"} // Restow

	if opts.DryRun {
		args = append(args, "-n")
	}

	if opts.Force {
		args = append(args, "--adopt")
	}

	args = append(args, "-t", homeDir)
	args = append(args, "-d", dotfilesPath)
	args = append(args, "--", configName)

	output, err := CurrentCommander.Run("stow", args...)

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

		// Check if config directory exists
		configPath := filepath.Join(dotfilesPath, cfg.Path)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			result.Skipped = append(result.Skipped, cfg.Name)
			if opts.ProgressFunc != nil {
				opts.ProgressFunc(current, total, fmt.Sprintf("⊘ Skipped %s (directory not found)", cfg.Name))
			}
			continue
		}

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
	output, err := CurrentCommander.Run("stow", "--version")
	if err != nil {
		return fmt.Errorf("stow command failed: %w", err)
	}

	// Check if it's actually GNU stow
	if !strings.Contains(string(output), "stow (GNU Stow)") && !strings.Contains(string(output), "GNU Stow") {
		return fmt.Errorf("unexpected stow version output: %s", string(output))
	}

	return nil
}
