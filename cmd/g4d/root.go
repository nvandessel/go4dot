package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information (set during build)
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "g4d",
	Short: "go4dot - A Go-based dotfiles manager",
	Long: `go4dot is a CLI tool for managing dotfiles across multiple machines.

It provides:
  - Platform detection (OS, distro, package manager)
  - Dependency management (check and install required tools)
  - Interactive setup with beautiful TUI
  - Machine-specific configuration prompts
  - Stow-based symlink management
  - External dependency cloning (themes, plugins, etc.)
  - Health checking with doctor command

go4dot works with any dotfiles repository that has a .go4dot.yaml config file.`,
	Run: runInteractive,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  "Display go4dot version, build time, and Go version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("go4dot %s\n", Version)
		fmt.Printf("Built:      %s\n", BuildTime)
		fmt.Printf("Go version: %s\n", GoVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
