package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/nvandessel/go4dot/internal/ui/dashboard"
	"github.com/nvandessel/go4dot/internal/version"
	"github.com/spf13/cobra"
)

// runInteractive starts the interactive dashboard
func runInteractive(cmd *cobra.Command, args []string) {
	// If arguments provided, don't run interactive
	if len(args) > 0 {
		return
	}

	// If non-interactive mode, don't run interactive UI
	if !ui.IsInteractive() {
		// In non-interactive mode, just run doctor check and exit
		doctorCmd.Run(doctorCmd, nil)
		return
	}

	// Check for updates in background
	updateMsgChan := make(chan string, 1)
	go func() {
		res, err := version.CheckForUpdates(Version)
		if err == nil && res != nil && res.IsOutdated {
			updateMsgChan <- fmt.Sprintf("Update: %s -> %s", res.CurrentVersion, res.LatestVersion)
		} else {
			updateMsgChan <- ""
		}
	}()

	// Get update message (non-blocking)
	updateMsg := ""

	// Detect platform once
	p, _ := platform.Detect()

	// Main application loop - stays in the app until user quits
	for {
		// Check for update message
		select {
		case msg := <-updateMsgChan:
			if msg != "" {
				updateMsg = msg
			}
		default:
		}

		// Try to load config to determine context
		cfg, configPath, err := config.LoadFromDiscovery()
		hasConfig := err == nil && cfg != nil

		var result *dashboard.Result

		if !hasConfig {
			// No config found - show polished init prompt
			result = runNoConfigPrompt()
			err = nil
		} else {
			// Config exists - show health dashboard
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()
			if st == nil {
				st = state.New()
			}

			var driftSummary *stow.DriftSummary
			hasBaseline := false
			if st != nil {
				driftSummary, _ = stow.FullDriftCheck(cfg, dotfilesPath)
				// Check if we have any stored symlink counts (indicates prior sync)
				hasBaseline = len(st.SymlinkCounts) > 0
			}

			allConfigs := cfg.GetAllConfigs()
			result, err = dashboard.Run(p, driftSummary, allConfigs, dotfilesPath, updateMsg, hasBaseline)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if result == nil {
			return
		}

		// Handle the action
		shouldExit := handleAction(result, cfg, configPath)
		if shouldExit {
			fmt.Println("Bye!")
			return
		}
	}
}

// runNoConfigPrompt shows a polished prompt when no config exists
func runNoConfigPrompt() *dashboard.Result {
	cwd, _ := os.Getwd()

	ui.PrintBanner(Version)
	fmt.Printf("\n  No .go4dot.yaml found in %s\n\n", filepath.Base(cwd))

	var initHere bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to initialize go4dot here?").
				Description("This will scan for configs and create a .go4dot.yaml file").
				Affirmative("Yes, set up go4dot").
				Negative("No, quit").
				Value(&initHere),
		),
	)

	if err := form.Run(); err != nil {
		return &dashboard.Result{Action: dashboard.ActionQuit}
	}

	if initHere {
		return &dashboard.Result{Action: dashboard.ActionInit}
	}
	return &dashboard.Result{Action: dashboard.ActionQuit}
}

// handleAction processes the user's action and returns true if we should exit
func handleAction(result *dashboard.Result, cfg *config.Config, configPath string) bool {
	switch result.Action {
	case dashboard.ActionQuit:
		return true

	case dashboard.ActionInit:
		initCmd.Run(initCmd, nil)
		waitForEnter()

	case dashboard.ActionSync:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()
			if st == nil {
				st = state.New()
			}
			runStowRefresh(dotfilesPath, cfg, st)
			waitForEnter()
		}

	case dashboard.ActionSyncConfig:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()
			if st == nil {
				st = state.New()
			}
			runStowSingle(dotfilesPath, result.ConfigName, cfg, st)
			waitForEnter()
		}

	case dashboard.ActionDoctor:
		doctorCmd.Run(doctorCmd, nil)
		waitForEnter()

	case dashboard.ActionInstall:
		installCmd.Run(installCmd, nil)
		waitForEnter()
	}

	return false
}

// runStowRefresh restows all configs
func runStowRefresh(dotfilesPath string, cfg *config.Config, st *state.State) {
	fmt.Println("\n  Checking for conflicts...")

	// Check for conflicts first
	conflicts, err := stow.DetectConflicts(cfg, dotfilesPath)
	if err != nil {
		ui.Error("Failed to check conflicts: %v", err)
		return
	}

	if len(conflicts) > 0 {
		// Show conflicts and ask how to handle
		if !resolveConflicts(conflicts) {
			fmt.Println("  Sync cancelled.")
			return
		}
	}

	fmt.Println("  Syncing all configs...")

	allConfigs := cfg.GetAllConfigs()
	result := stow.RestowConfigs(dotfilesPath, allConfigs, stow.StowOptions{
		ProgressFunc: func(msg string) {
			fmt.Printf("  %s\n", msg)
		},
	})

	// Update symlink counts in state
	if st != nil {
		stow.UpdateSymlinkCounts(cfg, dotfilesPath, st)
	}

	if len(result.Failed) > 0 {
		ui.Error("Failed to sync %d config(s)", len(result.Failed))
	} else {
		ui.Success("Synced %d config(s)", len(result.Success))
	}
}

// resolveConflicts prompts the user to handle conflicting files
func resolveConflicts(conflicts []stow.ConflictFile) bool {
	fmt.Printf("\n  Found %d conflicting file(s) that would be overwritten:\n\n", len(conflicts))

	// Group by config
	byConfig := make(map[string][]stow.ConflictFile)
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
			err = stow.BackupConflict(conflict)
			if err == nil {
				home := os.Getenv("HOME")
				relPath, _ := filepath.Rel(home, conflict.TargetPath)
				fmt.Printf("  Backed up ~/%s\n", relPath)
			}
		} else {
			err = stow.RemoveConflict(conflict)
			if err == nil {
				home := os.Getenv("HOME")
				relPath, _ := filepath.Rel(home, conflict.TargetPath)
				fmt.Printf("  Removed ~/%s\n", relPath)
			}
		}

		if err != nil {
			ui.Error("Failed to handle %s: %v", conflict.TargetPath, err)
			return false
		}
	}

	fmt.Println()
	return true
}

// runStowSingle restows a single config
func runStowSingle(dotfilesPath string, configName string, cfg *config.Config, st *state.State) {
	fmt.Printf("\n  Syncing %s...\n", configName)

	// Find the config item
	var configItem *config.ConfigItem
	for _, c := range cfg.GetAllConfigs() {
		if c.Name == configName {
			configItem = &c
			break
		}
	}

	if configItem == nil {
		ui.Error("Config '%s' not found", configName)
		return
	}

	err := stow.Restow(dotfilesPath, configItem.Path, stow.StowOptions{
		ProgressFunc: func(msg string) {
			fmt.Printf("  %s\n", msg)
		},
	})

	// Update symlink count for this config
	if st != nil {
		stow.UpdateSymlinkCounts(cfg, dotfilesPath, st)
	}

	if err != nil {
		ui.Error("Failed to sync %s: %v", configName, err)
	} else {
		ui.Success("Synced %s", configName)
	}
}

func waitForEnter() {
	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
}
