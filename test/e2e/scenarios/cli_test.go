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
	return helpers.GetProjectRoot(t)
}

// TestCLIHelp validates that g4d --help output matches expected golden file
// This test runs VHS inside a Docker container for complete isolation
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

			// Build the test binary for Linux
			binaryPath := helpers.BuildTestBinary(t)

			// Create a VHS-enabled Docker container
			container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
				BinaryPath: binaryPath,
				VHSEnabled: true,
			})

			cfg := helpers.VHSConfig{
				TapePath:     filepath.Join(projectRoot, tt.tapePath),
				OutputPath:   filepath.Join(projectRoot, tt.outputPath),
				GoldenPath:   filepath.Join(projectRoot, tt.goldenPath),
				UpdateGolden: updateGolden,
			}

			if err := helpers.RunVHSTapeInContainer(t, container, cfg); err != nil {
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
