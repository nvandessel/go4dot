package platform

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// DNFManager implements PackageManager for DNF (Fedora, RHEL 8+)
type DNFManager struct{}

func (d *DNFManager) Name() string {
	return "dnf"
}

func (d *DNFManager) IsAvailable() bool {
	return commandExists("dnf")
}

func (d *DNFManager) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Map package names
	mapped := make([]string, len(packages))
	for i, pkg := range packages {
		mapped[i] = MapPackageName(pkg, "dnf")
	}

	// Validate package names after mapping to prevent flag injection
	for _, m := range mapped {
		if err := validation.ValidatePackageName(m); err != nil {
			return fmt.Errorf("invalid package name %q: %w", m, err)
		}
	}

	args := []string{"install", "-y"}
	args = append(args, mapped...)

	cmd := exec.Command("sudo", append([]string{"dnf"}, args...)...)
	cmd.Stdout = nil // Could pipe to UI later
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	return nil
}

func (d *DNFManager) IsInstalled(pkg string) bool {
	pkg = MapPackageName(pkg, "dnf")
	output, err := runCommand("rpm", "-q", pkg)
	if err != nil {
		return false
	}
	return !strings.Contains(output, "not installed")
}

func (d *DNFManager) Update() error {
	cmd := exec.Command("sudo", "dnf", "check-update", "-y")
	// check-update returns 100 if updates are available, 0 if not
	// We just want to refresh the cache, so we ignore the exit code
	_ = cmd.Run()
	return nil
}

func (d *DNFManager) Search(query string) ([]string, error) {
	output, err := runCommand("dnf", "search", query)
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// DNF search output format: "package.arch : description"
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "=") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				pkg := strings.TrimSpace(parts[0])
				// Remove architecture suffix
				if strings.Contains(pkg, ".") {
					pkg = strings.Split(pkg, ".")[0]
				}
				results = append(results, pkg)
			}
		}
	}

	return results, nil
}

func (d *DNFManager) NeedsSudo() bool {
	return true
}
