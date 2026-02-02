package dashboard

import (
	"fmt"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
)

// ExternalSingleOptions configures a single external operation
type ExternalSingleOptions struct {
	Update bool // Update if exists, otherwise clone
}

// ExternalSingleResult holds the result of a single external operation
type ExternalSingleResult struct {
	Name    string
	Action  string // "cloned", "updated", "skipped", "failed"
	Message string
	Error   error
}

// Summary returns a summary string
func (r *ExternalSingleResult) Summary() string {
	if r.Error != nil {
		return fmt.Sprintf("Failed: %s", r.Message)
	}
	return fmt.Sprintf("%s: %s", r.Action, r.Name)
}

// RunExternalSingleOperation runs a clone/update operation for a single external dependency
func RunExternalSingleOperation(runner *OperationRunner, cfg *config.Config, dotfilesPath string, extID string, opts ExternalSingleOptions) (*ExternalSingleResult, error) {
	result := &ExternalSingleResult{}

	// Step 0: Check status
	runner.Progress(0, "Checking dependency status...")

	p, err := platform.Detect()
	if err != nil {
		runner.StepComplete(0, StepError, err.Error())
		result.Error = fmt.Errorf("failed to detect platform: %w", err)
		return result, result.Error
	}

	// Find the external dependency
	var ext *config.ExternalDep
	for i := range cfg.External {
		if cfg.External[i].ID == extID {
			ext = &cfg.External[i]
			break
		}
	}

	if ext == nil {
		runner.StepComplete(0, StepError, "Not found")
		result.Error = fmt.Errorf("external dependency '%s' not found", extID)
		return result, result.Error
	}

	result.Name = ext.Name
	if result.Name == "" {
		result.Name = ext.ID
	}

	// Check current status
	status := deps.CheckExternalStatus(cfg, p, dotfilesPath)
	var extStatus *deps.ExternalStatus
	for i := range status {
		if status[i].Dep.ID == extID {
			extStatus = &status[i]
			break
		}
	}

	if extStatus == nil {
		runner.StepComplete(0, StepError, "Status check failed")
		result.Error = fmt.Errorf("could not check status for '%s'", extID)
		return result, result.Error
	}

	if extStatus.Status == "skipped" {
		runner.StepComplete(0, StepSkipped, extStatus.Reason)
		runner.StepComplete(1, StepSkipped, "Condition not met")
		result.Action = "skipped"
		result.Message = extStatus.Reason
		return result, nil
	}

	// Fail fast on error status
	if extStatus.Status == "error" {
		runner.StepComplete(0, StepError, extStatus.Reason)
		result.Action = "failed"
		result.Message = extStatus.Reason
		result.Error = fmt.Errorf("external status error for '%s': %s", extID, extStatus.Reason)
		return result, result.Error
	}

	runner.StepComplete(0, StepSuccess, fmt.Sprintf("Status: %s", extStatus.Status))

	// Step 1: Clone or update
	if extStatus.Status == "installed" && opts.Update {
		runner.Progress(1, fmt.Sprintf("Updating %s...", result.Name))
	} else if extStatus.Status == "missing" {
		runner.Progress(1, fmt.Sprintf("Cloning %s...", result.Name))
	} else if extStatus.Status == "installed" {
		// Already installed and not updating
		runner.StepComplete(1, StepSuccess, "Already installed")
		result.Action = "skipped"
		result.Message = "Already installed"
		return result, nil
	}

	// Perform the operation
	extOpts := deps.ExternalOptions{
		Update:   opts.Update,
		RepoRoot: dotfilesPath,
		ProgressFunc: func(current, total int, msg string) {
			runner.Log("info", msg)
		},
	}

	err = deps.CloneSingle(cfg, p, extID, extOpts)
	if err != nil {
		runner.StepComplete(1, StepError, err.Error())
		result.Action = "failed"
		result.Message = err.Error()
		result.Error = err
		return result, err
	}

	if extStatus.Status == "installed" {
		result.Action = "updated"
		runner.StepComplete(1, StepSuccess, "Updated")
	} else {
		result.Action = "cloned"
		runner.StepComplete(1, StepSuccess, "Cloned")
	}

	return result, nil
}
