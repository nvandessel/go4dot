package platform

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// BrewManager implements PackageManager for Homebrew (macOS, Linux)
type BrewManager struct{}

func (b *BrewManager) Name() string {
	return "brew"
}

func (b *BrewManager) IsAvailable() bool {
	return commandExists("brew")
}

func (b *BrewManager) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Map package names
	mapped := make([]string, len(packages))
	for i, pkg := range packages {
		mapped[i] = MapPackageName(pkg, "brew")
	}

	// Validate package names after mapping to prevent flag injection
	for _, m := range mapped {
		if err := validation.ValidatePackageName(m); err != nil {
			return fmt.Errorf("invalid package name %q: %w", m, err)
		}
	}

	args := []string{"install"}
	args = append(args, mapped...)

	cmd := exec.Command("brew", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	return nil
}

func (b *BrewManager) IsInstalled(pkg string) bool {
	pkg = MapPackageName(pkg, "brew")
	// brew list --formula returns list of installed formula packages
	output, err := runCommand("brew", "list", "--formula")
	if err != nil {
		return false
	}

	// Check if package is in the list
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == pkg {
			return true
		}
	}

	return false
}

func (b *BrewManager) Update() error {
	cmd := exec.Command("brew", "update")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update brew: %w", err)
	}
	return nil
}

func (b *BrewManager) Search(query string) ([]string, error) {
	output, err := runCommand("brew", "search", query)
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "==>") {
			results = append(results, line)
		}
	}

	return results, nil
}

func (b *BrewManager) NeedsSudo() bool {
	// Homebrew doesn't need sudo
	return false
}
