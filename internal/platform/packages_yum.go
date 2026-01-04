package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

// YumManager implements PackageManager for YUM (RHEL 7, CentOS 7)
type YumManager struct{}

func (y *YumManager) Name() string {
	return "yum"
}

func (y *YumManager) IsAvailable() bool {
	return commandExists("yum")
}

func (y *YumManager) Install(packages ...string) error {
	if len(packages) == 0 {
		return nil
	}

	// Map package names
	mapped := make([]string, len(packages))
	for i, pkg := range packages {
		mapped[i] = MapPackageName(pkg, "yum")
	}

	args := []string{"install", "-y"}
	args = append(args, mapped...)

	cmd := exec.Command("sudo", append([]string{"yum"}, args...)...)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	return nil
}

func (y *YumManager) IsInstalled(pkg string) bool {
	pkg = MapPackageName(pkg, "yum")
	output, err := runCommand("rpm", "-q", pkg)
	if err != nil {
		return false
	}
	return !strings.Contains(output, "not installed")
}

func (y *YumManager) Update() error {
	cmd := exec.Command("sudo", "yum", "check-update", "-y")
	_ = cmd.Run()
	return nil
}

func (y *YumManager) Search(query string) ([]string, error) {
	output, err := runCommand("yum", "search", query)
	if err != nil {
		return nil, err
	}

	var results []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, ":") && !strings.HasPrefix(line, "=") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 0 {
				pkg := strings.TrimSpace(parts[0])
				if strings.Contains(pkg, ".") {
					pkg = strings.Split(pkg, ".")[0]
				}
				results = append(results, pkg)
			}
		}
	}

	return results, nil
}

func (y *YumManager) NeedsSudo() bool {
	return true
}
