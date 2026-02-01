package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/setup"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/nvandessel/go4dot/internal/ui/dashboard"
	"github.com/nvandessel/go4dot/internal/version"
	"github.com/spf13/cobra"
)

// runInteractive starts the interactive dashboard
func runInteractive(cmd *cobra.Command, args []string) {
	// If arguments provided, don't run interactive
	if len(args) > 0 {
		return
	}

	// If non-interactive mode, don't run interactive UI
	if !ui.IsInteractive() {
		// In non-interactive mode, just run doctor check and exit
		doctorCmd.Run(doctorCmd, nil)
		return
	}

	// Check for updates in background with proper lifecycle management
	// Derive from cmd.Context() so SIGINT and other cancellations propagate
	updateCtx, cancelUpdate := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancelUpdate()

	updateMsgChan := make(chan string, 1)
	go func() {
		res, err := version.CheckForUpdates(updateCtx, Version)
		if err == nil && res != nil && res.IsOutdated {
			select {
			case updateMsgChan <- fmt.Sprintf("Update: %s -> %s", res.CurrentVersion, res.LatestVersion):
			case <-updateCtx.Done():
				// Context cancelled, don't block on channel send
			}
		} else {
			select {
			case updateMsgChan <- "":
			case <-updateCtx.Done():
				// Context cancelled, don't block on channel send
			}
		}
	}()

	// Get update message (non-blocking)
	updateMsg := ""

	// Detect platform once
	p, _ := platform.Detect()

	// State preservation across dashboard runs
	lastFilter := ""
	lastSelected := ""

	// Main application loop - stays in the app until user quits
	for {
		// Check for update message
		select {
		case msg := <-updateMsgChan:
			if msg != "" {
				updateMsg = msg
			}
		default:
		}

		// Try to load config to determine context
		cfg, configPath, err := config.LoadFromDiscovery()
		hasConfig := err == nil && cfg != nil

		// Build dashboard state - works for both config and no-config cases
		var dotfilesPath string
		var driftSummary *stow.DriftSummary
		var linkStatus map[string]*stow.ConfigLinkStatus
		var dashStatus []dashboard.MachineStatus
		var allConfigs []config.ConfigItem
		hasBaseline := false

		if hasConfig {
			dotfilesPath = filepath.Dir(configPath)
			st, _ := state.Load()
			if st == nil {
				st = state.New()
			}

			driftSummary, _ = stow.FullDriftCheck(cfg, dotfilesPath)
			hasBaseline = len(st.SymlinkCounts) > 0
			linkStatus, _ = stow.GetAllConfigLinkStatus(cfg, dotfilesPath)

			machineStatus := machine.CheckMachineConfigStatus(cfg)
			for _, s := range machineStatus {
				dashStatus = append(dashStatus, dashboard.MachineStatus{
					ID:          s.ID,
					Description: s.Description,
					Status:      s.Status,
				})
			}

			allConfigs = cfg.GetAllConfigs()
		}

		// Always use the dashboard - it handles no-config case with viewNoConfig
		dashState := dashboard.State{
			Platform:       p,
			DriftSummary:   driftSummary,
			LinkStatus:     linkStatus,
			MachineStatus:  dashStatus,
			Configs:        allConfigs,
			Config:         cfg,
			DotfilesPath:   dotfilesPath,
			UpdateMsg:      updateMsg,
			HasBaseline:    hasBaseline,
			HasConfig:      hasConfig,
			FilterText:     lastFilter,
			SelectedConfig: lastSelected,
		}
		result, err := dashboard.Run(dashState)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		if result == nil {
			return
		}

		// Update state for next run
		lastFilter = result.FilterText
		lastSelected = result.SelectedConfig

		// Handle the action
		shouldExit := handleAction(result, cfg, configPath)
		if shouldExit {
			fmt.Println("Bye!")
			return
		}
	}
}

// handleAction processes the user's action and returns true if we should exit
func handleAction(result *dashboard.Result, cfg *config.Config, configPath string) bool {
	switch result.Action {
	case dashboard.ActionQuit:
		return true

	case dashboard.ActionInit:
		// Onboarding was completed inline in the dashboard.
		// The install prompt and execution is now handled within the TUI itself.
		// Nothing to do here - the main loop will reload the config on next iteration.

	// ActionSync, ActionSyncConfig, ActionBulkSync are now handled inline in dashboard
	// and no longer trigger handleAction

	case dashboard.ActionDoctor:
		doctorCmd.Run(doctorCmd, nil)
		waitForEnter()

	case dashboard.ActionMachineConfig:
		machine.RunInteractiveConfig(cfg)
		waitForEnter()

	// ActionInstall and ActionUpdate are now handled inline in dashboard
	// and no longer trigger handleAction

	case dashboard.ActionList:
		// This is the "More" menu
		runMoreMenu(cfg, configPath)
	}

	return false
}

func runMoreMenu(cfg *config.Config, configPath string) {
	var action string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("More Commands").
				Options(
					huh.NewOption("List Configs", "list"),
					huh.NewOption("External Dependencies", "external"),
					huh.NewOption("Uninstall go4dot", "uninstall"),
					huh.NewOption("Back", "back"),
				).
				Value(&action),
		),
	)

	if err := form.Run(); err != nil || action == "back" {
		return
	}

	switch action {
	case "list":
		st, _ := state.Load()
		p, _ := platform.Detect()
		ui.PrintConfigList(cfg, st, p, true)
		waitForEnter()

	case "external":
		runExternalMenu()

	case "uninstall":
		var confirm bool
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Are you sure you want to uninstall?").
					Description("This will remove all symlinks and state.").
					Value(&confirm),
			),
		).Run(); err != nil {
			return
		}

		if confirm {
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()
			opts := setup.UninstallOptions{
				RemoveExternal: true,
				RemoveMachine:  true,
				ProgressFunc: func(current, total int, msg string) {
					if total > 0 && current > 0 {
						fmt.Printf("  [%d/%d] %s\n", current, total, msg)
					} else {
						fmt.Println("  " + msg)
					}
				},
			}
			if err := setup.Uninstall(cfg, dotfilesPath, st, opts); err != nil {
				ui.Error("%v", err)
			} else {
				ui.Success("Uninstall complete")
			}
			waitForEnter()
		}
	}
}

func runExternalMenu() {
	var action string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("External Dependencies").
				Options(
					huh.NewOption("Show Status", "status"),
					huh.NewOption("Clone Missing", "clone"),
					huh.NewOption("Update All", "update"),
					huh.NewOption("Back", "back"),
				).
				Value(&action),
		),
	)

	if err := form.Run(); err != nil || action == "back" {
		return
	}

	switch action {
	case "status":
		externalStatusCmd.Run(externalStatusCmd, nil)
		waitForEnter()
	case "clone":
		externalCloneCmd.Run(externalCloneCmd, nil)
		waitForEnter()
	case "update":
		externalUpdateCmd.Run(externalUpdateCmd, nil)
		waitForEnter()
	}
}

func waitForEnter() {
	fmt.Println("\nPress Enter to continue...")
	_, _ = fmt.Scanln()
}
