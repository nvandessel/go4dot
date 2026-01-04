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

	ui.PrintBanner(Version)

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

	// Try to load config to determine context
	cfg, configPath, err := config.LoadFromDiscovery()
	hasConfig := err == nil && cfg != nil

	// Get update message (non-blocking)
	updateMsg := ""
	select {
	case msg := <-updateMsgChan:
		updateMsg = msg
	default:
	}

	if !hasConfig {
		// No config found - prompt to init
		runNoConfigFlow()
		return
	}

	// Config exists - run the new dashboard
	runDashboardFlow(cfg, configPath, updateMsg, updateMsgChan)
}

// runNoConfigFlow handles the case when no .go4dot.yaml is found
func runNoConfigFlow() {
	cwd, _ := os.Getwd()
	fmt.Printf("\n  No .go4dot.yaml found in %s\n\n", filepath.Base(cwd))

	var initHere bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to initialize go4dot here?").
				Affirmative("Yes, set up go4dot").
				Negative("No thanks").
				Value(&initHere),
		),
	)

	if err := form.Run(); err != nil {
		return
	}

	if initHere {
		initCmd.Run(initCmd, nil)
	} else {
		fmt.Println("\nRun 'g4d init' when you're ready to set up your dotfiles.")
	}
}

// runDashboardFlow runs the main dashboard when a config exists
func runDashboardFlow(cfg *config.Config, configPath string, updateMsg string, updateMsgChan chan string) {
	// Get dotfiles path from config path
	dotfilesPath := filepath.Dir(configPath)

	// Detect platform
	p, _ := platform.Detect()

	// Load state for drift detection
	st, _ := state.Load()

	for {
		// Check for update message if we looped back
		select {
		case msg := <-updateMsgChan:
			if msg != "" {
				updateMsg = msg
			}
		default:
		}

		// Quick drift check
		var driftSummary *stow.DriftSummary
		if st != nil {
			driftSummary, _ = stow.QuickDriftCheck(cfg, dotfilesPath, st)
		}

		// Get all configs for display
		allConfigs := cfg.GetAllConfigs()

		// Run the dashboard
		result, err := dashboard.Run(p, driftSummary, allConfigs, dotfilesPath, updateMsg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running dashboard: %v\n", err)
			os.Exit(1)
		}

		if result == nil {
			return
		}

		switch result.Action {
		case dashboard.ActionQuit:
			fmt.Println("Bye!")
			return

		case dashboard.ActionSync:
			// Sync all configs (restow)
			runStowRefresh(dotfilesPath, cfg, st)
			pause()

		case dashboard.ActionSyncConfig:
			// Sync specific config
			runStowSingle(dotfilesPath, result.ConfigName, cfg, st)
			pause()

		case dashboard.ActionDoctor:
			doctorCmd.Run(doctorCmd, nil)
			pause()

		case dashboard.ActionInstall:
			installCmd.Run(installCmd, nil)
			pause()

		case dashboard.ActionInit:
			initCmd.Run(initCmd, nil)
			pause()
		}
	}
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

func pause() {
	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
}
