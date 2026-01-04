package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/setup"
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
				fmt.Println(msg)
			},
		}

		// Print header
		fmt.Println("========================================")
		fmt.Println("         go4dot Installation           ")
		fmt.Println("========================================")
		fmt.Printf("\nDotfiles: %s\n", dotfilesPath)
		if cfg.Metadata.Name != "" {
			fmt.Printf("Config:   %s\n", cfg.Metadata.Name)
		}
		fmt.Println()

		result, err := setup.Install(cfg, dotfilesPath, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError: %v\n", err)
			os.Exit(1)
		}

		// Print summary
		fmt.Println("\n========================================")
		if result.HasErrors() {
			fmt.Println("Installation completed with errors")
			fmt.Println()
			fmt.Print(result.Summary())

			// Show specific errors
			for _, e := range result.DepsFailed {
				fmt.Printf("  x Dependency %s: %v\n", e.Item.Name, e.Error)
			}
			for _, e := range result.ConfigsFailed {
				fmt.Printf("  x Config %s: %v\n", e.ConfigName, e.Error)
			}
			for _, e := range result.ExternalFailed {
				fmt.Printf("  x External %s: %v\n", e.Dep.Name, e.Error)
			}
			for _, e := range result.Errors {
				fmt.Printf("  x %v\n", e)
			}
			os.Exit(1)
		} else {
			fmt.Println("+ Installation complete!")
			fmt.Println()
			fmt.Print(result.Summary())

			// Show post-install message if present
			if cfg.PostInstall != "" {
				fmt.Println("\n-- Next Steps --")
				fmt.Println(cfg.PostInstall)
			}
		}
	},
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
