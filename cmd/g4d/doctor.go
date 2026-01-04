package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/doctor"
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
			fmt.Fprintf(os.Stderr, "Error running checks: %v\n", err)
			os.Exit(1)
		}

		fmt.Println()

		if verbose {
			fmt.Print(result.DetailedReport())
		} else {
			fmt.Print(result.Report())
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
