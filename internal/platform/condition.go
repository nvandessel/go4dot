package platform

import (
	"strings"
)

// CheckCondition evaluates if a condition is met based on platform information.
// Conditions are a map of key-value pairs where keys can be:
// - platform, os: linux, darwin, windows
// - distro: fedora, ubuntu, debian, arch, etc.
// - package_manager: dnf, apt, brew, pacman, etc.
// - wsl: true, false
// - arch, architecture: amd64, arm64, etc.
func CheckCondition(condition map[string]string, p *Platform) bool {
	if len(condition) == 0 {
		return true // No condition means always true
	}

	for key, value := range condition {
		switch key {
		case "platform", "os":
			if !matchesValue(p.OS, value) {
				return false
			}
		case "distro":
			if !matchesValue(p.Distro, value) {
				return false
			}
		case "package_manager":
			if !matchesValue(p.PackageManager, value) {
				return false
			}
		case "wsl":
			if value == "true" && !p.IsWSL {
				return false
			}
			if value == "false" && p.IsWSL {
				return false
			}
		case "arch", "architecture":
			if !matchesValue(p.Architecture, value) {
				return false
			}
		}
	}
	return true
}

// matchesValue checks if actual matches expected (supports comma-separated list)
func matchesValue(actual, expected string) bool {
	// Support comma-separated values (e.g., "linux,darwin")
	values := strings.Split(expected, ",")
	for _, v := range values {
		if strings.TrimSpace(v) == actual {
			return true
		}
	}
	return false
}
