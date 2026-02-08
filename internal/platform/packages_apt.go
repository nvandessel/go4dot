package platform

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// APTManager implements PackageManager for APT (Debian, Ubuntu)
type APTManager struct{}

func (a *APTManager) Name() string {
	return "apt"
}

func (a *APTManager) IsAvailable() bool {
	return commandExists("apt")
}

func (a *APTManager) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Map package names
	mapped := make([]string, len(packages))
	for i, pkg := range packages {
		mapped[i] = MapPackageName(pkg, "apt")
	}

	// Validate package names after mapping to prevent flag injection
	for _, m := range mapped {
		if err := validation.ValidatePackageName(m); err != nil {
			return fmt.Errorf("invalid package name %q: %w", m, err)
		}
	}

	// Set DEBIAN_FRONTEND=noninteractive to avoid prompts
	args := []string{"apt-get", "install", "-y"}
	args = append(args, mapped...)

	cmd := exec.Command("sudo", args...)
	cmd.Env = append(cmd.Env, "DEBIAN_FRONTEND=noninteractive")
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	return nil
}

func (a *APTManager) IsInstalled(pkg string) bool {
	pkg = MapPackageName(pkg, "apt")
	// Use dpkg-query to check if package is installed
	output, err := runCommand("dpkg-query", "-W", "-f=${Status}", pkg)
	if err != nil {
		return false
	}
	return strings.Contains(output, "install ok installed")
}

func (a *APTManager) Update() error {
	cmd := exec.Command("sudo", "apt-get", "update")
	cmd.Env = append(cmd.Env, "DEBIAN_FRONTEND=noninteractive")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update package cache: %w", err)
	}
	return nil
}

func (a *APTManager) Search(query string) ([]string, error) {
	output, err := runCommand("apt-cache", "search", query)
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// apt-cache search format: "package - description"
		if strings.Contains(line, " - ") {
			parts := strings.SplitN(line, " - ", 2)
			if len(parts) > 0 {
				pkg := strings.TrimSpace(parts[0])
				results = append(results, pkg)
			}
		}
	}

	return results, nil
}

func (a *APTManager) NeedsSudo() bool {
	return true
}
