package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// scanDirectory scans a directory for potential dotfile configurations.
// It returns a list of ConfigItems representing directories that appear to be
// dotfile-related (e.g., nvim, tmux, zsh).
func scanDirectory(root string) ([]ConfigItem, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var items []ConfigItem

	// Directories to always ignore (not dotfiles-related)
	ignored := map[string]bool{
		// Version control
		".git":    true,
		".github": true,
		".gitlab": true,
		".svn":    true,

		// IDE/Editor
		".idea":   true,
		".vscode": true,
		".vim":    false, // This IS a dotfile config
		".nvim":   false, // This IS a dotfile config

		// Build/Output
		"bin":          true,
		"build":        true,
		"dist":         true,
		"node_modules": true,
		"vendor":       true,
		"target":       true,
		"__pycache__":  true,
		".cache":       true,

		// Project files (not dotfiles)
		ConfigFileName: true,
		"README.md":    true,
		"LICENSE":      true,
		"Makefile":     true,
		"go.mod":       true,
		"go.sum":       true,
		"package.json": true,
		"Cargo.toml":   true,

		// go4dot internal
		"test":    true,
		"sandbox": true,
	}

	for _, entry := range entries {
		name := entry.Name()

		// Check explicit ignore list
		if ignored[name] {
			continue
		}

		// Only include directories (dotfiles are usually directories for stow)
		if !entry.IsDir() {
			continue
		}

		// Skip hidden directories that start with . unless they look like dotfile configs
		// (e.g., .config is OK, .cache is not)
		if len(name) > 1 && name[0] == '.' {
			// Common hidden dotfile configs to include
			validHiddenDirs := map[string]bool{
				".config":      true,
				".local":       true,
				".vim":         true,
				".nvim":        true,
				".emacs.d":     true,
				".tmux":        true,
				".ssh":         true,
				".gnupg":       true,
				".fonts":       true,
				".themes":      true,
				".icons":       true,
				".mozilla":     true,
				".thunderbird": true,
			}
			if !validHiddenDirs[name] {
				continue
			}
		}

		items = append(items, ConfigItem{
			Name:        name,
			Path:        name,
			Description: fmt.Sprintf("%s configuration", name),
			Platforms:   []string{"linux", "macos"},
		})
	}

	return items, nil
}

// slugify converts a string to a URL-friendly slug.
// It lowercases the string and replaces non-alphanumeric characters with hyphens.
func slugify(s string) string {
	s = strings.ToLower(s)
	// Replace non-alphanumeric chars with hyphens
	reg := regexp.MustCompile("[^a-z0-9]+")
	s = reg.ReplaceAllString(s, "-")
	// Trim hyphens
	s = strings.Trim(s, "-")
	return s
}
