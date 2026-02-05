package stow

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

// countFiles counts the number of files (not directories) in a directory tree.
func countFiles(dir string) (int, error) {
	count := 0

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("counting files in %s: %w", dir, err)
	}

	return count, nil
}

// UpdateSymlinkCounts updates the stored file counts for all configs in state.
func UpdateSymlinkCounts(cfg *config.Config, dotfilesPath string, st *state.State) error {
	allConfigs := cfg.GetAllConfigs()

	for _, configItem := range allConfigs {
		configPath := filepath.Join(dotfilesPath, configItem.Path)

		count, err := countFiles(configPath)
		if err != nil {
			// Only treat "not exist" errors as missing config - remove from state
			if errors.Is(err, os.ErrNotExist) {
				st.RemoveSymlinkCount(configItem.Name)
				continue
			}
			// Surface other errors (permission, IO, etc.)
			return fmt.Errorf("updating symlink count for %s: %w", configItem.Name, err)
		}

		st.SetSymlinkCount(configItem.Name, count)
	}

	if err := st.Save(); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}
	return nil
}

// findOrphanedSymlinks finds symlinks in home that point to the given config directory
// but no longer have a corresponding file in that directory.
func findOrphanedSymlinks(configPath, home string) []string {
	var orphans []string

	// We need to find which directories in home might contain symlinks to configPath.
	// A simple approach is to walk the configPath to see what directories it HAS,
	// and then check those same directories in home.
	dirsToCheck := make(map[string]bool)
	dirsToCheck["."] = true
	_ = filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			rel, err := filepath.Rel(configPath, path)
			if err == nil {
				dirsToCheck[rel] = true
			}
		}
		return nil
	})

	for relDir := range dirsToCheck {
		targetDir := filepath.Join(home, relDir)
		entries, err := os.ReadDir(targetDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// We only care about symlinks
			if entry.Type()&os.ModeSymlink != 0 {
				targetPath := filepath.Join(targetDir, entry.Name())
				linkDest, err := os.Readlink(targetPath)
				if err != nil {
					continue
				}

				// Resolve to absolute path
				absLinkDest := linkDest
				if !filepath.IsAbs(absLinkDest) {
					absLinkDest = filepath.Join(targetDir, linkDest)
				}
				absLinkDest = filepath.Clean(absLinkDest)

				// If it points into our configPath
				relToConfig, err := filepath.Rel(configPath, absLinkDest)
				if err == nil && !strings.HasPrefix(relToConfig, "..") && relToConfig != ".." {
					// Check if the source file still exists
					if _, err := os.Stat(absLinkDest); os.IsNotExist(err) {
						relToHome, _ := filepath.Rel(home, targetPath)
						orphans = append(orphans, relToHome)
					}
				}
			}
		}
	}

	return orphans
}
