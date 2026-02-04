package stow

import (
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

// DriftDetector defines the interface for detecting symlink drift.
// This interface allows for easier testing by providing a mockable contract.
type DriftDetector interface {
	// Check performs a drift analysis on all configs.
	Check(cfg *config.Config, dotfilesPath string) (*DriftSummary, error)

	// CheckWithHome performs a drift analysis using a specific home directory.
	CheckWithHome(cfg *config.Config, dotfilesPath, home string, st *state.State) (*DriftSummary, error)
}

// DefaultDriftDetector is the production implementation of DriftDetector.
type DefaultDriftDetector struct{}

// Check performs a drift analysis on all configs.
func (d *DefaultDriftDetector) Check(cfg *config.Config, dotfilesPath string) (*DriftSummary, error) {
	return FullDriftCheck(cfg, dotfilesPath)
}

// CheckWithHome performs a drift analysis using a specific home directory.
func (d *DefaultDriftDetector) CheckWithHome(cfg *config.Config, dotfilesPath, home string, st *state.State) (*DriftSummary, error) {
	return FullDriftCheckWithHome(cfg, dotfilesPath, home, st)
}

// NewDriftDetector creates a new DefaultDriftDetector.
func NewDriftDetector() DriftDetector {
	return &DefaultDriftDetector{}
}

// StowManager defines the interface for managing GNU stow operations.
// This interface allows for easier testing by providing a mockable contract.
type StowManager interface {
	// Stow creates symlinks for a config directory.
	Stow(dotfilesPath, configName string, opts StowOptions) error

	// Unstow removes symlinks for a config directory.
	Unstow(dotfilesPath, configName string, opts StowOptions) error

	// Restow refreshes symlinks for a config directory (unstow + stow).
	Restow(dotfilesPath, configName string, opts StowOptions) error

	// StowConfigs stows multiple configurations in sequence.
	StowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult

	// UnstowConfigs unstows multiple configurations in sequence.
	UnstowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult

	// RestowConfigs restows multiple configurations in sequence.
	RestowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult

	// Validate checks if GNU stow is installed and working.
	Validate() error
}

// DefaultStowManager is the production implementation of StowManager.
type DefaultStowManager struct{}

// Stow creates symlinks for a config directory.
func (m *DefaultStowManager) Stow(dotfilesPath, configName string, opts StowOptions) error {
	return Stow(dotfilesPath, configName, opts)
}

// Unstow removes symlinks for a config directory.
func (m *DefaultStowManager) Unstow(dotfilesPath, configName string, opts StowOptions) error {
	return Unstow(dotfilesPath, configName, opts)
}

// Restow refreshes symlinks for a config directory (unstow + stow).
func (m *DefaultStowManager) Restow(dotfilesPath, configName string, opts StowOptions) error {
	return Restow(dotfilesPath, configName, opts)
}

// StowConfigs stows multiple configurations in sequence.
func (m *DefaultStowManager) StowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult {
	return StowConfigs(dotfilesPath, configs, opts)
}

// UnstowConfigs unstows multiple configurations in sequence.
func (m *DefaultStowManager) UnstowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult {
	return UnstowConfigs(dotfilesPath, configs, opts)
}

// RestowConfigs restows multiple configurations in sequence.
func (m *DefaultStowManager) RestowConfigs(dotfilesPath string, configs []config.ConfigItem, opts StowOptions) *StowResult {
	return RestowConfigs(dotfilesPath, configs, opts)
}

// Validate checks if GNU stow is installed and working.
func (m *DefaultStowManager) Validate() error {
	return ValidateStow()
}

// NewStowManager creates a new DefaultStowManager.
func NewStowManager() StowManager {
	return &DefaultStowManager{}
}
