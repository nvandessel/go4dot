package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/doctor"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check health of dotfiles installation",
	Long:  "Run health checks on your dotfiles installation and suggest fixes for issues",
	Run: func(cmd *cobra.Command, args []string) {
		// Load config
		var cfg *config.Config
		var dotfilesPath string
		var err error

		if len(args) > 0 {
			cfg, err = config.LoadFromPath(args[0])
			dotfilesPath = filepath.Dir(args[0])
		} else {
			cfg, dotfilesPath, err = config.LoadFromDiscovery()
			if dotfilesPath != "" {
				dotfilesPath = filepath.Dir(dotfilesPath)
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		verbose, _ := cmd.Flags().GetBool("verbose")

		opts := doctor.CheckOptions{
			DotfilesPath: dotfilesPath,
			ProgressFunc: func(msg string) {
				fmt.Println(msg)
			},
		}

		result, err := doctor.RunChecks(cfg, opts)
		if err != nil {
			ui.Error("Error running checks: %v", err)
			os.Exit(1)
		}

		ui.Section("Health Report")

		// Platform info
		if result.Platform != nil {
			fmt.Printf("Platform: %s", result.Platform.OS)
			if result.Platform.Distro != "" {
				fmt.Printf(" (%s)", result.Platform.Distro)
			}
			fmt.Printf(" [%s]\n\n", result.Platform.PackageManager)
		}

		// Checks
		for _, check := range result.Checks {
			switch check.Status {
			case doctor.StatusOK:
				ui.Success("%s: %s", check.Name, check.Message)
			case doctor.StatusWarning:
				ui.Warning("%s: %s", check.Name, check.Message)
			case doctor.StatusError:
				ui.Error("%s: %s", check.Name, check.Message)
			case doctor.StatusSkipped:
				// Print skipped nicely
				fmt.Printf("  ⊘ %s: %s\n", check.Name, check.Message)
			}

			if verbose && check.Fix != "" && check.Status != doctor.StatusOK {
				fmt.Printf("    Fix: %s\n", check.Fix)
			}
		}

		fmt.Println()
		ui.Section("Summary")

		ok, warnings, errors, skipped := result.CountByStatus()
		if errors > 0 {
			ui.Error("%d errors found", errors)
		}
		if warnings > 0 {
			ui.Warning("%d warnings", warnings)
		}
		if ok > 0 {
			ui.Success("%d checks passed", ok)
		}
		if skipped > 0 {
			fmt.Printf("  ⊘ %d skipped\n", skipped)
		}

		// Fixes
		if !result.IsHealthy() || result.HasWarnings() {
			fixes := result.GetFixes()
			if len(fixes) > 0 {
				ui.Section("Suggested Fixes")
				for i, fix := range fixes {
					fmt.Printf("%d. %s\n", i+1, fix)
				}
			}
		}

		// Exit with error code if unhealthy
		if !result.IsHealthy() {
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)

	// Flags for doctor
	doctorCmd.Flags().BoolP("verbose", "v", false, "Show detailed output including individual items")
}
