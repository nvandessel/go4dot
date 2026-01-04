package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
)

// CheckStatus represents the status of a health check
type CheckStatus string

const (
	StatusOK      CheckStatus = "ok"
	StatusWarning CheckStatus = "warning"
	StatusError   CheckStatus = "error"
	StatusSkipped CheckStatus = "skipped"
)

// Check represents a single health check result
type Check struct {
	Name        string
	Description string
	Status      CheckStatus
	Message     string
	Fix         string // Suggested fix command or action
}

// CheckResult contains all health check results
type CheckResult struct {
	Platform       *platform.Platform
	Checks         []Check
	DepsResult     *deps.CheckResult
	ExternalStatus []deps.ExternalStatus
	MachineStatus  []machine.MachineConfigStatus
	SymlinkStatus  []SymlinkCheck
}

// SymlinkCheck represents the status of a stowed symlink
type SymlinkCheck struct {
	Config     string
	TargetPath string
	Status     CheckStatus
	Message    string
}

// CheckOptions configures the health check behavior
type CheckOptions struct {
	DotfilesPath string
	ProgressFunc func(msg string)
}

// RunChecks performs all health checks and returns results
func RunChecks(cfg *config.Config, opts CheckOptions) (*CheckResult, error) {
	result := &CheckResult{}

	// Step 1: Detect platform
	progress(opts, "Checking platform...")
	p, err := platform.Detect()
	if err != nil {
		return nil, fmt.Errorf("failed to detect platform: %w", err)
	}
	result.Platform = p
	result.Checks = append(result.Checks, Check{
		Name:        "Platform Detection",
		Description: "Detect OS and package manager",
		Status:      StatusOK,
		Message:     fmt.Sprintf("%s (%s)", p.OS, p.PackageManager),
	})

	// Step 2: Check stow is installed
	progress(opts, "Checking GNU stow...")
	stowCheck := checkStow()
	result.Checks = append(result.Checks, stowCheck)

	// Step 3: Check git is installed
	progress(opts, "Checking git...")
	gitCheck := checkGit()
	result.Checks = append(result.Checks, gitCheck)

	// Step 4: Check dependencies
	progress(opts, "Checking dependencies...")
	depsResult, err := deps.Check(cfg, p)
	if err != nil {
		result.Checks = append(result.Checks, Check{
			Name:        "Dependencies",
			Description: "Check required packages",
			Status:      StatusError,
			Message:     err.Error(),
		})
	} else {
		result.DepsResult = depsResult
		depCheck := summarizeDepsCheck(depsResult)
		result.Checks = append(result.Checks, depCheck)
	}

	// Step 5: Check symlinks
	progress(opts, "Checking symlinks...")
	if opts.DotfilesPath != "" && !stowCheck.Status.isError() {
		symlinkStatus := checkSymlinks(cfg, opts.DotfilesPath)
		result.SymlinkStatus = symlinkStatus
		symlinkCheck := summarizeSymlinkCheck(symlinkStatus)
		result.Checks = append(result.Checks, symlinkCheck)
	} else {
		result.Checks = append(result.Checks, Check{
			Name:        "Symlinks",
			Description: "Check stowed config symlinks",
			Status:      StatusSkipped,
			Message:     "Dotfiles path not provided or stow not available",
		})
	}

	// Step 6: Check external dependencies
	progress(opts, "Checking external dependencies...")
	if len(cfg.External) > 0 {
		extStatus := deps.CheckExternalStatus(cfg, p, opts.DotfilesPath)
		result.ExternalStatus = extStatus
		extCheck := summarizeExternalCheck(extStatus)
		result.Checks = append(result.Checks, extCheck)
	}

	// Step 7: Check machine configs
	progress(opts, "Checking machine configurations...")
	if len(cfg.MachineConfig) > 0 {
		machineStatus := machine.CheckMachineConfigStatus(cfg)
		result.MachineStatus = machineStatus
		machineCheck := summarizeMachineCheck(machineStatus)
		result.Checks = append(result.Checks, machineCheck)
	}

	return result, nil
}

// checkStow verifies GNU stow is installed
func checkStow() Check {
	check := Check{
		Name:        "GNU Stow",
		Description: "Symlink farm manager",
	}

	if !stow.IsStowInstalled() {
		check.Status = StatusError
		check.Message = "GNU stow is not installed"
		check.Fix = "Install with your package manager (e.g., dnf install stow, apt install stow, brew install stow)"
		return check
	}

	if err := stow.ValidateStow(); err != nil {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("stow validation failed: %v", err)
		return check
	}

	check.Status = StatusOK
	check.Message = "Installed and working"
	return check
}

// checkGit verifies git is installed
func checkGit() Check {
	check := Check{
		Name:        "Git",
		Description: "Version control system",
	}

	path, err := exec.LookPath("git")
	if err != nil {
		check.Status = StatusError
		check.Message = "git is not installed"
		check.Fix = "Install with your package manager"
		return check
	}

	check.Status = StatusOK
	check.Message = fmt.Sprintf("Found at %s", path)
	return check
}

// summarizeDepsCheck creates a check summary from deps check result
func summarizeDepsCheck(result *deps.CheckResult) Check {
	check := Check{
		Name:        "Dependencies",
		Description: "Required packages",
	}

	missing := result.GetMissing()
	missingCritical := result.GetMissingCritical()

	if len(missingCritical) > 0 {
		check.Status = StatusError
		check.Message = fmt.Sprintf("%d critical dependencies missing", len(missingCritical))
		check.Fix = "Run 'g4d deps install' to install missing dependencies"
		return check
	}

	if len(missing) > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("%d optional dependencies missing", len(missing))
		check.Fix = "Run 'g4d deps install' to install missing dependencies"
		return check
	}

	check.Status = StatusOK
	check.Message = result.Summary()
	return check
}

// checkSymlinks verifies all stowed symlinks are valid
func checkSymlinks(cfg *config.Config, dotfilesPath string) []SymlinkCheck {
	var checks []SymlinkCheck
	home := os.Getenv("HOME")

	allConfigs := cfg.GetAllConfigs()
	for _, configItem := range allConfigs {
		configPath := filepath.Join(dotfilesPath, configItem.Path)

		// Check if config directory exists in dotfiles
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			checks = append(checks, SymlinkCheck{
				Config:  configItem.Name,
				Status:  StatusSkipped,
				Message: "Config directory not found in dotfiles",
			})
			continue
		}

		// Walk the config directory and check each file's symlink
		err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip on error
			}
			if info.IsDir() {
				return nil // Skip directories
			}

			// Calculate expected target path
			relPath, _ := filepath.Rel(configPath, path)
			targetPath := filepath.Join(home, relPath)

			check := SymlinkCheck{
				Config:     configItem.Name,
				TargetPath: targetPath,
			}

			// Check if target exists
			targetInfo, err := os.Lstat(targetPath)
			if os.IsNotExist(err) {
				check.Status = StatusWarning
				check.Message = "Symlink missing"
				checks = append(checks, check)
				return nil
			}

			// Check if it's a symlink
			if targetInfo.Mode()&os.ModeSymlink == 0 {
				// If not a symlink, check if it's the same file (handles directory folding)
				sourceInfo, err := os.Stat(path)
				if err == nil && os.SameFile(sourceInfo, targetInfo) {
					// It's the same file (synced via parent directory symlink) - OK
					check.Status = StatusOK
					check.Message = "Valid (via directory fold)"
					checks = append(checks, check)
					return nil
				}

				check.Status = StatusWarning
				check.Message = "Not a symlink (conflict)"
				checks = append(checks, check)
				return nil
			}

			// Check if symlink points to correct location
			linkDest, err := os.Readlink(targetPath)
			if err != nil {
				check.Status = StatusError
				check.Message = fmt.Sprintf("Cannot read symlink: %v", err)
				checks = append(checks, check)
				return nil
			}

			// Resolve to absolute path
			if !filepath.IsAbs(linkDest) {
				linkDest = filepath.Join(filepath.Dir(targetPath), linkDest)
			}
			linkDest = filepath.Clean(linkDest)

			if linkDest != path {
				check.Status = StatusWarning
				check.Message = fmt.Sprintf("Points to wrong location: %s", linkDest)
				checks = append(checks, check)
				return nil
			}

			check.Status = StatusOK
			check.Message = "Valid symlink"
			checks = append(checks, check)
			return nil
		})

		if err != nil {
			checks = append(checks, SymlinkCheck{
				Config:  configItem.Name,
				Status:  StatusError,
				Message: fmt.Sprintf("Error checking: %v", err),
			})
		}
	}

	return checks
}

// summarizeSymlinkCheck creates a check summary from symlink results
func summarizeSymlinkCheck(checks []SymlinkCheck) Check {
	check := Check{
		Name:        "Symlinks",
		Description: "Stowed config symlinks",
	}

	var ok, warning, errors int
	for _, c := range checks {
		switch c.Status {
		case StatusOK:
			ok++
		case StatusWarning:
			warning++
		case StatusError:
			errors++
		}
	}

	if errors > 0 {
		check.Status = StatusError
		check.Message = fmt.Sprintf("%d errors, %d warnings, %d ok", errors, warning, ok)
		check.Fix = "Run 'g4d stow refresh' to fix symlinks"
		return check
	}

	if warning > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("%d warnings, %d ok", warning, ok)
		check.Fix = "Run 'g4d stow add <config>' to create missing symlinks"
		return check
	}

	check.Status = StatusOK
	check.Message = fmt.Sprintf("%d symlinks verified", ok)
	return check
}

// summarizeExternalCheck creates a check summary from external status
func summarizeExternalCheck(statuses []deps.ExternalStatus) Check {
	check := Check{
		Name:        "External Dependencies",
		Description: "Cloned repos (themes, plugins)",
	}

	var installed, missing, skipped int
	for _, s := range statuses {
		switch s.Status {
		case "installed":
			installed++
		case "missing":
			missing++
		case "skipped":
			skipped++
		}
	}

	if missing > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("%d missing, %d installed, %d skipped", missing, installed, skipped)
		check.Fix = "Run 'g4d external clone' to install missing"
		return check
	}

	check.Status = StatusOK
	check.Message = fmt.Sprintf("%d installed, %d skipped", installed, skipped)
	return check
}

// summarizeMachineCheck creates a check summary from machine config status
func summarizeMachineCheck(statuses []machine.MachineConfigStatus) Check {
	check := Check{
		Name:        "Machine Configuration",
		Description: "Machine-specific config files",
	}

	var configured, missing, errors int
	for _, s := range statuses {
		switch s.Status {
		case "configured":
			configured++
		case "missing":
			missing++
		case "error":
			errors++
		}
	}

	if errors > 0 {
		check.Status = StatusError
		check.Message = fmt.Sprintf("%d errors, %d missing, %d configured", errors, missing, configured)
		return check
	}

	if missing > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("%d missing, %d configured", missing, configured)
		check.Fix = "Run 'g4d machine configure' to set up"
		return check
	}

	check.Status = StatusOK
	check.Message = fmt.Sprintf("%d configured", configured)
	return check
}

// isError returns true if the status represents an error condition
func (s CheckStatus) isError() bool {
	return s == StatusError
}

// IsHealthy returns true if all checks passed without errors
func (r *CheckResult) IsHealthy() bool {
	for _, check := range r.Checks {
		if check.Status == StatusError {
			return false
		}
	}
	return true
}

// HasWarnings returns true if any checks have warnings
func (r *CheckResult) HasWarnings() bool {
	for _, check := range r.Checks {
		if check.Status == StatusWarning {
			return true
		}
	}
	return false
}

// CountByStatus returns the count of checks by status
func (r *CheckResult) CountByStatus() (ok, warnings, errors, skipped int) {
	for _, check := range r.Checks {
		switch check.Status {
		case StatusOK:
			ok++
		case StatusWarning:
			warnings++
		case StatusError:
			errors++
		case StatusSkipped:
			skipped++
		}
	}
	return
}

// progress sends a progress message if the callback is set
func progress(opts CheckOptions, msg string) {
	if opts.ProgressFunc != nil {
		opts.ProgressFunc(msg)
	}
}
