package stow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

// DriftResult represents the drift status for a single config
type DriftResult struct {
	ConfigName    string   // Name of the config (e.g., "nvim")
	ConfigPath    string   // Path within dotfiles (e.g., "nvim")
	CurrentCount  int      // Current file count in the config directory
	StoredCount   int      // File count stored in state
	HasDrift      bool     // True if counts differ
	NewFiles      []string // Files in dotfiles but not symlinked (populated by FullDriftCheck)
	MissingFiles  []string // Symlinks pointing to deleted files
	ConflictFiles []string // Files that exist in home but aren't symlinks
}

// DriftSummary provides an overview of drift across all configs
type DriftSummary struct {
	TotalConfigs   int
	DriftedConfigs int
	TotalNewFiles  int
	Results        []DriftResult
	RemovedConfigs []string // Configs in state but not in current config
}

// HasDrift returns true if any config has drift or if there are removed configs
func (s *DriftSummary) HasDrift() bool {
	return s.DriftedConfigs > 0 || len(s.RemovedConfigs) > 0
}

// FullDriftCheck performs a complete analysis of all configs.
// It identifies exactly which files are new, missing, or in conflict.
func FullDriftCheck(cfg *config.Config, dotfilesPath string) (*DriftSummary, error) {
	home := os.Getenv("HOME")
	st, _ := state.Load() // Try to load state, ignore error if not found
	return FullDriftCheckWithHome(cfg, dotfilesPath, home, st)
}

// FullDriftCheckWithHome performs a complete analysis of all configs using a specific home directory.
func FullDriftCheckWithHome(cfg *config.Config, dotfilesPath, home string, st *state.State) (*DriftSummary, error) {
	var results []DriftResult

	allConfigs := cfg.GetAllConfigs()
	for _, configItem := range allConfigs {
		configPath := filepath.Join(dotfilesPath, configItem.Path)

		result := DriftResult{
			ConfigName: configItem.Name,
			ConfigPath: configItem.Path,
		}

		// Check if config directory exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		// Walk the config directory and check each file
		err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip on error
			}
			if info.IsDir() {
				return nil // Skip directories
			}

			result.CurrentCount++

			// Calculate expected target path in home
			relPath, _ := filepath.Rel(configPath, path)
			targetPath := filepath.Join(home, relPath)

			// Check target status
			targetInfo, err := os.Lstat(targetPath)
			if os.IsNotExist(err) {
				// File exists in dotfiles but no symlink in home
				result.NewFiles = append(result.NewFiles, relPath)
				return nil
			}

			if err != nil {
				return nil // Skip on other errors
			}

			// Check if it's a symlink
			if targetInfo.Mode()&os.ModeSymlink == 0 {
				// If not a symlink, check if it's the same file (handles directory folding)
				sourceInfo, err := os.Stat(path)
				if err == nil && os.SameFile(sourceInfo, targetInfo) {
					// It's the same file (synced via parent directory symlink) - OK
					return nil
				}

				// File exists but is not a symlink - conflict
				result.ConflictFiles = append(result.ConflictFiles, relPath)
				return nil
			}

			// Check if symlink points to the correct location
			linkDest, err := os.Readlink(targetPath)
			if err != nil {
				return nil
			}

			// Resolve to absolute path
			if !filepath.IsAbs(linkDest) {
				linkDest = filepath.Join(filepath.Dir(targetPath), linkDest)
			}
			linkDest = filepath.Clean(linkDest)

			// If symlink points to wrong location, count as conflict
			if linkDest != path {
				result.ConflictFiles = append(result.ConflictFiles, relPath)
			}

			return nil
		})

		if err != nil {
			continue
		}

		// Check for symlinks in home that point to deleted files in dotfiles
		// We can do this by walking the target directories that we know about
		// from the current config structure.
		result.MissingFiles = findOrphanedSymlinks(configPath, home)

		result.HasDrift = len(result.NewFiles) > 0 || len(result.ConflictFiles) > 0 || len(result.MissingFiles) > 0
		results = append(results, result)
	}

	summary := &DriftSummary{
		TotalConfigs: len(results),
		Results:      results,
	}

	if st != nil {
		currentConfigNames := make(map[string]bool)
		for _, c := range allConfigs {
			currentConfigNames[c.Name] = true
		}

		for _, sc := range st.Configs {
			if !currentConfigNames[sc.Name] {
				summary.RemovedConfigs = append(summary.RemovedConfigs, sc.Name)
				summary.DriftedConfigs++
			}
		}
	}

	for _, r := range results {
		if r.HasDrift {
			summary.DriftedConfigs++
			summary.TotalNewFiles += len(r.NewFiles)
		}
	}

	return summary, nil
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
				if strings.HasPrefix(absLinkDest, configPath) {
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

// UpdateSymlinkCounts updates the stored file counts for all configs
func UpdateSymlinkCounts(cfg *config.Config, dotfilesPath string, st *state.State) error {
	allConfigs := cfg.GetAllConfigs()

	for _, configItem := range allConfigs {
		configPath := filepath.Join(dotfilesPath, configItem.Path)

		count, err := countFiles(configPath)
		if err != nil {
			// Config dir doesn't exist - remove from state
			st.RemoveSymlinkCount(configItem.Name)
			continue
		}

		st.SetSymlinkCount(configItem.Name, count)
	}

	return st.Save()
}

// countFiles counts the number of files (not directories) in a directory tree
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
		return 0, err
	}

	return count, nil
}

// GetDriftedConfigs returns only configs that have drift
func GetDriftedConfigs(results []DriftResult) []DriftResult {
	var drifted []DriftResult
	for _, r := range results {
		if r.HasDrift {
			drifted = append(drifted, r)
		}
	}
	return drifted
}

// ConflictFile represents a file that would conflict with stow
type ConflictFile struct {
	ConfigName string // Which config this belongs to
	SourcePath string // Path in dotfiles
	TargetPath string // Path in home that has conflict
	IsDir      bool   // True if the conflict is a directory
}

// DetectConflicts checks for existing files in home that would block stow
func DetectConflicts(cfg *config.Config, dotfilesPath string) ([]ConflictFile, error) {
	var conflicts []ConflictFile
	home := os.Getenv("HOME")

	allConfigs := cfg.GetAllConfigs()
	for _, configItem := range allConfigs {
		configPath := filepath.Join(dotfilesPath, configItem.Path)

		// Check if config directory exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			continue
		}

		// Walk the config directory and check each file
		err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				return nil
			}

			// Calculate expected target path in home
			relPath, _ := filepath.Rel(configPath, path)
			targetPath := filepath.Join(home, relPath)

			// Check if target exists
			targetInfo, err := os.Lstat(targetPath)
			if os.IsNotExist(err) {
				// No conflict - file doesn't exist
				return nil
			}
			if err != nil {
				return nil
			}

			// If it's already a symlink pointing to the right place, no conflict
			if targetInfo.Mode()&os.ModeSymlink != 0 {
				linkDest, err := os.Readlink(targetPath)
				if err == nil {
					if !filepath.IsAbs(linkDest) {
						linkDest = filepath.Join(filepath.Dir(targetPath), linkDest)
					}
					linkDest = filepath.Clean(linkDest)
					if linkDest == path {
						// Already correctly symlinked
						return nil
					}
				}
			}

			// This is a conflict - file exists but isn't the right symlink
			conflicts = append(conflicts, ConflictFile{
				ConfigName: configItem.Name,
				SourcePath: path,
				TargetPath: targetPath,
				IsDir:      targetInfo.IsDir(),
			})

			return nil
		})

		if err != nil {
			continue
		}
	}

	return conflicts, nil
}

// BackupConflict moves a conflicting file to a backup location
func BackupConflict(conflict ConflictFile) error {
	backupPath := conflict.TargetPath + ".g4d-backup"

	// If backup already exists, add timestamp
	if _, err := os.Stat(backupPath); err == nil {
		backupPath = fmt.Sprintf("%s.g4d-backup-%d", conflict.TargetPath, os.Getpid())
	}

	return os.Rename(conflict.TargetPath, backupPath)
}

// RemoveConflict deletes a conflicting file
func RemoveConflict(conflict ConflictFile) error {
	if conflict.IsDir {
		return os.RemoveAll(conflict.TargetPath)
	}
	return os.Remove(conflict.TargetPath)
}
