package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Platform represents the detected platform information
type Platform struct {
	OS             string // linux, darwin, windows
	Distro         string // fedora, ubuntu, debian, arch, etc. (Linux only)
	DistroVersion  string // version number
	IsWSL          bool   // true if running under WSL
	PackageManager string // dnf, apt, brew, pacman, etc.
	Architecture   string // amd64, arm64, etc.
}

// Detect returns the current platform information
func Detect() (*Platform, error) {
	p := &Platform{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
	}

	p.IsWSL = detectWSL()
	switch p.OS {
	case "linux":
		if err := detectLinuxDistro(p); err != nil {
			return nil, fmt.Errorf("failed to detect Linux distro: %w", err)
		}
		detectLinuxPackageManager(p)
	case "darwin":
		detectMacOSPackageManager(p)
	case "windows":
		detectWindowsPackageManager(p)
	}

	return p, nil
}

// detectWindowsPackageManager checks for winget, choco, or scoop
func detectWindowsPackageManager(p *Platform) {
	if _, err := exec.LookPath("winget"); err == nil {
		p.PackageManager = "winget"
	} else if _, err := exec.LookPath("choco"); err == nil {
		p.PackageManager = "choco"
	} else if _, err := exec.LookPath("scoop"); err == nil {
		p.PackageManager = "scoop"
	} else {
		p.PackageManager = "none"
	}
}

// detectWSL checks if we're running under Windows Subsystem for Linux
func detectWSL() bool {
	// Check for WSL in /proc/version
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}

	content := strings.ToLower(string(data))
	return strings.Contains(content, "microsoft") || strings.Contains(content, "wsl")
}

// detectLinuxDistro parses /etc/os-release to determine the distro
func detectLinuxDistro(p *Platform) error {
	file, err := os.Open("/etc/os-release")
	if err != nil {
		// Try alternative location
		file, err = os.Open("/usr/lib/os-release")
		if err != nil {
			return fmt.Errorf("could not open os-release file: %w", err)
		}
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	osInfo := make(map[string]string)

	// Parse key=value pairs
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
		osInfo[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading os-release: %w", err)
	}

	// Extract distro information
	if id, ok := osInfo["ID"]; ok {
		p.Distro = strings.ToLower(id)
	}
	if version, ok := osInfo["VERSION_ID"]; ok {
		p.DistroVersion = version
	}

	return nil
}

// detectLinuxPackageManager determines which package manager is available
func detectLinuxPackageManager(p *Platform) {
	// Order matters - check most specific first
	managers := []struct {
		name   string
		binary string
	}{
		{"dnf", "dnf"},       // Fedora, RHEL 8+
		{"yum", "yum"},       // RHEL 7, CentOS 7
		{"apt", "apt"},       // Debian, Ubuntu
		{"pacman", "pacman"}, // Arch, Manjaro
		{"zypper", "zypper"}, // openSUSE
		{"apk", "apk"},       // Alpine
	}

	for _, mgr := range managers {
		if _, err := exec.LookPath(mgr.binary); err == nil {
			p.PackageManager = mgr.name
			return
		}
	}

	p.PackageManager = "unknown"
}

// detectMacOSPackageManager checks for Homebrew
func detectMacOSPackageManager(p *Platform) {
	if _, err := exec.LookPath("brew"); err == nil {
		p.PackageManager = "brew"
	} else {
		p.PackageManager = "none"
	}
}

// String returns a human-readable representation of the platform
func (p *Platform) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "OS: %s", p.OS)

	if p.OS == "linux" {
		fmt.Fprintf(&sb, "\nDistro: %s", p.Distro)
		if p.DistroVersion != "" {
			fmt.Fprintf(&sb, " %s", p.DistroVersion)
		}
		if p.IsWSL {
			sb.WriteString(" (WSL)")
		}
	}

	fmt.Fprintf(&sb, "\nArchitecture: %s", p.Architecture)
	fmt.Fprintf(&sb, "\nPackage Manager: %s", p.PackageManager)

	return sb.String()
}

// IsLinux returns true if the platform is Linux (including WSL)
func (p *Platform) IsLinux() bool {
	return p.OS == "linux"
}

// IsMacOS returns true if the platform is macOS
func (p *Platform) IsMacOS() bool {
	return p.OS == "darwin"
}

// IsWindows returns true if the platform is Windows
func (p *Platform) IsWindows() bool {
	return p.OS == "windows"
}

// SupportsPackageManager returns true if a package manager was detected
func (p *Platform) SupportsPackageManager() bool {
	return p.PackageManager != "unknown" && p.PackageManager != "none"
}
