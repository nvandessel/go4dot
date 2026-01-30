// Package main provides a CLI orchestrator for running E2E tests across different test suites.
package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version information
	version = "dev"

	// CLI flags
	suiteFlag      string
	parallelFlag   int
	updateGolden   bool
	outputFormat   string
	verboseFlag    bool
	timeoutSeconds int
)

var rootCmd = &cobra.Command{
	Use:   "orchestrator",
	Short: "E2E test orchestrator for go4dot",
	Long: `The E2E test orchestrator coordinates running different types of end-to-end tests:

Test Suites:
  visual  - VHS-based visual regression tests (golden file comparisons)
  docker  - Docker/Podman container-based integration tests
  tui     - Headless TUI tests using teatest framework
  all     - Run all test suites

Examples:
  # Run all tests
  orchestrator

  # Run only Docker tests
  orchestrator --suite=docker

  # Run visual tests and update golden files
  orchestrator --suite=visual --update-golden

  # Run all tests with 4 parallel workers
  orchestrator --parallel=4

  # Output results as JSON
  orchestrator --format=json`,
	RunE: runOrchestrator,
}

func init() {
	rootCmd.Flags().StringVarP(&suiteFlag, "suite", "s", "all", "Test suite to run: visual, docker, tui, or all")
	rootCmd.Flags().IntVarP(&parallelFlag, "parallel", "p", runtime.NumCPU(), "Number of parallel test workers")
	rootCmd.Flags().BoolVar(&updateGolden, "update-golden", false, "Update golden files for visual tests")
	rootCmd.Flags().StringVarP(&outputFormat, "format", "f", "text", "Output format: text or json")
	rootCmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntVarP(&timeoutSeconds, "timeout", "t", 600, "Timeout in seconds for entire test run")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(listCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("orchestrator %s\n", version)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available test suites and their tests",
	RunE:  listSuites,
}

func runOrchestrator(cmd *cobra.Command, args []string) error {
	// Validate suite flag
	validSuites := map[string]bool{
		"visual": true,
		"docker": true,
		"tui":    true,
		"all":    true,
	}

	if !validSuites[suiteFlag] {
		return fmt.Errorf("invalid suite: %s (valid options: visual, docker, tui, all)", suiteFlag)
	}

	// Validate output format
	if outputFormat != "text" && outputFormat != "json" {
		return fmt.Errorf("invalid format: %s (valid options: text, json)", outputFormat)
	}

	// Create runner configuration
	cfg := RunnerConfig{
		Suite:        suiteFlag,
		Parallel:     parallelFlag,
		UpdateGolden: updateGolden,
		Verbose:      verboseFlag,
		Timeout:      timeoutSeconds,
	}

	// Create runner and execute tests
	runner := NewRunner(cfg)
	results, err := runner.Run()
	if err != nil {
		return fmt.Errorf("test execution failed: %w", err)
	}

	// Create reporter and output results
	reporter := NewReporter(outputFormat, verboseFlag)
	if err := reporter.Report(results); err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// Return error if any tests failed
	if results.Failed > 0 {
		return fmt.Errorf("tests failed: %d/%d", results.Failed, results.Total)
	}

	return nil
}

func listSuites(cmd *cobra.Command, args []string) error {
	suites := []struct {
		Name        string
		Description string
		Tags        string
	}{
		{
			Name:        "visual",
			Description: "VHS-based visual regression tests",
			Tags:        "e2e",
		},
		{
			Name:        "docker",
			Description: "Docker/Podman container integration tests",
			Tags:        "e2e",
		},
		{
			Name:        "tui",
			Description: "Headless TUI tests using teatest",
			Tags:        "e2e",
		},
	}

	fmt.Println("Available test suites:")
	fmt.Println()

	for _, s := range suites {
		fmt.Printf("  %s\n", s.Name)
		fmt.Printf("    Description: %s\n", s.Description)
		fmt.Printf("    Build tags:  %s\n", s.Tags)
		fmt.Println()
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
