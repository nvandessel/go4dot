package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/setup"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/nvandessel/go4dot/internal/ui/dashboard"
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

		// Use unified dashboard UI for interactive mode
		if ui.IsInteractive() && !auto {
			runInstallDashboard(cfg, dotfilesPath, dashboard.InstallOptions{
				Auto:         auto,
				Minimal:      minimal,
				SkipDeps:     skipDeps,
				SkipExternal: skipExternal,
				SkipMachine:  skipMachine,
				SkipStow:     skipStow,
				Overwrite:    overwrite,
			})
			return
		}

		// Non-interactive mode: use legacy stdout-based flow
		opts := setup.InstallOptions{
			Auto:         auto,
			Minimal:      minimal,
			SkipDeps:     skipDeps,
			SkipExternal: skipExternal,
			SkipMachine:  skipMachine,
			SkipStow:     skipStow,
			Overwrite:    overwrite,
			ProgressFunc: func(current, total int, msg string) {
				// Simple heuristic to style the output from setup package
				if len(msg) > 0 && msg[0] == '\n' {
					ui.Section(msg[1:]) // Remove newline and print as section
					return
				}

				// Build item counter prefix if we have counts
				var counterPrefix string
				if total > 0 && current > 0 {
					counterPrefix = fmt.Sprintf("[%d/%d] ", current, total)
				}

				// Already styled symbols from setup package: ✓, ⚠, ⊘, ✗, ⬇, ↻
				// We can just print them, or replace them with our UI icons
				if len(msg) > 2 {
					prefix := msg[:2] // Get symbol and space
					content := msg[2:]

					switch prefix {
					case "✓ ":
						ui.Success("%s%s", counterPrefix, content)
						return
					case "⚠ ":
						ui.Warning("%s%s", counterPrefix, content)
						return
					case "✗ ":
						ui.Error("%s%s", counterPrefix, content)
						return
					case "⊘ ":
						// Skip symbol, print as info/subtle
						fmt.Printf("  %s%s\n", counterPrefix, msg)
						return
					case "⬇ ", "↻ ":
						// Download/update in progress
						fmt.Printf("  %s%s\n", counterPrefix, msg)
						return
					}
				}

				// Default - include counter if present
				if counterPrefix != "" {
					fmt.Printf("%s%s\n", counterPrefix, msg)
				} else {
					fmt.Println(msg)
				}
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
			if err := setup.SaveState(cfg, dotfilesPath, result); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save state: %v\n", err)
			}

			// Show post-install message if present
			if cfg.PostInstall != "" {
				ui.Section("Next Steps")
				fmt.Println(cfg.PostInstall)
			}
		}
	},
}

// runInstallDashboard runs the install process within the unified dashboard UI
func runInstallDashboard(cfg *config.Config, dotfilesPath string, opts dashboard.InstallOptions) {
	p, _ := platform.Detect()

	driftSummary, _ := stow.FullDriftCheck(cfg, dotfilesPath)
	linkStatus, _ := stow.GetAllConfigLinkStatus(cfg, dotfilesPath)
	machineStatus := machine.CheckMachineConfigStatus(cfg)

	var dashStatus []dashboard.MachineStatus
	for _, s := range machineStatus {
		dashStatus = append(dashStatus, dashboard.MachineStatus{
			ID:          s.ID,
			Description: s.Description,
			Status:      s.Status,
		})
	}

	state := dashboard.State{
		Platform:      p,
		DriftSummary:  driftSummary,
		LinkStatus:    linkStatus,
		MachineStatus: dashStatus,
		Configs:       cfg.GetAllConfigs(),
		DotfilesPath:  dotfilesPath,
		HasConfig:     true,
	}

	_, err := dashboard.RunWithOperation(state, dashboard.OpInstall, "", nil, func(runner *dashboard.OperationRunner) error {
		_, err := dashboard.RunInstallOperation(runner, cfg, dotfilesPath, opts)
		return err
	})

	if err != nil {
		ui.Error("Installation failed: %v", err)
		os.Exit(1)
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
