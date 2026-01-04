package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

// PacmanManager implements PackageManager for Pacman (Arch Linux, Manjaro)
type PacmanManager struct{}

func (p *PacmanManager) Name() string {
	return "pacman"
}

func (p *PacmanManager) IsAvailable() bool {
	return commandExists("pacman")
}

func (p *PacmanManager) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Map package names
	mapped := make([]string, len(packages))
	for i, pkg := range packages {
		mapped[i] = MapPackageName(pkg, "pacman")
	}

	args := []string{"-S", "--noconfirm"}
	args = append(args, mapped...)

	cmd := exec.Command("sudo", append([]string{"pacman"}, args...)...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	return nil
}

func (p *PacmanManager) IsInstalled(pkg string) bool {
	pkg = MapPackageName(pkg, "pacman")
	// pacman -Q returns info if package is installed
	_, err := runCommand("pacman", "-Q", pkg)
	return err == nil
}

func (p *PacmanManager) Update() error {
	cmd := exec.Command("sudo", "pacman", "-Sy")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update package database: %w", err)
	}
	return nil
}

func (p *PacmanManager) Search(query string) ([]string, error) {
	output, err := runCommand("pacman", "-Ss", query)
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Pacman search format: "repo/package version"
		if strings.Contains(line, "/") && !strings.HasPrefix(line, " ") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				// Extract package name after /
				pkgParts := strings.Split(parts[0], "/")
				if len(pkgParts) > 1 {
					results = append(results, pkgParts[1])
				}
			}
		}
	}

	return results, nil
}

func (p *PacmanManager) NeedsSudo() bool {
	return true
}
