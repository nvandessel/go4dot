package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync [config-name]",
	Short: "Sync symlinks for dotfiles configs",
	Long: `Sync symlinks for all or specific dotfiles configurations.

This command restows your dotfiles, creating any new symlinks for files
that have been added to your configs. It's useful when you've added new
files to a config (like a new Neovim plugin) and need to create symlinks.

Without arguments, syncs all configs. With a config name, syncs only that config.

Examples:
  g4d sync           # Sync all configs
  g4d sync nvim      # Sync only the nvim config
  g4d sync -y        # Sync all without confirmation`,
	Run: runSync,
}

func init() {
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) {
	// Load config
	cfg, configPath, err := config.LoadFromDiscovery()
	if err != nil {
		ui.Error("Failed to load config: %v", err)
		os.Exit(1)
	}

	dotfilesPath := filepath.Dir(configPath)

	// Load state
	st, _ := state.Load()
	if st == nil {
		st = state.New()
	}

	// If a specific config is specified, sync just that one
	if len(args) > 0 {
		syncSingleConfig(args[0], cfg, dotfilesPath, st)
		return
	}

	// Sync all configs
	syncAllConfigs(cfg, dotfilesPath, st)
}

func syncSingleConfig(configName string, cfg *config.Config, dotfilesPath string, st *state.State) {
	// Find the config
	var configItem *config.ConfigItem
	for _, c := range cfg.GetAllConfigs() {
		if c.Name == configName {
			configItem = &c
			break
		}
	}

	if configItem == nil {
		ui.Error("Config '%s' not found", configName)
		os.Exit(1)
	}

	// Check what will be synced
	summary, err := stow.FullDriftCheck(cfg, dotfilesPath)
	if err != nil {
		ui.Error("Failed to check drift: %v", err)
		os.Exit(1)
	}

	var drift *stow.DriftResult
	for _, r := range summary.Results {
		if r.ConfigName == configName {
			drift = &r
			break
		}
	}

	// Show what will be synced
	if drift != nil && drift.HasDrift {
		fmt.Printf("\nChanges to sync for %s:\n", configName)
		for _, f := range drift.NewFiles {
			fmt.Printf("  + %s (new)\n", f)
		}
		for _, f := range drift.ConflictFiles {
			fmt.Printf("  ! %s (conflict)\n", f)
		}
		for _, f := range drift.MissingFiles {
			fmt.Printf("  - %s (missing/orphaned)\n", f)
		}
		fmt.Println()
	} else {
		fmt.Printf("\n%s is already in sync.\n", configName)
	}

	// Confirm unless non-interactive
	if ui.IsInteractive() {
		var proceed bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Sync %s?", configName)).
					Affirmative("Yes").
					Negative("No").
					Value(&proceed),
			),
		).Run()

		if err != nil || !proceed {
			fmt.Println("Sync cancelled.")
			return
		}
	}

	// Do the sync
	err = stow.SyncSingle(dotfilesPath, configName, cfg, st, stow.StowOptions{
		ProgressFunc: func(current, total int, msg string) {
			if total > 0 && current > 0 {
				fmt.Printf("  [%d/%d] %s\n", current, total, msg)
			} else {
				fmt.Printf("  %s\n", msg)
			}
		},
	})

	if err != nil {
		ui.Error("Failed to sync %s: %v", configName, err)
		os.Exit(1)
	}

	ui.Success("Synced %s", configName)
}

func syncAllConfigs(cfg *config.Config, dotfilesPath string, st *state.State) {
	// Check what will be synced
	summary, err := stow.FullDriftCheck(cfg, dotfilesPath)
	if err != nil {
		ui.Error("Failed to check drift: %v", err)
		os.Exit(1)
	}

	drifted := stow.GetDriftedConfigs(summary.Results)

	// Show what will be synced
	if summary.HasDrift() {
		if len(drifted) > 0 {
			fmt.Println("\nConfigs with changes:")
			for _, r := range drifted {
				fmt.Printf("  %s:\n", r.ConfigName)
				for _, f := range r.NewFiles {
					fmt.Printf("    + %s (new)\n", f)
				}
				for _, f := range r.ConflictFiles {
					fmt.Printf("    ! %s (conflict)\n", f)
				}
				for _, f := range r.MissingFiles {
					fmt.Printf("    - %s (missing/orphaned)\n", f)
				}
			}
		}

		if len(summary.RemovedConfigs) > 0 {
			fmt.Println("\nRemoved configs still stowed:")
			for _, name := range summary.RemovedConfigs {
				fmt.Printf("  - %s (removed from YAML)\n", name)
			}
		}
		fmt.Println()
	} else {
		fmt.Println("\nAll configs are in sync.")
	}

	allConfigs := cfg.GetAllConfigs()

	// Confirm unless non-interactive
	if ui.IsInteractive() {
		var proceed bool
		err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Sync %d config(s)?", len(allConfigs))).
					Affirmative("Yes").
					Negative("No").
					Value(&proceed),
			),
		).Run()

		if err != nil || !proceed {
			fmt.Println("Sync cancelled.")
			return
		}
	}

	// Do the sync
	result, err := stow.SyncAll(dotfilesPath, cfg, st, ui.IsInteractive(), stow.StowOptions{
		ProgressFunc: func(current, total int, msg string) {
			if total > 0 && current > 0 {
				fmt.Printf("  [%d/%d] %s\n", current, total, msg)
			} else {
				fmt.Printf("  %s\n", msg)
			}
		},
	})

	if err != nil {
		ui.Error("%v", err)
		os.Exit(1)
	}

	if len(result.Failed) > 0 {
		ui.Error("Failed to sync %d config(s)", len(result.Failed))
		for _, f := range result.Failed {
			fmt.Printf("  %s: %v\n", f.ConfigName, f.Error)
		}
		os.Exit(1)
	}

	ui.Success("Synced %d config(s)", len(result.Success))
}
