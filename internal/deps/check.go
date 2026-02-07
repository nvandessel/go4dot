package deps

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

// DepStatus represents the status of a dependency
type DepStatus string

const (
	StatusInstalled       DepStatus = "installed"
	StatusMissing         DepStatus = "missing"
	StatusCheckFailed     DepStatus = "check_failed"
	StatusVersionMismatch DepStatus = "version_mismatch"
	StatusManualMissing   DepStatus = "manual_missing" // Manual dep not found; user must install
)

// DependencyCheck represents the check result for a single dependency
type DependencyCheck struct {
	Item             config.DependencyItem
	Status           DepStatus
	InstalledPath    string // Path where binary was found
	InstalledVersion string // Version found
	RequiredVersion  string // Version required
	Error            error  // Error if check failed
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
		Item:            dep,
		RequiredVersion: dep.Version,
	}

	// Determine which binary to check for
	binaryName := dep.Binary
	if binaryName == "" {
		binaryName = dep.Name
	}

	// Check if binary exists in PATH
	path, err := exec.LookPath(binaryName)
	if err != nil {
		if dep.Manual {
			check.Status = StatusManualMissing
		} else {
			check.Status = StatusMissing
		}
		return check
	}

	check.InstalledPath = path
	check.Status = StatusInstalled

	// Check version if required
	if dep.Version != "" {
		version, err := getVersion(binaryName, dep.VersionCmd)
		if err != nil {
			if dep.Manual {
				check.Status = StatusManualMissing
			} else {
				check.Status = StatusCheckFailed
			}
			check.Error = fmt.Errorf("failed to get version: %w", err)
			return check
		}
		check.InstalledVersion = version

		if !compareVersions(version, dep.Version) {
			if dep.Manual {
				check.Status = StatusManualMissing
			} else {
				check.Status = StatusVersionMismatch
			}
		}
	}

	return check
}

func getVersion(binary, cmd string) (string, error) {
	if cmd == "" {
		cmd = "--version"
	}

	args := strings.Fields(cmd)
	out, err := exec.Command(binary, args...).Output()
	if err != nil {
		return "", err
	}

	// Common version patterns: "v1.2.3", "1.2.3", "Neovim v0.10.1"
	re := regexp.MustCompile(`v?(\d+\.\d+\.\d+(?:-\w+)?)`)
	match := re.FindStringSubmatch(string(out))
	if len(match) > 1 {
		return match[1], nil
	}

	// Fallback to a simpler pattern if patch is missing
	re = regexp.MustCompile(`v?(\d+\.\d+)`)
	match = re.FindStringSubmatch(string(out))
	if len(match) > 1 {
		return match[1], nil
	}

	return strings.TrimSpace(string(out)), nil
}

func compareVersions(installed, required string) bool {
	// Handle "0.11+" format
	isAtLeast := strings.HasSuffix(required, "+")
	req := strings.TrimSuffix(required, "+")

	v1 := parseVersion(installed)
	v2 := parseVersion(req)

	if isAtLeast {
		return versionGreaterOrEqual(v1, v2)
	}

	return versionEqual(v1, v2)
}

func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var res []int
	for _, p := range parts {
		// Extract only digits from the part (handles things like 0.10.1-dev)
		re := regexp.MustCompile(`^\d+`)
		digitPart := re.FindString(p)
		if i, err := strconv.Atoi(digitPart); err == nil {
			res = append(res, i)
		}
	}
	return res
}

func versionGreaterOrEqual(v1, v2 []int) bool {
	for i := 0; i < len(v1) && i < len(v2); i++ {
		if v1[i] > v2[i] {
			return true
		}
		if v1[i] < v2[i] {
			return false
		}
	}
	return len(v1) >= len(v2)
}

func versionEqual(v1, v2 []int) bool {
	if len(v1) != len(v2) {
		return false
	}
	for i := range v1 {
		if v1[i] != v2[i] {
			return false
		}
	}
	return true
}

// GetMissing returns all missing dependencies or those with version mismatch.
// Manual dependencies (StatusManualMissing) are excluded; use GetManualMissing instead.
func (r *CheckResult) GetMissing() []DependencyCheck {
	var missing []DependencyCheck

	for _, checks := range [][]DependencyCheck{r.Critical, r.Core, r.Optional} {
		for _, check := range checks {
			if check.Status == StatusMissing || check.Status == StatusVersionMismatch {
				missing = append(missing, check)
			}
		}
	}

	return missing
}

// GetMissingCritical returns only missing critical dependencies or those with version mismatch.
// Manual dependencies are excluded.
func (r *CheckResult) GetMissingCritical() []DependencyCheck {
	var missing []DependencyCheck

	for _, dep := range r.Critical {
		if dep.Status == StatusMissing || dep.Status == StatusVersionMismatch {
			missing = append(missing, dep)
		}
	}

	return missing
}

// GetManualMissing returns all manual dependencies that are not installed.
func (r *CheckResult) GetManualMissing() []DependencyCheck {
	var missing []DependencyCheck

	for _, checks := range [][]DependencyCheck{r.Critical, r.Core, r.Optional} {
		for _, check := range checks {
			if check.Status == StatusManualMissing {
				missing = append(missing, check)
			}
		}
	}

	return missing
}

// AllInstalled returns true if all non-manual dependencies are installed.
// Manual dependencies are not considered by this check.
func (r *CheckResult) AllInstalled() bool {
	return len(r.GetMissing()) == 0
}

// Summary returns a formatted summary of the check results
func (r *CheckResult) Summary() string {
	totalInstalled := 0
	totalMissing := 0
	totalManualMissing := 0

	for _, checks := range [][]DependencyCheck{r.Critical, r.Core, r.Optional} {
		for _, check := range checks {
			switch check.Status {
			case StatusInstalled:
				totalInstalled++
			case StatusMissing:
				totalMissing++
			case StatusManualMissing:
				totalManualMissing++
			}
		}
	}

	summary := fmt.Sprintf("%d installed, %d missing", totalInstalled, totalMissing)
	if totalManualMissing > 0 {
		summary += fmt.Sprintf(", %d manual (not installed)", totalManualMissing)
	}
	return summary
}
