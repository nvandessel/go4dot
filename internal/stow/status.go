package stow

import (
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
)

// FileStatus represents the link status of a single file
type FileStatus struct {
	RelPath  string // Relative path from config directory
	IsLinked bool   // True if correctly symlinked
	Issue    string // Description of issue if not linked
}

// ConfigLinkStatus represents the overall link status for a config
type ConfigLinkStatus struct {
	ConfigName  string
	ConfigPath  string
	LinkedCount int
	TotalCount  int
	Files       []FileStatus
}

// IsFullyLinked returns true if all files are correctly symlinked
func (s *ConfigLinkStatus) IsFullyLinked() bool {
	return s.LinkedCount == s.TotalCount && s.TotalCount > 0
}

// IsPartiallyLinked returns true if some but not all files are symlinked
func (s *ConfigLinkStatus) IsPartiallyLinked() bool {
	return s.LinkedCount > 0 && s.LinkedCount < s.TotalCount
}

// IsNotLinked returns true if no files are symlinked
func (s *ConfigLinkStatus) IsNotLinked() bool {
	return s.LinkedCount == 0
}

// GetLinkedFiles returns only files that are correctly linked
func (s *ConfigLinkStatus) GetLinkedFiles() []FileStatus {
	var linked []FileStatus
	for _, f := range s.Files {
		if f.IsLinked {
			linked = append(linked, f)
		}
	}
	return linked
}

// GetMissingFiles returns only files that are not linked
func (s *ConfigLinkStatus) GetMissingFiles() []FileStatus {
	var missing []FileStatus
	for _, f := range s.Files {
		if !f.IsLinked {
			missing = append(missing, f)
		}
	}
	return missing
}

// GetAllConfigLinkStatus returns link status for all configs
func GetAllConfigLinkStatus(cfg *config.Config, dotfilesPath string) (map[string]*ConfigLinkStatus, error) {
	result := make(map[string]*ConfigLinkStatus)
	home := os.Getenv("HOME")

	allConfigs := cfg.GetAllConfigs()
	for _, configItem := range allConfigs {
		status, err := getConfigLinkStatusInternal(configItem, dotfilesPath, home)
		if err != nil {
			continue
		}
		result[configItem.Name] = status
	}

	return result, nil
}

// getConfigLinkStatusInternal checks the link status of a single config
func getConfigLinkStatusInternal(configItem config.ConfigItem, dotfilesPath, home string) (*ConfigLinkStatus, error) {
	configPath := filepath.Join(dotfilesPath, configItem.Path)

	status := &ConfigLinkStatus{
		ConfigName: configItem.Name,
		ConfigPath: configItem.Path,
	}

	// Check if config directory exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return status, nil
	}

	// Walk the config directory and check each file
	err := filepath.Walk(configPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip on error
		}
		if info.IsDir() {
			return nil // Skip directories
		}

		status.TotalCount++

		// Calculate expected target path in home
		relPath, _ := filepath.Rel(configPath, path)
		targetPath := filepath.Join(home, relPath)

		fileStatus := FileStatus{
			RelPath: relPath,
		}

		// Check if the symlink exists and is correct
		if checkLinkStatus(path, targetPath, &fileStatus) {
			fileStatus.IsLinked = true
			status.LinkedCount++
		}

		status.Files = append(status.Files, fileStatus)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return status, nil
}

// checkLinkStatus checks if targetPath is correctly linked to sourcePath
// and populates the issue field if not
func checkLinkStatus(sourcePath, targetPath string, fileStatus *FileStatus) bool {
	targetInfo, err := os.Lstat(targetPath)
	if os.IsNotExist(err) {
		fileStatus.Issue = "not linked"
		return false
	}
	if err != nil {
		fileStatus.Issue = "error checking"
		return false
	}

	// Check if it's a symlink
	if targetInfo.Mode()&os.ModeSymlink == 0 {
		// Not a symlink - check if it's the same file (handles directory folding)
		sourceInfo, err := os.Stat(sourcePath)
		if err != nil {
			fileStatus.Issue = "source error"
			return false
		}
		if os.SameFile(sourceInfo, targetInfo) {
			// Linked via directory fold
			return true
		}
		fileStatus.Issue = "file exists (conflict)"
		return false
	}

	// It's a symlink - check if it points to the correct location
	linkDest, err := os.Readlink(targetPath)
	if err != nil {
		fileStatus.Issue = "cannot read link"
		return false
	}

	// Resolve to absolute path (stow creates relative symlinks)
	if !filepath.IsAbs(linkDest) {
		linkDest = filepath.Join(filepath.Dir(targetPath), linkDest)
	}
	linkDest = filepath.Clean(linkDest)

	if linkDest != sourcePath {
		fileStatus.Issue = "points elsewhere"
		return false
	}

	return true
}
