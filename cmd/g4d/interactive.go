package main

import (
	"fmt"
	"os"
	"path/filepath"

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
			// No config found - show setup screen
			result, err = dashboard.RunSetup(p, updateMsg)
		} else {
			// Config exists - show health dashboard
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()

			var driftSummary *stow.DriftSummary
			if st != nil {
				driftSummary, _ = stow.QuickDriftCheck(cfg, dotfilesPath, st)
			}

			allConfigs := cfg.GetAllConfigs()
			result, err = dashboard.Run(p, driftSummary, allConfigs, dotfilesPath, updateMsg)
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
			runStowRefresh(dotfilesPath, cfg, st)
			waitForEnter()
		}

	case dashboard.ActionSyncConfig:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()
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
	fmt.Println("\n  Syncing all configs...")

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
