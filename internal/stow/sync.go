package stow

import (
	"fmt"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
)

// SyncAll restows all configs and updates state.
// It handles conflict detection and resolution if interactive.
func SyncAll(dotfilesPath string, cfg *config.Config, st *state.State, interactive bool, opts StowOptions) (*StowResult, error) {
	if interactive {
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(0, 0, "Checking for conflicts...")
		}

		conflicts, err := DetectConflicts(cfg, dotfilesPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check conflicts: %w", err)
		}

		if len(conflicts) > 0 {
			if !ResolveConflicts(conflicts) {
				return nil, fmt.Errorf("sync cancelled due to unresolved conflicts")
			}
		}
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, 0, "Syncing all configs...")
	}

	allConfigs := cfg.GetAllConfigs()
	result := RestowConfigs(dotfilesPath, allConfigs, opts)

	// Update symlink counts in state
	if st != nil {
		if err := UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
			return result, fmt.Errorf("failed to update symlink counts: %w", err)
		}
	}

	return result, nil
}

// SyncSingle restows a single config and updates state.
func SyncSingle(dotfilesPath string, configName string, cfg *config.Config, st *state.State, opts StowOptions) error {
	// Find the config item
	var configItem *config.ConfigItem
	for _, c := range cfg.GetAllConfigs() {
		if c.Name == configName {
			configItem = &c
			break
		}
	}

	if configItem == nil {
		return fmt.Errorf("config '%s' not found", configName)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, 0, fmt.Sprintf("Syncing %s...", configName))
	}

	err := Restow(dotfilesPath, configItem.Path, opts)
	if err != nil {
		return err
	}

	// Update symlink count for this config
	if st != nil {
		if err := UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
			return fmt.Errorf("failed to update symlink counts: %w", err)
		}
	}

	return nil
}

// SyncResultSummary returns a human-readable summary of the sync result
func SyncResultSummary(result *StowResult) string {
	if len(result.Failed) > 0 {
		return fmt.Sprintf("Failed to sync %d config(s)", len(result.Failed))
	}
	return fmt.Sprintf("Synced %d config(s)", len(result.Success))
}
