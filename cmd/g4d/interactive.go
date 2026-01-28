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

		var result *dashboard.Result

		if !hasConfig {
			// No config found - show polished init prompt
			result = runNoConfigPrompt()
			err = nil
		} else {
			// Config exists - show health dashboard
			dotfilesPath := filepath.Dir(configPath)
			st, _ := state.Load()
			if st == nil {
				st = state.New()
			}

			var driftSummary *stow.DriftSummary
			hasBaseline := false
			if st != nil {
				driftSummary, _ = stow.FullDriftCheck(cfg, dotfilesPath)
				// Check if we have any stored symlink counts (indicates prior sync)
				hasBaseline = len(st.SymlinkCounts) > 0
			}

			// Get link status for all configs
			linkStatus, _ := stow.GetAllConfigLinkStatus(cfg, dotfilesPath)

			machineStatus := machine.CheckMachineConfigStatus(cfg)

			// Convert to dashboard type
			var dashStatus []dashboard.MachineStatus
			for _, s := range machineStatus {
				dashStatus = append(dashStatus, dashboard.MachineStatus{
					ID:          s.ID,
					Description: s.Description,
					Status:      s.Status,
				})
			}

			allConfigs := cfg.GetAllConfigs()
			state := dashboard.State{
				Platform:       p,
				DriftSummary:   driftSummary,
				LinkStatus:     linkStatus,
				MachineStatus:  dashStatus,
				Configs:        allConfigs,
				Config:         cfg, // Full config for inline operations
				DotfilesPath:   dotfilesPath,
				UpdateMsg:      updateMsg,
				HasBaseline:    hasBaseline,
				HasConfig:      len(allConfigs) > 0,
				FilterText:     lastFilter,
				SelectedConfig: lastSelected,
			}
			result, err = dashboard.Run(state)
		}

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

// runNoConfigPrompt shows a polished prompt when no config exists
func runNoConfigPrompt() *dashboard.Result {
	cwd, _ := os.Getwd()

	ui.PrintBanner(Version)
	fmt.Printf("\n  No .go4dot.yaml found in %s\n\n", filepath.Base(cwd))

	var initHere bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Would you like to initialize go4dot here?").
				Description("This will scan for configs and create a .go4dot.yaml file").
				Affirmative("Yes, set up go4dot").
				Negative("No, quit").
				Value(&initHere),
		),
	)

	if err := form.Run(); err != nil {
		return &dashboard.Result{Action: dashboard.ActionQuit}
	}

	if initHere {
		return &dashboard.Result{Action: dashboard.ActionInit}
	}
	return &dashboard.Result{Action: dashboard.ActionQuit}
}

// handleAction processes the user's action and returns true if we should exit
func handleAction(result *dashboard.Result, cfg *config.Config, configPath string) bool {
	switch result.Action {
	case dashboard.ActionQuit:
		return true

	case dashboard.ActionInit:
		initCmd.Run(initCmd, nil)

		// Check if config now exists and prompt for install
		if newCfg, newConfigPath, err := config.LoadFromDiscovery(); err == nil {
			var runInstall bool
			form := huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title("Would you like to run install now?").
						Description("This will set up dependencies and clone external repos").
						Affirmative("Yes, install").
						Negative("No, skip").
						Value(&runInstall),
				),
			)
			if form.Run() == nil && runInstall {
				// Check for conflicts before install
				dotfilesPath := filepath.Dir(newConfigPath)
				conflicts, err := stow.DetectConflicts(newCfg, dotfilesPath)
				if err != nil {
					ui.Error("Failed to check conflicts: %v", err)
				} else if len(conflicts) > 0 {
					// Resolve conflicts before proceeding
					if !stow.ResolveConflicts(conflicts) {
						fmt.Println("  Install cancelled.")
						waitForEnter()
						return false
					}
				}
				// Run install within dashboard
				runInstallInDashboard(newCfg, dotfilesPath)
			}
		}

	case dashboard.ActionSync:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			runSyncInDashboard(cfg, dotfilesPath, "", nil)
		}

	case dashboard.ActionSyncConfig:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			runSyncInDashboard(cfg, dotfilesPath, result.ConfigName, nil)
		}

	case dashboard.ActionBulkSync:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			runSyncInDashboard(cfg, dotfilesPath, "", result.ConfigNames)
		}

	case dashboard.ActionDoctor:
		doctorCmd.Run(doctorCmd, nil)
		waitForEnter()

	case dashboard.ActionMachineConfig:
		machine.RunInteractiveConfig(cfg)
		waitForEnter()

	case dashboard.ActionInstall:
		if cfg != nil && configPath != "" {
			// Check for conflicts before install
			dotfilesPath := filepath.Dir(configPath)
			conflicts, err := stow.DetectConflicts(cfg, dotfilesPath)
			if err != nil {
				ui.Error("Failed to check conflicts: %v", err)
			} else if len(conflicts) > 0 {
				// Resolve conflicts before proceeding
				if !stow.ResolveConflicts(conflicts) {
					fmt.Println("  Install cancelled.")
					waitForEnter()
					return false
				}
			}
			runInstallInDashboard(cfg, dotfilesPath)
		}

	case dashboard.ActionUpdate:
		if cfg != nil && configPath != "" {
			dotfilesPath := filepath.Dir(configPath)
			runUpdateInDashboard(cfg, dotfilesPath)
		}

	case dashboard.ActionList:
		// This is the "More" menu
		runMoreMenu(cfg, configPath)
	}

	return false
}

// buildDashboardState creates a dashboard.State with current system information
func buildDashboardState(cfg *config.Config, dotfilesPath string) dashboard.State {
	p, _ := platform.Detect()

	driftSummary, _ := stow.FullDriftCheck(cfg, dotfilesPath)
	linkStatus, _ := stow.GetAllConfigLinkStatus(cfg, dotfilesPath)
	machineStatus := machine.CheckMachineConfigStatus(cfg)

	dashStatus := make([]dashboard.MachineStatus, 0, len(machineStatus))
	for _, s := range machineStatus {
		dashStatus = append(dashStatus, dashboard.MachineStatus{
			ID:          s.ID,
			Description: s.Description,
			Status:      s.Status,
		})
	}

	return dashboard.State{
		Platform:      p,
		DriftSummary:  driftSummary,
		LinkStatus:    linkStatus,
		MachineStatus: dashStatus,
		Configs:       cfg.GetAllConfigs(),
		Config:        cfg, // Full config for inline operations
		DotfilesPath:  dotfilesPath,
		HasConfig:     true,
	}
}

// runInstallInDashboard runs the install operation within the dashboard UI
func runInstallInDashboard(cfg *config.Config, dotfilesPath string) {
	state := buildDashboardState(cfg, dotfilesPath)

	opts := dashboard.InstallOptions{
		Auto: !ui.IsInteractive(),
	}

	_, err := dashboard.RunWithOperation(state, dashboard.OpInstall, "", nil, func(runner *dashboard.OperationRunner) error {
		_, err := dashboard.RunInstallOperation(runner, cfg, dotfilesPath, opts)
		return err
	})

	if err != nil {
		ui.Error("Dashboard error: %v", err)
	}
}

// runSyncInDashboard runs sync operations within the dashboard UI
func runSyncInDashboard(cfg *config.Config, dotfilesPath string, configName string, configNames []string) {
	state := buildDashboardState(cfg, dotfilesPath)

	opts := dashboard.SyncOptions{
		Force:       false, // Don't force by default; only force when explicitly requested
		Interactive: true,  // Enable interactive conflict resolution in dashboard
	}

	var opType dashboard.OperationType
	if len(configNames) > 0 {
		opType = dashboard.OpBulkSync
	} else if configName != "" {
		opType = dashboard.OpSyncSingle
	} else {
		opType = dashboard.OpSync
	}

	_, err := dashboard.RunWithOperation(state, opType, configName, configNames, func(runner *dashboard.OperationRunner) error {
		if len(configNames) > 0 {
			_, err := dashboard.RunBulkSyncOperation(runner, cfg, dotfilesPath, configNames, opts)
			return err
		} else if configName != "" {
			_, err := dashboard.RunSyncSingleOperation(runner, cfg, dotfilesPath, configName, opts)
			return err
		} else {
			_, err := dashboard.RunSyncAllOperation(runner, cfg, dotfilesPath, opts)
			return err
		}
	})

	if err != nil {
		ui.Error("Dashboard error: %v", err)
	}
}

// runUpdateInDashboard runs the update operation within the dashboard UI
func runUpdateInDashboard(cfg *config.Config, dotfilesPath string) {
	state := buildDashboardState(cfg, dotfilesPath)

	opts := dashboard.UpdateOptions{
		UpdateExternal: true,
	}

	_, err := dashboard.RunWithOperation(state, dashboard.OpUpdate, "", nil, func(runner *dashboard.OperationRunner) error {
		_, err := dashboard.RunUpdateOperation(runner, cfg, dotfilesPath, opts)
		return err
	})

	if err != nil {
		ui.Error("Dashboard error: %v", err)
	}
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
