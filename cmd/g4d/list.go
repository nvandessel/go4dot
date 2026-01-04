package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list [config-path]",
	Short: "List installed and available configs",
	Long: `Show the status of all dotfile configurations.

Displays:
- Installed configs (currently stowed)
- Available configs (can be installed)
- Platform-specific configs (not available on this platform)
- Archived configs (deprecated/old)`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		var cfg *config.Config
		var err error

		if len(args) > 0 {
			cfg, err = config.LoadFromPath(args[0])
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Load state if it exists
		st, err := state.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load state: %v\n", err)
		}

		// Detect platform
		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		showAll, _ := cmd.Flags().GetBool("all")

		// Build installed set from state
		installed := make(map[string]bool)
		if st != nil {
			for _, c := range st.Configs {
				installed[c.Name] = true
			}
		}

		// Core configs
		ui.Section("Core Configs")
		for _, c := range cfg.Configs.Core {
			printConfigStatus(c, installed, p, showAll)
		}

		// Optional configs
		if len(cfg.Configs.Optional) > 0 {
			ui.Section("Optional Configs")
			for _, c := range cfg.Configs.Optional {
				printConfigStatus(c, installed, p, showAll)
			}
		}

		// External deps
		if len(cfg.External) > 0 {
			ui.Section("External Dependencies")
			for _, e := range cfg.External {
				status := "x"
				info := "not installed"

				if st != nil {
					if ext, ok := st.ExternalDeps[e.ID]; ok && ext.Installed {
						status = "+"
						info = ext.Path
					}
				}

				// Check if skipped due to platform
				if !deps.CheckCondition(e.Condition, p) {
					if showAll {
						fmt.Printf("  ⊘ %s (skipped - platform mismatch)\n", e.Name)
					}
					continue
				}

				if status == "+" {
					ui.Success("%s (%s)", e.Name, info)
				} else {
					fmt.Printf("  • %s (%s)\n", e.Name, info)
				}
			}
		}

		// Machine configs
		if len(cfg.MachineConfig) > 0 {
			ui.Section("Machine Configurations")
			for _, mc := range cfg.MachineConfig {
				status := "x"
				info := "not configured"

				if st != nil {
					if m, ok := st.MachineConfig[mc.ID]; ok {
						status = "+"
						info = m.ConfigPath
					}
				}

				if status == "+" {
					ui.Success("%s (%s)", mc.Description, info)
				} else {
					fmt.Printf("  • %s (%s)\n", mc.Description, info)
				}
			}
		}

		// Archived configs
		if len(cfg.Archived) > 0 && showAll {
			ui.Section("Archived Configs (deprecated)")
			for _, c := range cfg.Archived {
				fmt.Printf("  - %s\n", c.Name)
				if c.Description != "" {
					fmt.Printf("    %s\n", c.Description)
				}
			}
		}

		// Summary
		ui.Section("Summary")
		if st != nil {
			fmt.Printf("Installed: %d configs\n", len(st.Configs))
			if st.DotfilesPath != "" {
				fmt.Printf("Dotfiles:  %s\n", st.DotfilesPath)
			}
		} else {
			ui.Warning("No installation state found. Run 'g4d install' to set up.")
		}
	},
}

func printConfigStatus(c config.ConfigItem, installed map[string]bool, p *platform.Platform, showAll bool) {
	// Check platform compatibility
	if len(c.Platforms) > 0 && !isPlatformMatch(c.Platforms, p) {
		if showAll {
			fmt.Printf("  ⊘ %s (not available on %s)\n", c.Name, p.OS)
		}
		return
	}

	if installed[c.Name] {
		ui.Success("%s - %s (installed)", c.Name, c.Description)
	} else {
		fmt.Printf("  • %s - %s (not installed)\n", c.Name, c.Description)
	}
}

func isPlatformMatch(platforms []string, p *platform.Platform) bool {
	for _, plat := range platforms {
		if plat == p.OS || plat == "all" {
			return true
		}
		// Also check distro
		if plat == p.Distro {
			return true
		}
	}
	return false
}

func init() {
	rootCmd.AddCommand(listCmd)

	listCmd.Flags().BoolP("all", "a", false, "Show all configs including platform-specific and archived")
}
