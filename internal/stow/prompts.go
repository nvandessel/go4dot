package stow

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/print"
)

// ResolveConflicts prompts the user to handle conflicting files.
// Returns true if conflicts were resolved, false if cancelled.
func ResolveConflicts(conflicts []ConflictFile) bool {
	fmt.Printf("\n  Found %d conflicting file(s) that would be overwritten:\n\n", len(conflicts))

	// Group by config
	byConfig := make(map[string][]ConflictFile)
	for _, c := range conflicts {
		byConfig[c.ConfigName] = append(byConfig[c.ConfigName], c)
	}

	for configName, files := range byConfig {
		fmt.Printf("  %s:\n", configName)
		for i, f := range files {
			if i >= 5 {
				fmt.Printf("    ... and %d more\n", len(files)-5)
				break
			}
			// Show just the filename relative to home
			home := os.Getenv("HOME")
			relPath, _ := filepath.Rel(home, f.TargetPath)
			fmt.Printf("    ~/%s\n", relPath)
		}
	}

	fmt.Println()

	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("How would you like to handle these conflicts?").
				Options(
					huh.NewOption("Backup existing files (rename to .g4d-backup)", "backup"),
					huh.NewOption("Delete existing files (use dotfiles version)", "delete"),
					huh.NewOption("Cancel sync", "cancel"),
				).
				Value(&action),
		),
	)

	if err := form.Run(); err != nil {
		return false
	}

	if action == "cancel" {
		return false
	}

	// Process conflicts
	for _, conflict := range conflicts {
		var err error
		if action == "backup" {
			err = BackupConflict(conflict)
			if err == nil {
				home := os.Getenv("HOME")
				relPath, _ := filepath.Rel(home, conflict.TargetPath)
				fmt.Printf("  Backed up ~/%s\n", relPath)
			}
		} else {
			err = RemoveConflict(conflict)
			if err == nil {
				home := os.Getenv("HOME")
				relPath, _ := filepath.Rel(home, conflict.TargetPath)
				fmt.Printf("  Removed ~/%s\n", relPath)
			}
		}

		if err != nil {
			print.Error("Failed to handle %s: %v", conflict.TargetPath, err)
			return false
		}
	}

	fmt.Println()
	return true
}
