package dashboard

import (
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/stow"
)

// ConflictResolutionChoice represents the user's choice for handling conflicts
type ConflictResolutionChoice int

const (
	// ConflictChoiceBackup backs up conflicting files with .g4d-backup suffix
	ConflictChoiceBackup ConflictResolutionChoice = iota
	// ConflictChoiceDelete deletes conflicting files
	ConflictChoiceDelete
	// ConflictChoiceCancel cancels the operation
	ConflictChoiceCancel
)

// ConflictResolvedMsg is sent when the conflict resolution modal closes
type ConflictResolvedMsg struct {
	Choice   ConflictResolutionChoice
	Resolved bool // true if conflicts were handled, false if cancelled
	Error    error
}

// ResolveConflictsAction executes the backup or delete action based on user choice
func ResolveConflictsAction(conflicts []stow.ConflictFile, choice ConflictResolutionChoice) error {
	switch choice {
	case ConflictChoiceCancel:
		return nil
	case ConflictChoiceBackup:
		for _, conflict := range conflicts {
			if err := stow.BackupConflict(conflict); err != nil {
				return err
			}
		}
	case ConflictChoiceDelete:
		for _, conflict := range conflicts {
			if err := stow.RemoveConflict(conflict); err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckForConflicts detects files that would conflict with stow operations.
// If configNames is empty, checks all configs. Otherwise, filters to specified configs.
func CheckForConflicts(cfg *config.Config, dotfilesPath string, configNames []string) ([]stow.ConflictFile, error) {
	allConflicts, err := stow.DetectConflicts(cfg, dotfilesPath)
	if err != nil {
		return nil, err
	}

	if len(configNames) == 0 {
		return allConflicts, nil
	}

	// Filter to specified configs only
	configSet := make(map[string]bool)
	for _, name := range configNames {
		configSet[name] = true
	}

	var filtered []stow.ConflictFile
	for _, c := range allConflicts {
		if configSet[c.ConfigName] {
			filtered = append(filtered, c)
		}
	}

	return filtered, nil
}

// GroupConflictsByConfig groups conflicts by their config name for display
func GroupConflictsByConfig(conflicts []stow.ConflictFile) map[string][]stow.ConflictFile {
	byConfig := make(map[string][]stow.ConflictFile)
	for _, c := range conflicts {
		byConfig[c.ConfigName] = append(byConfig[c.ConfigName], c)
	}
	return byConfig
}
