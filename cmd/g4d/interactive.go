package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/nvandessel/go4dot/internal/version"
	"github.com/spf13/cobra"
)

// runInteractive starts the interactive dashboard
func runInteractive(cmd *cobra.Command, args []string) {
	// If arguments provided, don't run interactive
	if len(args) > 0 {
		return
	}

	ui.PrintBanner(Version)

	// Check for updates in background
	updateMsgChan := make(chan string)
	go func() {
		res, err := version.CheckForUpdates(Version)
		if err == nil && res != nil && res.IsOutdated {
			updateMsgChan <- fmt.Sprintf("• Update available: %s -> %s", res.CurrentVersion, res.LatestVersion)
		} else {
			updateMsgChan <- ""
		}
	}()

	updateMsg := ""
	firstRun := true

	for {
		// Wait for update check only on first run, but don't block too long?
		// Actually, if it's slow, we don't want to delay the menu.
		// But since we are already inside the loop, we can just check if channel has data non-blocking
		if firstRun {
			// We can't easily wait for background task without delaying startup.
			// Let's just pass empty string first time, and if we loop back, check channel.
			// Or better: just wait up to 500ms? No, responsiveness is key.
			// Let's just use select with default
			select {
			case msg := <-updateMsgChan:
				updateMsg = msg
			default:
				// Not ready yet
			}
			firstRun = false
		} else {
			// Subsequent runs (after returning from a command)
			select {
			case msg := <-updateMsgChan:
				updateMsg = msg
			default:
			}
		}

		action, err := ui.RunInteractiveMenu(updateMsg)
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
