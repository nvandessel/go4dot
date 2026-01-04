package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/spf13/cobra"
)

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
		fmt.Println("---------------------")
		fmt.Println(p.String())
	},
}

func init() {
	rootCmd.AddCommand(detectCmd)
}
