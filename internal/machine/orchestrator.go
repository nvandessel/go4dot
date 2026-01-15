package machine

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/ui"
)

// RunInteractiveConfig handles machine configuration interactively.
func RunInteractiveConfig(cfg *config.Config) {
	if len(cfg.MachineConfig) == 0 {
		ui.Warning("No machine configurations defined in .go4dot.yaml")
		return
	}

	// Show current status
	statuses := CheckMachineConfigStatus(cfg)
	fmt.Println("\nMachine Configuration Status")
	fmt.Println("----------------------------")

	options := []huh.Option[string]{}
	options = append(options, huh.NewOption("Configure All", "all"))

	for _, s := range statuses {
		statusIcon := " "
		switch s.Status {
		case "configured":
			statusIcon = "+"
		case "missing":
			statusIcon = "x"
		case "error":
			statusIcon = "!"
		}

		label := fmt.Sprintf("%s %s (%s)", statusIcon, s.Description, s.ID)
		options = append(options, huh.NewOption(label, s.ID))
	}

	options = append(options, huh.NewOption("Back", "back"))

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select configuration to update").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return
	}

	if selected == "back" {
		return
	}

	promptOpts := PromptOptions{
		ProgressFunc: func(current, total int, msg string) {
			fmt.Println(msg)
		},
	}

	renderOpts := RenderOptions{
		Overwrite: true,
		ProgressFunc: func(current, total int, msg string) {
			fmt.Println(msg)
		},
	}

	if selected == "all" {
		fmt.Printf("\nConfiguring %d machine settings...\n\n", len(cfg.MachineConfig))
		results, err := CollectMachineConfig(cfg, promptOpts)
		if err != nil {
			ui.Error("Error: %v", err)
			return
		}

		_, err = RenderAll(cfg, results, renderOpts)
		if err != nil {
			ui.Error("Error: %v", err)
			return
		}
	} else {
		// Configure single
		fmt.Printf("\nConfiguring %s...\n\n", selected)
		result, err := CollectSingleConfig(cfg, selected, promptOpts)
		if err != nil {
			ui.Error("Error: %v", err)
			return
		}

		mc := GetMachineConfigByID(cfg, selected)
		_, err = RenderAndWrite(mc, result.Values, renderOpts)
		if err != nil {
			ui.Error("Error: %v", err)
			return
		}
	}

	ui.Success("Configuration complete")
}
