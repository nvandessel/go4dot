package deps

import (
	"fmt"
	"os/exec"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

// DepStatus represents the status of a dependency
type DepStatus string

const (
	StatusInstalled   DepStatus = "installed"
	StatusMissing     DepStatus = "missing"
	StatusCheckFailed DepStatus = "check_failed"
)

// DependencyCheck represents the check result for a single dependency
type DependencyCheck struct {
	Item          config.DependencyItem
	Status        DepStatus
	InstalledPath string // Path where binary was found
	Error         error  // Error if check failed
}

// CheckResult contains the results of checking all dependencies
type CheckResult struct {
	Critical []DependencyCheck
	Core     []DependencyCheck
	Optional []DependencyCheck
}

// Check verifies if dependencies are installed
func Check(cfg *config.Config, p *platform.Platform) (*CheckResult, error) {
	result := &CheckResult{}

	// Check critical dependencies
	for _, dep := range cfg.Dependencies.Critical {
		check := checkDependency(dep)
		result.Critical = append(result.Critical, check)
	}

	// Check core dependencies
	for _, dep := range cfg.Dependencies.Core {
		check := checkDependency(dep)
		result.Core = append(result.Core, check)
	}

	// Check optional dependencies
	for _, dep := range cfg.Dependencies.Optional {
		check := checkDependency(dep)
		result.Optional = append(result.Optional, check)
	}

	return result, nil
}

// checkDependency checks if a single dependency is installed
func checkDependency(dep config.DependencyItem) DependencyCheck {
	check := DependencyCheck{
		Item: dep,
	}

	// Determine which binary to check for
	binaryName := dep.Binary
	if binaryName == "" {
		binaryName = dep.Name
	}

	// Check if binary exists in PATH
	path, err := exec.LookPath(binaryName)
	if err != nil {
		check.Status = StatusMissing
		return check
	}

	check.Status = StatusInstalled
	check.InstalledPath = path
	return check
}

// GetMissing returns all missing dependencies
func (r *CheckResult) GetMissing() []DependencyCheck {
	var missing []DependencyCheck

	for _, dep := range r.Critical {
		if dep.Status == StatusMissing {
			missing = append(missing, dep)
		}
	}

	for _, dep := range r.Core {
		if dep.Status == StatusMissing {
			missing = append(missing, dep)
		}
	}

	for _, dep := range r.Optional {
		if dep.Status == StatusMissing {
			missing = append(missing, dep)
		}
	}

	return missing
}

// GetMissingCritical returns only missing critical dependencies
func (r *CheckResult) GetMissingCritical() []DependencyCheck {
	var missing []DependencyCheck

	for _, dep := range r.Critical {
		if dep.Status == StatusMissing {
			missing = append(missing, dep)
		}
	}

	return missing
}

// AllInstalled returns true if all dependencies are installed
func (r *CheckResult) AllInstalled() bool {
	return len(r.GetMissing()) == 0
}

// Summary returns a formatted summary of the check results
func (r *CheckResult) Summary() string {
	totalInstalled := 0
	totalMissing := 0

	for _, checks := range [][]DependencyCheck{r.Critical, r.Core, r.Optional} {
		for _, check := range checks {
			if check.Status == StatusInstalled {
				totalInstalled++
			} else if check.Status == StatusMissing {
				totalMissing++
			}
		}
	}

	return fmt.Sprintf("%d installed, %d missing", totalInstalled, totalMissing)
}
