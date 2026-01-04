package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/setup"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [config-path]",
	Short: "Install and configure dotfiles",
	Long: `Run the full dotfiles installation process.

This command orchestrates:
1. Dependency checking and installation
2. Stowing dotfile configurations
3. Cloning external dependencies (plugins, themes)
4. Configuring machine-specific settings

Use flags to customize the installation:
  --auto       Non-interactive mode, use defaults
  --minimal    Only install core configs
  --skip-deps  Skip dependency installation
  --skip-external  Skip external dependency cloning
  --skip-machine   Skip machine-specific configuration
  --skip-stow      Skip stowing configs`,
	Args: cobra.MaximumNArgs(1),
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

		// Get flags
		auto, _ := cmd.Flags().GetBool("auto")
		minimal, _ := cmd.Flags().GetBool("minimal")
		skipDeps, _ := cmd.Flags().GetBool("skip-deps")
		skipExternal, _ := cmd.Flags().GetBool("skip-external")
		skipMachine, _ := cmd.Flags().GetBool("skip-machine")
		skipStow, _ := cmd.Flags().GetBool("skip-stow")
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		opts := setup.InstallOptions{
			Auto:         auto,
			Minimal:      minimal,
			SkipDeps:     skipDeps,
			SkipExternal: skipExternal,
			SkipMachine:  skipMachine,
			SkipStow:     skipStow,
			Overwrite:    overwrite,
			ProgressFunc: func(msg string) {
				// Simple heuristic to style the output from setup package
				if len(msg) > 0 && msg[0] == '\n' {
					ui.Section(msg[1:]) // Remove newline and print as section
					return
				}

				// Already styled symbols from setup package: ✓, ⚠, ⊘
				// We can just print them, or replace them with our UI icons
				if len(msg) > 2 {
					prefix := msg[:2] // Get symbol and space
					content := msg[2:]

					switch prefix {
					case "✓ ":
						ui.Success("%s", content)
						return
					case "⚠ ":
						ui.Warning("%s", content)
						return
					case "⊘ ":
						// Skip symbol, print as info/subtle
						fmt.Println("  " + msg)
						return
					}
				}

				// Default
				fmt.Println(msg)
			},
		}

		// Print header
		ui.PrintBanner(Version)
		ui.Section("Installation")

		fmt.Printf("Dotfiles: %s\n", dotfilesPath)
		if cfg.Metadata.Name != "" {
			fmt.Printf("Config:   %s\n", cfg.Metadata.Name)
		}

		result, err := setup.Install(cfg, dotfilesPath, opts)
		if err != nil {
			ui.Error("%s", err.Error())
			os.Exit(1)
		}

		// Print summary
		ui.Section("Summary")
		if result.HasErrors() {
			ui.Error("Installation completed with errors")
			fmt.Println()
			fmt.Print(result.Summary())

			// Show specific errors
			for _, e := range result.DepsFailed {
				ui.Error("Dependency %s: %v", e.Item.Name, e.Error)
			}
			for _, e := range result.ConfigsFailed {
				ui.Error("Config %s: %v", e.ConfigName, e.Error)
			}
			for _, e := range result.ExternalFailed {
				ui.Error("External %s: %v", e.Dep.Name, e.Error)
			}
			for _, e := range result.Errors {
				ui.Error("%v", e)
			}
			os.Exit(1)
		} else {
			ui.Success("Installation complete!")
			fmt.Println()
			fmt.Print(result.Summary())

			// Save state
			saveInstallState(cfg, dotfilesPath, result)

			// Show post-install message if present
			if cfg.PostInstall != "" {
				ui.Section("Next Steps")
				fmt.Println(cfg.PostInstall)
			}
		}
	},
}

// saveInstallState saves the installation state for future reference
func saveInstallState(cfg *config.Config, dotfilesPath string, result *setup.InstallResult) {
	st := state.New()
	st.DotfilesPath = dotfilesPath

	// Save platform info
	if result.Platform != nil {
		st.Platform = state.PlatformState{
			OS:             result.Platform.OS,
			Distro:         result.Platform.Distro,
			DistroVersion:  result.Platform.DistroVersion,
			PackageManager: result.Platform.PackageManager,
		}
	}

	// Save installed configs
	for _, configName := range result.ConfigsStowed {
		item := cfg.GetConfigByName(configName)
		isCore := false
		if item != nil {
			// Check if it's a core config
			for _, c := range cfg.Configs.Core {
				if c.Name == configName {
					isCore = true
					break
				}
			}
		}
		st.AddConfig(configName, configName, isCore)
	}

	// Save external deps
	for _, ext := range result.ExternalCloned {
		st.SetExternalDep(ext.ID, ext.Destination, true)
	}

	// Save machine configs
	for _, mc := range result.MachineConfigs {
		st.SetMachineConfig(mc.ID, mc.Destination, false, false)
	}

	// Save state
	if err := st.Save(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
	}
}

func init() {
	rootCmd.AddCommand(installCmd)

	// Flags for install
	installCmd.Flags().Bool("auto", false, "Non-interactive mode, use defaults")
	installCmd.Flags().Bool("minimal", false, "Only install core configs, skip optional")
	installCmd.Flags().Bool("skip-deps", false, "Skip dependency installation")
	installCmd.Flags().Bool("skip-external", false, "Skip external dependency cloning")
	installCmd.Flags().Bool("skip-machine", false, "Skip machine-specific configuration")
	installCmd.Flags().Bool("skip-stow", false, "Skip stowing configs")
	installCmd.Flags().Bool("overwrite", false, "Overwrite existing files")
}
