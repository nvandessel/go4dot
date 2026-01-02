package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/gopherdot/internal/platform"
	"github.com/spf13/cobra"
)

var (
	// Version information (set during build)
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "gopherdot",
	Short: "GopherDot - A Go-based dotfiles manager",
	Long: `GopherDot is a CLI tool for managing dotfiles across multiple machines.
	
It provides:
  • Platform detection (OS, distro, package manager)
  • Dependency management (check and install required tools)
  • Interactive setup with beautiful TUI
  • Machine-specific configuration prompts
  • Stow-based symlink management
  • External dependency cloning (themes, plugins, etc.)
  • Health checking with doctor command
  
GopherDot works with any dotfiles repository that has a .gopherdot.yaml config file.`,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  "Display GopherDot version, build time, and Go version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("GopherDot %s\n", Version)
		fmt.Printf("Built:      %s\n", BuildTime)
		fmt.Printf("Go version: %s\n", GoVersion)
	},
}

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect platform information",
	Long:  "Detect and display information about the current platform (OS, distro, package manager)",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := platform.Detect()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error detecting platform: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Platform Information:")
		fmt.Println("─────────────────────")
		fmt.Println(p.String())
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(detectCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
