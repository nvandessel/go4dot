package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

// PackageManager defines the interface for package management operations
type PackageManager interface {
	// Name returns the package manager name (e.g., "dnf", "apt", "brew")
	Name() string

	// IsAvailable checks if the package manager is available on the system
	IsAvailable() bool

	// Install installs one or more packages
	Install(packages ...string) error

	// IsInstalled checks if a package is installed
	IsInstalled(pkg string) bool

	// Update updates the package cache/repository information
	Update() error

	// Search searches for packages matching a query
	Search(query string) ([]string, error)

	// NeedsSudo returns true if the package manager requires sudo for installation
	NeedsSudo() bool
}

// GetPackageManager returns the appropriate package manager for the platform
func GetPackageManager(p *Platform) (PackageManager, error) {
	switch p.PackageManager {
	case "dnf":
		return &DNFManager{}, nil
	case "yum":
		return &YumManager{}, nil
	case "apt":
		return &APTManager{}, nil
	case "brew":
		return &BrewManager{}, nil
	case "pacman":
		return &PacmanManager{}, nil
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", p.PackageManager)
	}
}

// runCommand executes a command and returns the output
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// commandExists checks if a command exists in PATH
func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// MapPackageName maps a generic/canonical package name to a manager-specific
// name. For example, "fd" becomes "fd-find" on apt and dnf, while remaining
// "fd" on brew and pacman.
//
// Unknown packages are returned unchanged, making this safe to call with any
// package name.
func MapPackageName(genericName string, manager string) string {
	return ResolvePackageName(genericName, manager)
}
