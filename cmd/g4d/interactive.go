package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

// runInteractive starts the interactive dashboard
func runInteractive(cmd *cobra.Command, args []string) {
	// If arguments provided, don't run interactive
	if len(args) > 0 {
		return
	}

	ui.PrintBanner(Version)

	for {
		action, err := ui.RunInteractiveMenu()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error running interactive menu: %v\n", err)
			os.Exit(1)
		}

		switch action {
		case ui.ActionInstall:
			// Run install command logic
			installCmd.Run(installCmd, nil)
			pause()
		case ui.ActionUpdate:
			// Run update command logic
			updateCmd.Run(updateCmd, nil)
			pause()
		case ui.ActionDoctor:
			// Run doctor command logic
			doctorCmd.Run(doctorCmd, nil)
			pause()
		case ui.ActionList:
			// Run list command logic
			listCmd.Run(listCmd, nil)
			pause()
		case ui.ActionInit:
			// Run init command logic
			initCmd.Run(initCmd, nil)
			pause()
		case ui.ActionQuit:
			fmt.Println("Bye! 👋")
			os.Exit(0)
		}
	}
}

func pause() {
	fmt.Println("\nPress Enter to continue...")
	fmt.Scanln()
}
