package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/spf13/cobra"
)

var depsCmd = &cobra.Command{
	Use:   "deps",
	Short: "Manage dependencies",
	Long:  "Commands for checking and installing system dependencies",
}

var depsCheckCmd = &cobra.Command{
	Use:   "check [config-path]",
	Short: "Check dependency status",
	Long:  "Check which dependencies are installed and which are missing",
	Args:  cobra.MaximumNArgs(1),
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

		// Detect platform
		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		// Check dependencies
		result, err := deps.Check(cfg, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking dependencies: %v\n", err)
			os.Exit(1)
		}

		// Display results
		fmt.Println("Dependency Status")
		fmt.Println("-----------------")
		fmt.Printf("Package Manager: %s\n", p.PackageManager)
		fmt.Printf("Summary: %s\n\n", result.Summary())

		// Show critical deps
		if len(result.Critical) > 0 {
			fmt.Println("Critical Dependencies:")
			for _, dep := range result.Critical {
				printDepStatus(dep)
			}
			fmt.Println()
		}

		// Show core deps
		if len(result.Core) > 0 {
			fmt.Println("Core Dependencies:")
			for _, dep := range result.Core {
				printDepStatus(dep)
			}
			fmt.Println()
		}

		// Show optional deps
		if len(result.Optional) > 0 {
			fmt.Println("Optional Dependencies:")
			for _, dep := range result.Optional {
				printDepStatus(dep)
			}
		}

		// Exit with error if critical deps are missing
		if len(result.GetMissingCritical()) > 0 {
			fmt.Fprintf(os.Stderr, "\nError: Missing critical dependencies. Run 'g4d deps install' to install them.\n")
			os.Exit(1)
		}
	},
}

var depsInstallCmd = &cobra.Command{
	Use:   "install [config-path]",
	Short: "Install missing dependencies",
	Long:  "Install system packages for missing dependencies",
	Args:  cobra.MaximumNArgs(1),
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

		// Detect platform
		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		// Check current status
		checkResult, err := deps.Check(cfg, p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking dependencies: %v\n", err)
			os.Exit(1)
		}

		missing := checkResult.GetMissing()
		if len(missing) == 0 {
			fmt.Println("All dependencies are already installed!")
			return
		}

		fmt.Printf("Installing %d missing dependencies...\n\n", len(missing))

		// Install with progress
		opts := deps.InstallOptions{
			OnlyMissing: true,
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		result, err := deps.Install(cfg, p, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error during installation: %v\n", err)
			os.Exit(1)
		}

		// Show results
		fmt.Println()
		fmt.Printf("Installed: %d packages\n", len(result.Installed))
		if len(result.Failed) > 0 {
			fmt.Printf("Failed: %d packages\n", len(result.Failed))
			for _, fail := range result.Failed {
				fmt.Printf("  - %s: %v\n", fail.Item.Name, fail.Error)
			}
			os.Exit(1)
		}
	},
}

func printDepStatus(dep deps.DependencyCheck) {
	status := "x"
	info := "missing"

	switch dep.Status {
	case deps.StatusInstalled:
		status = "+"
		info = dep.InstalledPath
		if dep.InstalledVersion != "" {
			info = fmt.Sprintf("%s (v%s)", info, dep.InstalledVersion)
		}
	case deps.StatusVersionMismatch:
		status = "!"
		info = fmt.Sprintf("version mismatch: found v%s, want %s", dep.InstalledVersion, dep.RequiredVersion)
	case deps.StatusCheckFailed:
		status = "?"
		info = fmt.Sprintf("check failed: %v", dep.Error)
	}

	fmt.Printf("  %s %s (%s)\n", status, dep.Item.Name, info)
}

func init() {
	rootCmd.AddCommand(depsCmd)
	depsCmd.AddCommand(depsCheckCmd)
	depsCmd.AddCommand(depsInstallCmd)
}
