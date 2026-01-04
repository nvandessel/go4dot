package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect platform information",
	Long:  "Detect and display information about the current platform (OS, distro, package manager)",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := platform.Detect()
		if err != nil {
			ui.Error("Error detecting platform: %v", err)
			os.Exit(1)
		}

		ui.Section("Platform Information")
		fmt.Printf("OS:              %s\n", p.OS)
		if p.Distro != "" {
			fmt.Printf("Distro:          %s\n", p.Distro)
		}
		if p.DistroVersion != "" {
			fmt.Printf("Version:         %s\n", p.DistroVersion)
		}
		fmt.Printf("Package Manager: %s\n", p.PackageManager)
		if p.IsWSL {
			ui.Info("Running inside WSL")
		}
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
