package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/spf13/cobra"
)

var externalCmd = &cobra.Command{
	Use:   "external",
	Short: "Manage external dependencies",
	Long:  "Commands for cloning, updating, and managing external dependencies (plugins, themes, etc.)",
}

var externalStatusCmd = &cobra.Command{
	Use:   "status [config-path]",
	Short: "Show status of external dependencies",
	Long:  "Display the installation status of all external dependencies",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		if len(cfg.External) == 0 {
			fmt.Println("No external dependencies defined in config")
			return
		}

		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		statuses := deps.CheckExternalStatus(cfg, p)

		fmt.Println("External Dependencies Status")
		fmt.Println("----------------------------")

		var installed, missing, skipped int
		for _, s := range statuses {
			var statusIcon string
			var info string

			switch s.Status {
			case "installed":
				statusIcon = "+"
				info = s.Path
				installed++
			case "missing":
				statusIcon = "x"
				info = "not installed"
				missing++
			case "skipped":
				statusIcon = "o"
				info = s.Reason
				skipped++
			case "error":
				statusIcon = "!"
				info = s.Reason
			}

			fmt.Printf("  %s %s (%s)\n", statusIcon, s.Dep.Name, info)
		}

		fmt.Printf("\nSummary: %d installed, %d missing, %d skipped\n", installed, missing, skipped)

		if missing > 0 {
			fmt.Println("\nRun 'g4d external clone' to install missing dependencies.")
		}
	},
}

var externalCloneCmd = &cobra.Command{
	Use:   "clone [id] [config-path]",
	Short: "Clone external dependencies",
	Long: `Clone external dependencies from their repositories.

Without arguments, clones all missing external dependencies.
With an ID argument, clones only that specific dependency.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var err error
		var specificID string

		// Parse arguments
		configPathArg := ""
		if len(args) >= 1 {
			// Could be ID or config path
			// If it looks like a path, treat it as such
			if _, statErr := os.Stat(args[0]); statErr == nil || filepath.Ext(args[0]) == ".yaml" || filepath.Ext(args[0]) == ".yml" {
				configPathArg = args[0]
			} else {
				specificID = args[0]
				if len(args) >= 2 {
					configPathArg = args[1]
				}
			}
		}

		if configPathArg != "" {
			cfg, err = config.LoadFromPath(configPathArg)
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.External) == 0 {
			fmt.Println("No external dependencies defined in config")
			return
		}

		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		opts := deps.ExternalOptions{
			ProgressFunc: func(msg string) {
				fmt.Println(msg)
			},
		}

		if specificID != "" {
			// Clone single
			fmt.Printf("Cloning %s...\n\n", specificID)
			err = deps.CloneSingle(cfg, p, specificID, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nDone")
		} else {
			// Clone all
			fmt.Printf("Cloning %d external dependencies...\n\n", len(cfg.External))
			result, err := deps.CloneExternal(cfg, p, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Show results
			fmt.Println()
			if len(result.Cloned) > 0 {
				fmt.Printf("Cloned: %d\n", len(result.Cloned))
			}
			if len(result.Updated) > 0 {
				fmt.Printf("Updated: %d\n", len(result.Updated))
			}
			if len(result.Skipped) > 0 {
				fmt.Printf("Skipped: %d\n", len(result.Skipped))
			}
			if len(result.Failed) > 0 {
				fmt.Printf("Failed: %d\n", len(result.Failed))
				for _, fail := range result.Failed {
					fmt.Printf("  - %s: %v\n", fail.Dep.Name, fail.Error)
				}
				os.Exit(1)
			}
		}
	},
}

var externalUpdateCmd = &cobra.Command{
	Use:   "update [id] [config-path]",
	Short: "Update external dependencies",
	Long: `Pull updates for installed external dependencies.

Without arguments, updates all installed external dependencies.
With an ID argument, updates only that specific dependency.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var err error
		var specificID string

		configPathArg := ""
		if len(args) >= 1 {
			if _, statErr := os.Stat(args[0]); statErr == nil || filepath.Ext(args[0]) == ".yaml" || filepath.Ext(args[0]) == ".yml" {
				configPathArg = args[0]
			} else {
				specificID = args[0]
				if len(args) >= 2 {
					configPathArg = args[1]
				}
			}
		}

		if configPathArg != "" {
			cfg, err = config.LoadFromPath(configPathArg)
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.External) == 0 {
			fmt.Println("No external dependencies defined in config")
			return
		}

		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		opts := deps.ExternalOptions{
			Update: true,
			ProgressFunc: func(msg string) {
				fmt.Println(msg)
			},
		}

		if specificID != "" {
			// Update single
			fmt.Printf("Updating %s...\n\n", specificID)
			err = deps.CloneSingle(cfg, p, specificID, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("\nDone")
		} else {
			// Update all
			fmt.Printf("Updating %d external dependencies...\n\n", len(cfg.External))
			result, err := deps.CloneExternal(cfg, p, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			// Show results
			fmt.Println()
			if len(result.Updated) > 0 {
				fmt.Printf("Updated: %d\n", len(result.Updated))
			}
			if len(result.Cloned) > 0 {
				fmt.Printf("Cloned (new): %d\n", len(result.Cloned))
			}
			if len(result.Skipped) > 0 {
				fmt.Printf("Skipped: %d\n", len(result.Skipped))
			}
			if len(result.Failed) > 0 {
				fmt.Printf("Failed: %d\n", len(result.Failed))
				for _, fail := range result.Failed {
					fmt.Printf("  - %s: %v\n", fail.Dep.Name, fail.Error)
				}
				os.Exit(1)
			}
		}
	},
}

var externalRemoveCmd = &cobra.Command{
	Use:   "remove <id> [config-path]",
	Short: "Remove an external dependency",
	Long:  "Remove an installed external dependency by its ID",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		var cfg *config.Config
		var err error

		if len(args) > 1 {
			cfg, err = config.LoadFromPath(args[1])
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		opts := deps.ExternalOptions{
			ProgressFunc: func(msg string) {
				fmt.Println(msg)
			},
		}

		err = deps.RemoveExternal(cfg, id, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(externalCmd)
	externalCmd.AddCommand(externalStatusCmd)
	externalCmd.AddCommand(externalCloneCmd)
	externalCmd.AddCommand(externalUpdateCmd)
	externalCmd.AddCommand(externalRemoveCmd)
}
