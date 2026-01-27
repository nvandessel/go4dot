package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/spf13/cobra"
)

var stowCmd = &cobra.Command{
	Use:   "stow",
	Short: "Manage dotfile symlinks",
	Long:  "Commands for stowing, unstowing, and managing dotfile symlinks",
}

var stowAddCmd = &cobra.Command{
	Use:   "add <config-name> [config-path]",
	Short: "Stow a specific config",
	Long: `Create symlinks for a specific dotfile configuration.

DEPRECATED: Consider using the unified dashboard (g4d) for a better experience.
The dashboard provides progress tracking, conflict resolution, and visual feedback.`,
	Args: cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintln(os.Stderr, "Note: For a better experience, use 'g4d' to access the unified dashboard.")
		fmt.Fprintln(os.Stderr, "")
		configName := args[0]

		// Load config
		var cfg *config.Config
		var configPath string
		var err error

		if len(args) > 1 {
			cfg, err = config.LoadFromPath(args[1])
			configPath = args[1]
		} else {
			cfg, configPath, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Find the config item
		cfgItem := cfg.GetConfigByName(configName)
		if cfgItem == nil {
			fmt.Fprintf(os.Stderr, "Error: config '%s' not found\n", configName)
			os.Exit(1)
		}

		// Get dotfiles directory
		dotfilesPath := filepath.Dir(configPath)

		// Stow it
		opts := stow.StowOptions{
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		err = stow.Stow(dotfilesPath, cfgItem.Path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var stowRemoveCmd = &cobra.Command{
	Use:   "remove <config-name> [config-path]",
	Short: "Unstow a specific config",
	Long:  "Remove symlinks for a specific dotfile configuration",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		configName := args[0]

		var cfg *config.Config
		var configPath string
		var err error

		if len(args) > 1 {
			cfg, err = config.LoadFromPath(args[1])
			configPath = args[1]
		} else {
			cfg, configPath, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		cfgItem := cfg.GetConfigByName(configName)
		if cfgItem == nil {
			fmt.Fprintf(os.Stderr, "Error: config '%s' not found\n", configName)
			os.Exit(1)
		}

		dotfilesPath := filepath.Dir(configPath)

		opts := stow.StowOptions{
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		err = stow.Unstow(dotfilesPath, cfgItem.Path, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var stowRefreshCmd = &cobra.Command{
	Use:   "refresh [config-path]",
	Short: "Refresh all stowed configs",
	Long:  "Restow all configs to update symlinks",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var configPath string
		var err error

		if len(args) > 0 {
			cfg, err = config.LoadFromPath(args[0])
			configPath = args[0]
		} else {
			cfg, configPath, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		dotfilesPath := filepath.Dir(configPath)

		// Restow all configs
		opts := stow.StowOptions{
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		allConfigs := cfg.GetAllConfigs()
		fmt.Printf("Refreshing %d configs...\n\n", len(allConfigs))

		result := stow.RestowConfigs(dotfilesPath, allConfigs, opts)

		// Show results
		fmt.Println()
		if len(result.Success) > 0 {
			fmt.Printf("Refreshed: %d configs\n", len(result.Success))
		}
		if len(result.Skipped) > 0 {
			fmt.Printf("Skipped: %d configs\n", len(result.Skipped))
		}
		if len(result.Failed) > 0 {
			fmt.Printf("Failed: %d configs\n", len(result.Failed))
			for _, fail := range result.Failed {
				fmt.Printf("  - %s: %v\n", fail.ConfigName, fail.Error)
			}
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(stowCmd)
	stowCmd.AddCommand(stowAddCmd)
	stowCmd.AddCommand(stowRemoveCmd)
	stowCmd.AddCommand(stowRefreshCmd)
}
