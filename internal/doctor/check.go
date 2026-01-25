package doctor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
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
	Platform              *platform.Platform
	Checks                []Check
	DepsResult            *deps.CheckResult
	ExternalStatus        []deps.ExternalStatus
	MachineStatus         []machine.MachineConfigStatus
	DriftSummary          *stow.DriftSummary
	UnmanagedLinks        []UnmanagedSymlink
	AdoptionOpportunities []AdoptionOpportunity
}

// SymlinkCheck represents the status of a stowed symlink
type SymlinkCheck struct {
	Config     string
	TargetPath string
	Status     CheckStatus
	Message    string
}

// UnmanagedSymlink represents a symlink pointing to dotfiles but not in config
type UnmanagedSymlink struct {
	TargetPath string
	SourcePath string
}

// AdoptionOpportunity represents a config that could be adopted into state
type AdoptionOpportunity struct {
	ConfigName    string
	LinkedCount   int
	TotalCount    int
	IsFullyLinked bool
}

// CheckOptions configures the health check behavior
type CheckOptions struct {
	DotfilesPath string
	ProgressFunc func(current, total int, msg string)
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
		driftSummary, err := stow.FullDriftCheck(cfg, opts.DotfilesPath)
		if err != nil {
			result.Checks = append(result.Checks, Check{
				Name:        "Symlinks",
				Description: "Check stowed config symlinks",
				Status:      StatusError,
				Message:     fmt.Sprintf("Drift check failed: %v", err),
			})
		} else {
			result.DriftSummary = driftSummary
			symlinkCheck := summarizeDriftCheck(driftSummary)
			result.Checks = append(result.Checks, symlinkCheck)
		}
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

	// Step 8: Check for unmanaged symlinks
	progress(opts, "Checking for unmanaged symlinks...")
	if opts.DotfilesPath != "" {
		unmanaged := checkUnmanagedSymlinks(cfg, opts.DotfilesPath)
		result.UnmanagedLinks = unmanaged
		if len(unmanaged) > 0 {
			result.Checks = append(result.Checks, Check{
				Name:        "Unmanaged Symlinks",
				Description: "Symlinks pointing to dotfiles but not in config",
				Status:      StatusWarning,
				Message:     fmt.Sprintf("%d unmanaged symlinks found", len(unmanaged)),
				Fix:         "Add these to your .go4dot.yaml or remove them",
			})
		} else {
			result.Checks = append(result.Checks, Check{
				Name:        "Unmanaged Symlinks",
				Description: "Symlinks pointing to dotfiles but not in config",
				Status:      StatusOK,
				Message:     "No unmanaged symlinks found",
			})
		}
	}

	// Step 9: Check for adoption opportunities
	progress(opts, "Checking for adoption opportunities...")
	if opts.DotfilesPath != "" {
		opportunities := checkAdoptionOpportunities(cfg, opts.DotfilesPath)
		result.AdoptionOpportunities = opportunities
		if len(opportunities) > 0 {
			fullyLinked := 0
			for _, op := range opportunities {
				if op.IsFullyLinked {
					fullyLinked++
				}
			}
			if fullyLinked > 0 {
				result.Checks = append(result.Checks, Check{
					Name:        "Adoption Opportunities",
					Description: "Configs with existing symlinks not in state",
					Status:      StatusWarning,
					Message:     fmt.Sprintf("%d config(s) can be adopted", fullyLinked),
					Fix:         "Run 'g4d adopt' to adopt existing symlinks into state",
				})
			}
		}
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

// summarizeDriftCheck creates a check summary from drift check result
func summarizeDriftCheck(summary *stow.DriftSummary) Check {
	check := Check{
		Name:        "Symlinks",
		Description: "Stowed config symlinks",
	}

	if summary.DriftedConfigs > 0 {
		var conflicts, missing, news int
		for _, r := range summary.Results {
			conflicts += len(r.ConflictFiles)
			missing += len(r.MissingFiles)
			news += len(r.NewFiles)
		}

		if conflicts > 0 {
			check.Status = StatusError
			check.Message = fmt.Sprintf("%d configs have drift (%d conflicts, %d missing, %d new)",
				summary.DriftedConfigs, conflicts, missing, news)
			check.Fix = "Run 'g4d stow refresh' to fix symlinks"
		} else {
			check.Status = StatusWarning
			check.Message = fmt.Sprintf("%d configs have drift (%d missing, %d new)",
				summary.DriftedConfigs, missing, news)
			check.Fix = "Run 'g4d stow refresh' to update symlinks"
		}
		return check
	}

	if len(summary.RemovedConfigs) > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("%d configs in state are missing from configuration", len(summary.RemovedConfigs))
		check.Fix = "Run 'g4d stow remove' or update your config"
		return check
	}

	check.Status = StatusOK
	check.Message = fmt.Sprintf("%d configs verified", summary.TotalConfigs)
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
		opts.ProgressFunc(0, 0, msg)
	}
}

// checkUnmanagedSymlinks finds symlinks in home pointing to dotfiles but not in config
func checkUnmanagedSymlinks(cfg *config.Config, dotfilesPath string) []UnmanagedSymlink {
	var unmanaged []UnmanagedSymlink
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	absDotfiles, err := filepath.Abs(dotfilesPath)
	if err != nil {
		absDotfiles = dotfilesPath
	}

	// Map of managed target paths for quick lookup
	managedTargets := make(map[string]bool)
	allConfigs := cfg.GetAllConfigs()
	for _, configItem := range allConfigs {
		configPath := filepath.Join(absDotfiles, configItem.Path)
		_ = filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err == nil {
				relPath, _ := filepath.Rel(configPath, path)
				if relPath != "." {
					targetPath := filepath.Join(home, relPath)
					managedTargets[filepath.Clean(targetPath)] = true
				}
			}
			return nil
		})
	}

	// Directories to scan for symlinks
	scanDirs := []string{
		home,
		filepath.Join(home, ".config"),
		filepath.Join(home, ".local", "bin"),
		filepath.Join(home, ".local", "share"),
	}

	for _, scanDir := range scanDirs {
		if _, err := os.Stat(scanDir); os.IsNotExist(err) {
			continue
		}

		// Use a simple walk to catch symlinks in these directories
		// We limit depth manually to avoid scanning the entire home dir
		_ = filepath.Walk(scanDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return filepath.SkipDir
			}

			// Calculate depth relative to scanDir
			rel, _ := filepath.Rel(scanDir, path)
			depth := 0
			if rel != "." {
				depth = len(strings.Split(rel, string(filepath.Separator)))
			}

			// Limit depth
			if info.IsDir() {
				// Don't go deep into directories that are not common config locations
				if depth > 2 {
					return filepath.SkipDir
				}
				// Skip hidden directories in home except .config etc
				if scanDir == home && strings.HasPrefix(info.Name(), ".") &&
					info.Name() != ".config" && info.Name() != ".local" && info.Name() != ".ssh" && info.Name() != ".gnupg" {
					if depth == 1 {
						// We still want to check top-level hidden files, so don't skip yet,
						// but don't recurse into them if they are not the ones we want.
						// Actually, most hidden files are just files, not dirs.
						// If it's a dir like .cache, skip it.
						if info.Name() == ".cache" || info.Name() == ".git" || info.Name() == ".mozilla" || info.Name() == ".var" {
							return filepath.SkipDir
						}
					}
				}
				return nil
			}

			if info.Mode()&os.ModeSymlink == 0 {
				return nil
			}

			// It's a symlink, check where it points
			linkDest, err := os.Readlink(path)
			if err != nil {
				return nil
			}

			// Resolve to absolute path
			if !filepath.IsAbs(linkDest) {
				linkDest = filepath.Join(filepath.Dir(path), linkDest)
			}
			linkDest = filepath.Clean(linkDest)

			// Check if it points into dotfiles
			if strings.HasPrefix(linkDest, absDotfiles) {
				if !managedTargets[filepath.Clean(path)] {
					unmanaged = append(unmanaged, UnmanagedSymlink{
						TargetPath: path,
						SourcePath: linkDest,
					})
				}
			}
			return nil
		})
	}

	return unmanaged
}

// checkAdoptionOpportunities finds configs with existing symlinks that aren't in state
func checkAdoptionOpportunities(cfg *config.Config, dotfilesPath string) []AdoptionOpportunity {
	var opportunities []AdoptionOpportunity

	// Load current state to see what's already tracked
	st, err := state.Load()
	if err != nil {
		return nil
	}

	// Get set of installed config names
	installedConfigs := make(map[string]bool)
	if st != nil {
		installedConfigs = st.GetInstalledConfigNames()
	}

	// Scan for existing symlinks
	summary, err := stow.ScanExistingSymlinks(cfg, dotfilesPath)
	if err != nil {
		return nil
	}

	// Find configs that have symlinks but aren't in state
	for _, result := range summary.Results {
		if installedConfigs[result.ConfigName] {
			// Already in state, skip
			continue
		}

		if len(result.LinkedFiles) > 0 {
			// Has some symlinks but not in state - adoption opportunity
			opportunities = append(opportunities, AdoptionOpportunity{
				ConfigName:    result.ConfigName,
				LinkedCount:   len(result.LinkedFiles),
				TotalCount:    result.TotalFiles,
				IsFullyLinked: result.IsFullyLinked(),
			})
		}
	}

	return opportunities
}
