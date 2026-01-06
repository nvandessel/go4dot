package stow

import (
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

// AdoptResult represents the result of scanning a config for existing symlinks
type AdoptResult struct {
	ConfigName   string   // Name of the config (e.g., "nvim")
	ConfigPath   string   // Path within dotfiles (e.g., "nvim")
	LinkedFiles  []string // Files that are already correctly symlinked
	MissingFiles []string // Files in dotfiles but not symlinked
	TotalFiles   int      // Total files in the config
	IsCore       bool     // Whether this is a core config
}

// IsFullyLinked returns true if all files are correctly symlinked
func (r *AdoptResult) IsFullyLinked() bool {
	return len(r.MissingFiles) == 0 && len(r.LinkedFiles) > 0
}

// IsPartiallyLinked returns true if some but not all files are symlinked
func (r *AdoptResult) IsPartiallyLinked() bool {
	return len(r.LinkedFiles) > 0 && len(r.MissingFiles) > 0
}

// IsNotLinked returns true if no files are symlinked
func (r *AdoptResult) IsNotLinked() bool {
	return len(r.LinkedFiles) == 0
}

// AdoptSummary provides an overview of adoption across all configs
type AdoptSummary struct {
	Results         []AdoptResult
	FullyLinked     int // Configs where all files are linked
	PartiallyLinked int // Configs where some files are linked
	NotLinked       int // Configs where no files are linked
}

// HasAdoptableConfigs returns true if there are any configs that can be adopted
func (s *AdoptSummary) HasAdoptableConfigs() bool {
	return s.FullyLinked > 0
}

// GetFullyLinked returns only configs that are fully linked
func (s *AdoptSummary) GetFullyLinked() []AdoptResult {
	var results []AdoptResult
	for _, r := range s.Results {
		if r.IsFullyLinked() {
			results = append(results, r)
		}
	}
	return results
}

// GetPartiallyLinked returns only configs that are partially linked
func (s *AdoptSummary) GetPartiallyLinked() []AdoptResult {
	var results []AdoptResult
	for _, r := range s.Results {
		if r.IsPartiallyLinked() {
			results = append(results, r)
		}
	}
	return results
}

// GetNotLinked returns only configs that have no symlinks
func (s *AdoptSummary) GetNotLinked() []AdoptResult {
	var results []AdoptResult
	for _, r := range s.Results {
		if r.IsNotLinked() {
			results = append(results, r)
		}
	}
	return results
}

// ScanExistingSymlinks scans all configs and identifies which files are already correctly symlinked.
// This is used to detect pre-existing stow setups that should be adopted into go4dot state.
func ScanExistingSymlinks(cfg *config.Config, dotfilesPath string) (*AdoptSummary, error) {
	summary := &AdoptSummary{}
	home := os.Getenv("HOME")

	// Process core configs
	for _, configItem := range cfg.Configs.Core {
		result, err := scanConfigSymlinks(configItem, dotfilesPath, home, true)
		if err != nil {
			continue
		}
		summary.Results = append(summary.Results, *result)
	}

	// Process optional configs
	for _, configItem := range cfg.Configs.Optional {
		result, err := scanConfigSymlinks(configItem, dotfilesPath, home, false)
		if err != nil {
			continue
		}
		summary.Results = append(summary.Results, *result)
	}

	// Calculate summary counts
	for _, r := range summary.Results {
		switch {
		case r.IsFullyLinked():
			summary.FullyLinked++
		case r.IsPartiallyLinked():
			summary.PartiallyLinked++
		case r.IsNotLinked():
			summary.NotLinked++
		}
	}

	return summary, nil
}

// scanConfigSymlinks checks a single config for existing symlinks
func scanConfigSymlinks(configItem config.ConfigItem, dotfilesPath, home string, isCore bool) (*AdoptResult, error) {
	configPath := filepath.Join(dotfilesPath, configItem.Path)

	result := &AdoptResult{
		ConfigName: configItem.Name,
		ConfigPath: configItem.Path,
		IsCore:     isCore,
	}

	// Check if config directory exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return result, nil
	}

	// Walk the config directory and check each file
	err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip on error
		}
		if info.IsDir() {
			return nil // Skip directories
		}

		result.TotalFiles++

		// Calculate expected target path in home
		relPath, _ := filepath.Rel(configPath, path)
		targetPath := filepath.Join(home, relPath)

		// Check if the symlink exists and is correct
		if isCorrectlyLinked(path, targetPath) {
			result.LinkedFiles = append(result.LinkedFiles, relPath)
		} else {
			result.MissingFiles = append(result.MissingFiles, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// isCorrectlyLinked checks if targetPath is a symlink pointing to sourcePath
func isCorrectlyLinked(sourcePath, targetPath string) bool {
	targetInfo, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}

	// Check if it's a symlink
	if targetInfo.Mode()&os.ModeSymlink == 0 {
		// Not a symlink - check if it's the same file (handles directory folding)
		sourceInfo, err := os.Stat(sourcePath)
		if err != nil {
			return false
		}
		// os.SameFile handles the case where stow created a directory symlink
		// that makes the files accessible via the target path
		return os.SameFile(sourceInfo, targetInfo)
	}

	// It's a symlink - check if it points to the correct location
	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		return false
	}

	// Resolve to absolute path (stow creates relative symlinks)
	if !filepath.IsAbs(linkDest) {
		linkDest = filepath.Join(filepath.Dir(targetPath), linkDest)
	}
	linkDest = filepath.Clean(linkDest)

	return linkDest == sourcePath
}

// AdoptExistingSymlinks scans for existing symlinks and updates state for configs that are fully linked.
// This allows go4dot to take over management of pre-existing stow symlinks.
func AdoptExistingSymlinks(cfg *config.Config, dotfilesPath string, st *state.State, force bool) (*AdoptSummary, error) {
	summary, err := ScanExistingSymlinks(cfg, dotfilesPath)
	if err != nil {
		return nil, err
	}

	// Adopt fully linked configs
	for _, result := range summary.Results {
		if result.IsFullyLinked() || (force && result.IsPartiallyLinked()) {
			st.AddConfig(result.ConfigName, result.ConfigPath, result.IsCore)
			st.SetSymlinkCount(result.ConfigName, len(result.LinkedFiles))
		}
	}

	if err := st.Save(); err != nil {
		return nil, err
	}

	return summary, nil
}

// GetConfigLinkStatus returns the link status for a single config
func GetConfigLinkStatus(configItem config.ConfigItem, dotfilesPath string) (*AdoptResult, error) {
	home := os.Getenv("HOME")
	return scanConfigSymlinks(configItem, dotfilesPath, home, false)
}
