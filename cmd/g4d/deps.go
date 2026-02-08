package main

import (
	"fmt"
	"io"
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

		if err := runDepsInstall(cfg, p, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
}

func runDepsInstall(cfg *config.Config, p *platform.Platform, stdout io.Writer) error {
	// Check current status
	checkResult, err := deps.Check(cfg, p)
	if err != nil {
		return fmt.Errorf("error checking dependencies: %w", err)
	}

	manualMissing := checkResult.GetManualMissing()
	missing := checkResult.GetMissing()
	if len(missing) == 0 {
		if len(manualMissing) == 0 {
			_, _ = fmt.Fprintln(stdout, "All dependencies are already installed!")
			return nil
		}

		_, _ = fmt.Fprintf(stdout, "Manual (install required): %d packages\n", len(manualMissing))
		for _, dep := range manualMissing {
			_, _ = fmt.Fprintf(stdout, "  - %s (install manually)\n", dep.Item.Name)
		}
		_, _ = fmt.Fprintln(stdout, "\nAll auto-installable dependencies are already installed.")
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Installing %d missing dependencies...\n\n", len(missing))

	// Install with progress
	opts := deps.InstallOptions{
		OnlyMissing: true,
		ProgressFunc: func(current, total int, msg string) {
			if total > 0 && current > 0 {
				_, _ = fmt.Fprintf(stdout, "[%d/%d] %s\n", current, total, msg)
			} else {
				_, _ = fmt.Fprintln(stdout, msg)
			}
		},
	}

	result, err := deps.Install(cfg, p, opts)
	if err != nil {
		return fmt.Errorf("error during installation: %w", err)
	}

	// Show results
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintf(stdout, "Installed: %d packages\n", len(result.Installed))
	if len(result.ManualSkipped) > 0 {
		_, _ = fmt.Fprintf(stdout, "Manual (skipped): %d packages\n", len(result.ManualSkipped))
		for _, dep := range result.ManualSkipped {
			_, _ = fmt.Fprintf(stdout, "  - %s (install manually)\n", dep.Name)
		}
	}
	if len(result.Failed) > 0 {
		_, _ = fmt.Fprintf(stdout, "Failed: %d packages\n", len(result.Failed))
		for _, fail := range result.Failed {
			_, _ = fmt.Fprintf(stdout, "  - %s: %v\n", fail.Item.Name, fail.Error)
		}
		return fmt.Errorf("error during installation: %d packages failed", len(result.Failed))
	}

	return nil
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
	case deps.StatusManualMissing:
		status = "m"
		info = "missing (manual install required)"
	}

	label := dep.Item.Name
	if dep.Item.Manual {
		label += " [manual]"
	}

	fmt.Printf("  %s %s (%s)\n", status, label, info)
}

func init() {
	rootCmd.AddCommand(depsCmd)
	depsCmd.AddCommand(depsCheckCmd)
	depsCmd.AddCommand(depsInstallCmd)
}
