package dashboard

import (
	"fmt"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
)

// SyncOptions configures the sync operation
type SyncOptions struct {
	Force       bool // Force restow even if no drift detected
	Interactive bool // Enable interactive conflict resolution
}

// SyncResult holds the result of a sync operation
type SyncResult struct {
	Success []string
	Failed  []stow.StowError
	Skipped []string
}

// HasErrors returns true if any errors occurred
func (r *SyncResult) HasErrors() bool {
	return len(r.Failed) > 0
}

// Summary returns a summary string
func (r *SyncResult) Summary() string {
	if len(r.Success) == 0 && len(r.Failed) == 0 && len(r.Skipped) == 0 {
		return "No configs to sync"
	}

	summary := fmt.Sprintf("%d synced", len(r.Success))
	if len(r.Failed) > 0 {
		summary += fmt.Sprintf(", %d failed", len(r.Failed))
	}
	if len(r.Skipped) > 0 {
		summary += fmt.Sprintf(", %d skipped", len(r.Skipped))
	}
	return summary
}

// loadOrCreateState loads existing state or creates a new one if unavailable
func loadOrCreateState() *state.State {
	st, err := state.Load()
	if err != nil || st == nil {
		return state.New()
	}
	return st
}

// RunSyncAllOperation runs a sync all operation within the dashboard
func RunSyncAllOperation(runner *OperationRunner, cfg *config.Config, dotfilesPath string, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Step 0: Check symlinks
	runner.Progress(0, "Analyzing symlink status...")

	st := loadOrCreateState()

	if _, err := stow.FullDriftCheck(cfg, dotfilesPath); err != nil {
		runner.Log("warning", fmt.Sprintf("Drift check failed: %v", err))
	}

	runner.StepComplete(0, StepSuccess, fmt.Sprintf("%d configs analyzed", len(cfg.GetAllConfigs())))

	// Step 1: Sync configs
	runner.Progress(1, fmt.Sprintf("Syncing %d configs...", len(cfg.GetAllConfigs())))

	stowOpts := stow.StowOptions{
		Force: opts.Force,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	syncResult, err := stow.SyncAll(dotfilesPath, cfg, st, opts.Interactive, stowOpts)
	if err != nil {
		runner.StepComplete(1, StepError, err.Error())
		return nil, fmt.Errorf("sync all failed: %w", err)
	}

	result.Success = syncResult.Success
	result.Failed = syncResult.Failed
	result.Skipped = syncResult.Skipped

	if len(syncResult.Failed) > 0 {
		runner.StepComplete(1, StepWarning, fmt.Sprintf("%d synced, %d failed", len(syncResult.Success), len(syncResult.Failed)))
		for _, f := range syncResult.Failed {
			runner.Log("error", fmt.Sprintf("Failed: %s - %v", f.ConfigName, f.Error))
		}
	} else {
		runner.StepComplete(1, StepSuccess, fmt.Sprintf("%d configs synced", len(syncResult.Success)))
	}

	// Step 2: Update state
	runner.Progress(2, "Updating state...")

	if err := stow.UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to update symlink counts: %v", err))
	}

	if err := st.Save(); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to save state: %v", err))
	}

	runner.StepComplete(2, StepSuccess, "State updated")

	// Report completion
	if result.HasErrors() {
		runner.Done(false, result.Summary(), collectSyncErrors(result.Failed))
	} else {
		runner.Done(true, result.Summary(), nil)
	}

	return result, nil
}

// collectSyncErrors combines multiple sync errors into one
func collectSyncErrors(failed []stow.StowError) error {
	if len(failed) == 0 {
		return nil
	}
	if len(failed) == 1 {
		return fmt.Errorf("sync failed for %s: %w", failed[0].ConfigName, failed[0].Error)
	}
	return fmt.Errorf("sync failed for %d configs; first error: %s: %w", len(failed), failed[0].ConfigName, failed[0].Error)
}

// collectUpdateErrors combines multiple update errors into one
func collectUpdateErrors(failed []deps.ExternalError) error {
	if len(failed) == 0 {
		return nil
	}
	if len(failed) == 1 {
		return fmt.Errorf("update failed for %s: %w", failed[0].Dep.Name, failed[0].Error)
	}
	return fmt.Errorf("update failed for %d dependencies; first error: %s: %w", len(failed), failed[0].Dep.Name, failed[0].Error)
}

// RunSyncSingleOperation runs a sync operation for a single config
func RunSyncSingleOperation(runner *OperationRunner, cfg *config.Config, dotfilesPath string, configName string, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Step 0: Check symlinks
	runner.Progress(0, fmt.Sprintf("Checking %s...", configName))

	st := loadOrCreateState()

	runner.StepComplete(0, StepSuccess, "Status checked")

	// Step 1: Sync config
	runner.Progress(1, fmt.Sprintf("Syncing %s...", configName))

	stowOpts := stow.StowOptions{
		Force: opts.Force,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	err := stow.SyncSingle(dotfilesPath, configName, cfg, st, stowOpts)
	if err != nil {
		runner.StepComplete(1, StepError, err.Error())
		result.Failed = append(result.Failed, stow.StowError{ConfigName: configName, Error: err})
	} else {
		runner.StepComplete(1, StepSuccess, fmt.Sprintf("%s synced", configName))
		result.Success = append(result.Success, configName)
	}

	// Step 2: Update state
	runner.Progress(2, "Updating state...")

	if err := stow.UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to update symlink counts: %v", err))
	}

	if err := st.Save(); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to save state: %v", err))
	}

	runner.StepComplete(2, StepSuccess, "State updated")

	// Report completion
	if result.HasErrors() {
		runner.Done(false, result.Summary(), collectSyncErrors(result.Failed))
	} else {
		runner.Done(true, result.Summary(), nil)
	}

	return result, nil
}

// RunBulkSyncOperation runs a sync operation for multiple configs
func RunBulkSyncOperation(runner *OperationRunner, cfg *config.Config, dotfilesPath string, configNames []string, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{}

	// Step 0: Check symlinks
	runner.Progress(0, fmt.Sprintf("Checking %d configs...", len(configNames)))

	st := loadOrCreateState()

	runner.StepComplete(0, StepSuccess, fmt.Sprintf("%d configs to sync", len(configNames)))

	// Step 1: Sync configs
	runner.Progress(1, fmt.Sprintf("Syncing %d configs...", len(configNames)))

	stowOpts := stow.StowOptions{
		Force: opts.Force,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	for i, name := range configNames {
		runner.Log("info", fmt.Sprintf("[%d/%d] Syncing %s...", i+1, len(configNames), name))

		err := stow.SyncSingle(dotfilesPath, name, cfg, st, stowOpts)
		if err != nil {
			result.Failed = append(result.Failed, stow.StowError{ConfigName: name, Error: err})
			runner.Log("error", fmt.Sprintf("Failed: %s - %v", name, err))
		} else {
			result.Success = append(result.Success, name)
			runner.Log("success", fmt.Sprintf("Synced: %s", name))
		}
	}

	if len(result.Failed) > 0 {
		runner.StepComplete(1, StepWarning, fmt.Sprintf("%d synced, %d failed", len(result.Success), len(result.Failed)))
	} else {
		runner.StepComplete(1, StepSuccess, fmt.Sprintf("%d configs synced", len(result.Success)))
	}

	// Step 2: Update state
	runner.Progress(2, "Updating state...")

	if err := stow.UpdateSymlinkCounts(cfg, dotfilesPath, st); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to update symlink counts: %v", err))
	}

	if err := st.Save(); err != nil {
		runner.Log("warning", fmt.Sprintf("Failed to save state: %v", err))
	}

	runner.StepComplete(2, StepSuccess, "State updated")

	// Report completion
	if result.HasErrors() {
		runner.Done(false, result.Summary(), collectSyncErrors(result.Failed))
	} else {
		runner.Done(true, result.Summary(), nil)
	}

	return result, nil
}

// UpdateOptions configures the update operation
type UpdateOptions struct {
	UpdateExternal bool
}

// UpdateResult holds the result of an update operation
type UpdateResult struct {
	Updated []string
	Failed  []string
	Skipped []string
}

// Summary returns a summary string
func (r *UpdateResult) Summary() string {
	if len(r.Updated) == 0 && len(r.Failed) == 0 && len(r.Skipped) == 0 {
		return "No updates needed"
	}

	summary := fmt.Sprintf("%d updated", len(r.Updated))
	if len(r.Failed) > 0 {
		summary += fmt.Sprintf(", %d failed", len(r.Failed))
	}
	if len(r.Skipped) > 0 {
		summary += fmt.Sprintf(", %d skipped", len(r.Skipped))
	}
	return summary
}

// RunUpdateOperation runs an update operation for external dependencies
func RunUpdateOperation(runner *OperationRunner, cfg *config.Config, dotfilesPath string, opts UpdateOptions) (*UpdateResult, error) {
	result := &UpdateResult{}

	// Check if external updates are disabled
	if !opts.UpdateExternal {
		runner.StepComplete(0, StepSkipped, "External updates disabled")
		runner.StepComplete(1, StepSkipped, "Skipped by configuration")
		runner.Done(true, "External updates skipped", nil)
		return result, nil
	}

	if len(cfg.External) == 0 {
		runner.StepComplete(0, StepSuccess, "No external dependencies")
		runner.StepComplete(1, StepSkipped, "Nothing to update")
		runner.Done(true, "No external dependencies to update", nil)
		return result, nil
	}

	// Step 0: Check external dependencies
	runner.Progress(0, fmt.Sprintf("Checking %d external dependencies...", len(cfg.External)))

	p, err := platform.Detect()
	if err != nil {
		wrappedErr := fmt.Errorf("platform detection failed: %w", err)
		runner.StepComplete(0, StepError, wrappedErr.Error())
		return nil, wrappedErr
	}

	runner.StepComplete(0, StepSuccess, fmt.Sprintf("%d dependencies found", len(cfg.External)))

	// Step 1: Update repositories
	runner.Progress(1, "Updating repositories...")

	extOpts := deps.ExternalOptions{
		Update:   true, // Enable update mode
		RepoRoot: dotfilesPath,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	// Use CloneExternal with Update: true to update existing repos
	updateResult, err := deps.CloneExternal(cfg, p, extOpts)
	if err != nil {
		wrappedErr := fmt.Errorf("clone external repos: %w", err)
		runner.StepComplete(1, StepError, wrappedErr.Error())
		return nil, wrappedErr
	}

	for _, ext := range updateResult.Updated {
		result.Updated = append(result.Updated, ext.Name)
	}
	for _, ext := range updateResult.Failed {
		result.Failed = append(result.Failed, ext.Dep.Name)
		runner.Log("error", fmt.Sprintf("Failed: %s - %v", ext.Dep.Name, ext.Error))
	}
	for _, ext := range updateResult.Skipped {
		result.Skipped = append(result.Skipped, ext.Dep.Name)
	}

	if len(result.Failed) > 0 {
		runner.StepComplete(1, StepWarning, fmt.Sprintf("%d updated, %d failed", len(result.Updated), len(result.Failed)))
	} else {
		runner.StepComplete(1, StepSuccess, fmt.Sprintf("%d repositories updated", len(result.Updated)))
	}

	// Report completion
	if len(updateResult.Failed) > 0 {
		runner.Done(false, result.Summary(), collectUpdateErrors(updateResult.Failed))
	} else {
		runner.Done(true, result.Summary(), nil)
	}

	return result, nil
}
