package main

import (
	"fmt"
	"os"

	"github.com/nvandessel/go4dot/internal/status"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show dotfiles status overview",
	Long: `Display a quick overview of your dotfiles status.

Shows platform info, config sync status, dependency health, and last sync time.
Suitable for scripting with the --json flag.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonOutput, _ := cmd.Flags().GetBool("json")
		skipDeps, _ := cmd.Flags().GetBool("skip-deps")
		skipDrift, _ := cmd.Flags().GetBool("skip-drift")

		gatherer := status.NewGatherer()
		overview, err := gatherer.Gather(status.GatherOptions{
			SkipDrift: skipDrift,
			SkipDeps:  skipDeps,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		output, err := status.Render(overview, status.RenderOptions{
			JSON: jsonOutput,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)

	statusCmd.Flags().Bool("json", false, "Output status as JSON")
	statusCmd.Flags().Bool("skip-deps", false, "Skip dependency checking (faster)")
	statusCmd.Flags().Bool("skip-drift", false, "Skip drift detection (faster)")
}
