package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// StateDir is the directory where state files are stored
	StateDir = ".config/go4dot"
	// StateFileName is the name of the state file
	StateFileName = "state.json"
	// StateVersion is the current state file format version
	StateVersion = "1.0"
)

// State represents the installation state of go4dot
type State struct {
	Version       string                   `json:"version"`
	InstalledAt   time.Time                `json:"installed_at"`
	LastUpdate    time.Time                `json:"last_update"`
	DotfilesPath  string                   `json:"dotfiles_path"`
	Platform      PlatformState            `json:"platform"`
	Configs       []ConfigState            `json:"configs"`
	MachineConfig map[string]MachineState  `json:"machine_config"`
	ExternalDeps  map[string]ExternalState `json:"external_deps"`
	SymlinkCounts map[string]int           `json:"symlink_counts,omitempty"` // File count per config for quick drift detection
}

// PlatformState stores detected platform information
type PlatformState struct {
	OS             string `json:"os"`
	Distro         string `json:"distro"`
	DistroVersion  string `json:"distro_version"`
	PackageManager string `json:"package_manager"`
}

// ConfigState tracks an installed config
type ConfigState struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	InstalledAt time.Time `json:"installed_at"`
	IsCore      bool      `json:"is_core"`
}

// MachineState tracks machine-specific configuration
type MachineState struct {
	ConfigPath  string    `json:"config_path"`
	CreatedAt   time.Time `json:"created_at"`
	HasGPG      bool      `json:"has_gpg,omitempty"`
	HasSSH      bool      `json:"has_ssh,omitempty"`
}

// ExternalState tracks an external dependency
type ExternalState struct {
	Installed  bool      `json:"installed"`
	Path       string    `json:"path"`
	LastUpdate time.Time `json:"last_update"`
}

// New creates a new empty state
func New() *State {
	return &State{
		Version:       StateVersion,
		InstalledAt:   time.Now(),
		LastUpdate:    time.Now(),
		MachineConfig: make(map[string]MachineState),
		ExternalDeps:  make(map[string]ExternalState),
	}
}

// GetStatePath returns the full path to the state file
func GetStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, StateDir, StateFileName), nil
}

// GetStateDir returns the state directory path
func GetStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, StateDir), nil
}

// Load reads the state from disk
func Load() (*State, error) {
	statePath, err := GetStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state file exists yet
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	return &state, nil
}

// Save writes the state to disk
func (s *State) Save() error {
	stateDir, err := GetStateDir()
	if err != nil {
		return err
	}

	// Ensure directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	statePath, err := GetStatePath()
	if err != nil {
		return err
	}

	// Update last update time
	s.LastUpdate = time.Now()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Delete removes the state file
func Delete() error {
	statePath, err := GetStatePath()
	if err != nil {
		return err
	}

	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete state file: %w", err)
	}

	return nil
}

// Exists checks if a state file exists
func Exists() bool {
	statePath, err := GetStatePath()
	if err != nil {
		return false
	}

	_, err = os.Stat(statePath)
	return err == nil
}

// AddConfig adds a config to the installed list
func (s *State) AddConfig(name, path string, isCore bool) {
	// Check if already exists
	for i, c := range s.Configs {
		if c.Name == name {
			s.Configs[i].Path = path
			s.Configs[i].IsCore = isCore
			return
		}
	}

	s.Configs = append(s.Configs, ConfigState{
		Name:        name,
		Path:        path,
		InstalledAt: time.Now(),
		IsCore:      isCore,
	})
}

// RemoveConfig removes a config from the installed list
func (s *State) RemoveConfig(name string) {
	for i, c := range s.Configs {
		if c.Name == name {
			s.Configs = append(s.Configs[:i], s.Configs[i+1:]...)
			return
		}
	}
}

// HasConfig checks if a config is installed
func (s *State) HasConfig(name string) bool {
	for _, c := range s.Configs {
		if c.Name == name {
			return true
		}
	}
	return false
}

// GetConfigNames returns list of installed config names
func (s *State) GetConfigNames() []string {
	names := make([]string, len(s.Configs))
	for i, c := range s.Configs {
		names[i] = c.Name
	}
	return names
}

// SetExternalDep updates or adds an external dependency state
func (s *State) SetExternalDep(id string, path string, installed bool) {
	if s.ExternalDeps == nil {
		s.ExternalDeps = make(map[string]ExternalState)
	}
	s.ExternalDeps[id] = ExternalState{
		Installed:  installed,
		Path:       path,
		LastUpdate: time.Now(),
	}
}

// RemoveExternalDep removes an external dependency from state
func (s *State) RemoveExternalDep(id string) {
	delete(s.ExternalDeps, id)
}

// SetMachineConfig updates or adds a machine config state
func (s *State) SetMachineConfig(id string, configPath string, hasGPG, hasSSH bool) {
	if s.MachineConfig == nil {
		s.MachineConfig = make(map[string]MachineState)
	}
	s.MachineConfig[id] = MachineState{
		ConfigPath: configPath,
		CreatedAt:  time.Now(),
		HasGPG:     hasGPG,
		HasSSH:     hasSSH,
	}
}

// RemoveMachineConfig removes a machine config from state
func (s *State) RemoveMachineConfig(id string) {
	delete(s.MachineConfig, id)
}

// SetSymlinkCount updates the file count for a config (for drift detection)
func (s *State) SetSymlinkCount(configName string, count int) {
	if s.SymlinkCounts == nil {
		s.SymlinkCounts = make(map[string]int)
	}
	s.SymlinkCounts[configName] = count
}

// GetSymlinkCount returns the stored file count for a config
func (s *State) GetSymlinkCount(configName string) (int, bool) {
	if s.SymlinkCounts == nil {
		return 0, false
	}
	count, ok := s.SymlinkCounts[configName]
	return count, ok
}

// RemoveSymlinkCount removes the file count for a config
func (s *State) RemoveSymlinkCount(configName string) {
	if s.SymlinkCounts != nil {
		delete(s.SymlinkCounts, configName)
	}
}
