//go:build e2e

package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/test/e2e/helpers"
)

// getProjectRoot returns the absolute path to the project root
func getProjectRoot(t *testing.T) string {
	t.Helper()
	// From test/e2e/scenarios, go up 3 levels to reach project root
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return filepath.Join(cwd, "..", "..", "..")
}

// TestCLIHelp validates that g4d --help output matches expected golden file
// This is a foundational test to ensure the VHS infrastructure works correctly
func TestCLIHelp(t *testing.T) {
	tests := []struct {
		name       string
		tapePath   string
		outputPath string
		goldenPath string
	}{
		{
			name:       "cli help output",
			tapePath:   "test/e2e/tapes/cli_help.tape",
			outputPath: "test/e2e/outputs/cli_help.txt",
			goldenPath: "test/e2e/golden/cli_help.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check if UPDATE_GOLDEN env var is set
			updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

			projectRoot := getProjectRoot(t)

			cfg := helpers.VHSConfig{
				TapePath:     filepath.Join(projectRoot, tt.tapePath),
				OutputPath:   filepath.Join(projectRoot, tt.outputPath),
				GoldenPath:   filepath.Join(projectRoot, tt.goldenPath),
				UpdateGolden: updateGolden,
			}

			if err := helpers.RunVHSTape(t, cfg); err != nil {
				t.Fatalf("VHS test failed: %v", err)
			}

			// Cleanup test outputs (not golden files)
			t.Cleanup(func() {
				if !updateGolden {
					helpers.CleanupVHSOutputs(
						filepath.Join(projectRoot, tt.outputPath),
					)
				}
			})
		})
	}
}
