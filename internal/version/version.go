package version

import "sync"

// toolVersion holds the current tool version, set at startup by the main package.
// Protected by a mutex for safe concurrent access.
var (
	toolVersionMu sync.RWMutex
	toolVersion   string
)

// SetToolVersion sets the current tool version.
// This should be called once at startup from the main package.
func SetToolVersion(v string) {
	toolVersionMu.Lock()
	defer toolVersionMu.Unlock()
	toolVersion = v
}

// GetToolVersion returns the current tool version.
func GetToolVersion() string {
	toolVersionMu.RLock()
	defer toolVersionMu.RUnlock()
	return toolVersion
}
