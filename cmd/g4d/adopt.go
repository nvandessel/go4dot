package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/state"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
	"github.com/spf13/cobra"
)

var adoptCmd = &cobra.Command{
	Use:   "adopt [config-path]",
	Short: "Adopt existing stow symlinks into go4dot state",
	Long: `Scan for existing symlinks and update go4dot state to reflect current reality.

This is useful when:
  - You have existing stow symlinks but go4dot state is empty/missing
  - You set up symlinks manually and want go4dot to track them
  - You're migrating from another dotfiles manager

The command will:
  1. Scan each config in .go4dot.yaml
  2. Check which files are already correctly symlinked
  3. Update go4dot state for configs that are fully linked
  4. Report any partially-linked or missing configs`,
	Args: cobra.MaximumNArgs(1),
	Run:  runAdopt,
}

func init() {
	rootCmd.AddCommand(adoptCmd)

	adoptCmd.Flags().Bool("dry-run", false, "Show what would be adopted without changing state")
	adoptCmd.Flags().Bool("force", false, "Also adopt partially-linked configs")
}

func runAdopt(cmd *cobra.Command, args []string) {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	// Load config
	var cfg *config.Config
	var configPath string
	var err error

	if len(args) > 0 {
		cfg, err = config.LoadFromPath(args[0])
		configPath = args[0]
	} else {
		cfg, configPath, err = config.LoadFromDiscovery()
	}

	if err != nil {
		ui.Error("Error loading config: %v", err)
		os.Exit(1)
	}

	dotfilesPath := filepath.Dir(configPath)

	// Scan for existing symlinks
	fmt.Println("Scanning for existing symlinks...")
	fmt.Println()

	summary, err := stow.ScanExistingSymlinks(cfg, dotfilesPath)
	if err != nil {
		ui.Error("Error scanning symlinks: %v", err)
		os.Exit(1)
	}

	// Display results
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor).Bold(true)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor).Bold(true)
	subtleStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	// Fully linked
	fullyLinked := summary.GetFullyLinked()
	if len(fullyLinked) > 0 {
		fmt.Println(okStyle.Render("Fully Linked (will be adopted):"))
		for _, r := range fullyLinked {
			fmt.Printf("  %s %s (%d/%d files)\n",
				okStyle.Render("✓"),
				r.ConfigName,
				len(r.LinkedFiles),
				r.TotalFiles,
			)
		}
		fmt.Println()
	}

	// Partially linked
	partiallyLinked := summary.GetPartiallyLinked()
	if len(partiallyLinked) > 0 {
		if force {
			fmt.Println(warnStyle.Render("Partially Linked (will be adopted with --force):"))
		} else {
			fmt.Println(warnStyle.Render("Partially Linked:"))
		}
		for _, r := range partiallyLinked {
			fmt.Printf("  %s %s (%d/%d files)\n",
				warnStyle.Render("⚠"),
				r.ConfigName,
				len(r.LinkedFiles),
				r.TotalFiles,
			)
			// Show first few missing files
			for i, f := range r.MissingFiles {
				if i >= 3 {
					fmt.Printf("      %s\n", subtleStyle.Render(fmt.Sprintf("... and %d more", len(r.MissingFiles)-3)))
					break
				}
				fmt.Printf("      %s\n", subtleStyle.Render("Missing: "+f))
			}
		}
		fmt.Println()
	}

	// Not linked
	notLinked := summary.GetNotLinked()
	if len(notLinked) > 0 {
		fmt.Println(errStyle.Render("Not Linked:"))
		for _, r := range notLinked {
			fmt.Printf("  %s %s (0/%d files)\n",
				errStyle.Render("✗"),
				r.ConfigName,
				r.TotalFiles,
			)
		}
		fmt.Println()
	}

	// Determine what to adopt
	toAdopt := len(fullyLinked)
	if force {
		toAdopt += len(partiallyLinked)
	}

	if toAdopt == 0 {
		fmt.Println("No configs to adopt.")
		if len(partiallyLinked) > 0 {
			fmt.Println(subtleStyle.Render("Use --force to adopt partially-linked configs."))
		}
		return
	}

	if dryRun {
		fmt.Printf("Would adopt %d config(s).\n", toAdopt)
		fmt.Println(subtleStyle.Render("Run without --dry-run to apply changes."))
		return
	}

	// Confirm
	fmt.Printf("Adopt %d config(s)? [Y/n] ", toAdopt)
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "" && response != "y" && response != "yes" {
		fmt.Println("Aborted.")
		return
	}

	// Load or create state
	st, err := state.Load()
	if err != nil {
		ui.Error("Error loading state: %v", err)
		os.Exit(1)
	}
	if st == nil {
		st = state.New()
		st.DotfilesPath = dotfilesPath
	}

	// Perform adoption
	_, err = stow.AdoptExistingSymlinks(cfg, dotfilesPath, st, force)
	if err != nil {
		ui.Error("Error adopting symlinks: %v", err)
		os.Exit(1)
	}

	fmt.Println()
	ui.Success("Adopted %d config(s) into go4dot state.", toAdopt)
	fmt.Println(subtleStyle.Render("Run 'g4d doctor' to verify your installation."))
}
