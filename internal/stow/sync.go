package stow

import (
	"fmt"
	"os"
	"path/filepath"

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

	// Unstow removed configs
	if st != nil {
		home := os.Getenv("HOME")
		summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, home, st)
		if err == nil {
			// Unstow removed configs
			if len(summary.RemovedConfigs) > 0 {
				for _, name := range summary.RemovedConfigs {
					if opts.ProgressFunc != nil {
						opts.ProgressFunc(0, 0, fmt.Sprintf("Unstowing removed config %s...", name))
					}
					var removedPath string
					for _, sc := range st.Configs {
						if sc.Name == name {
							removedPath = sc.Path
							break
						}
					}

					if removedPath != "" {
						err := Unstow(dotfilesPath, removedPath, opts)
						if err == nil {
							st.RemoveConfig(name)
							st.RemoveSymlinkCount(name)
						} else if opts.ProgressFunc != nil {
							opts.ProgressFunc(0, 0, fmt.Sprintf("Warning: failed to unstow removed config %s: %v", name, err))
						}
					}
				}
			}

			// Clean up orphaned symlinks for active configs
			for _, res := range summary.Results {
				if len(res.MissingFiles) > 0 {
					for _, relPath := range res.MissingFiles {
						if opts.ProgressFunc != nil {
							opts.ProgressFunc(0, 0, fmt.Sprintf("Removing orphaned symlink %s...", relPath))
						}
						if !opts.DryRun {
							targetPath := filepath.Join(home, relPath)
							if err := os.Remove(targetPath); err != nil {
								if opts.ProgressFunc != nil {
									opts.ProgressFunc(0, 0, fmt.Sprintf("Warning: failed to remove orphaned symlink %s: %v", relPath, err))
								}
							}
						}
					}
				}
			}
		}
	}

	// Update state
	if st != nil {
		// Update configs in state
		for _, cfgItem := range allConfigs {
			st.AddConfig(cfgItem.Name, cfgItem.Path, true) // Assume core if in main config
		}

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

	// Clean up orphaned symlinks for this config
	home := os.Getenv("HOME")
	summary, err := FullDriftCheckWithHome(cfg, dotfilesPath, home, st)
	if err != nil {
		if opts.ProgressFunc != nil {
			opts.ProgressFunc(0, 0, fmt.Sprintf("Warning: drift check failed: %v", err))
		}
	} else {
		if res := summary.ResultByName(configName); res != nil && len(res.MissingFiles) > 0 {
			for _, relPath := range res.MissingFiles {
				if opts.ProgressFunc != nil {
					opts.ProgressFunc(0, 0, fmt.Sprintf("Removing orphaned symlink %s...", relPath))
				}
				if !opts.DryRun {
					targetPath := filepath.Join(home, relPath)
					if err := os.Remove(targetPath); err != nil {
						if opts.ProgressFunc != nil {
							opts.ProgressFunc(0, 0, fmt.Sprintf("Warning: failed to remove orphaned symlink %s: %v", relPath, err))
						}
					}
				}
			}
		}
	}

	// Update state for this config
	if st != nil {
		st.AddConfig(configItem.Name, configItem.Path, true)
		if err := UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
			return fmt.Errorf("failed to update symlink counts: %w", err)
		}
	}

	return nil
}

// SyncResultSummary returns a human-readable summary of the sync result.
// It displays the number of successfully synced configs or a failure message.
func SyncResultSummary(result *StowResult) string {
	if len(result.Failed) > 0 {
		return fmt.Sprintf("Failed to sync %d config(s)", len(result.Failed))
	}
	return fmt.Sprintf("Synced %d config(s)", len(result.Success))
}
