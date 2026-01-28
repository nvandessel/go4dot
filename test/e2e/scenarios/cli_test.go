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
	// Check if UPDATE_GOLDEN env var is set
	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"

	projectRoot := getProjectRoot(t)

	cfg := helpers.VHSConfig{
		TapePath:       filepath.Join(projectRoot, "test/e2e/tapes/dashboard_navigation.tape"),
		OutputPath:     filepath.Join(projectRoot, "test/e2e/golden/cli_help.txt"),
		GoldenPath:     filepath.Join(projectRoot, "test/e2e/golden/cli_help.txt"),
		UpdateGolden:   updateGolden,
		ScreenshotPath: filepath.Join(projectRoot, "test/e2e/screenshots/cli_help.png"),
	}

	if err := helpers.RunVHSTape(t, cfg); err != nil {
		t.Fatalf("VHS test failed: %v", err)
	}

	// Cleanup is handled by t.Cleanup automatically
	t.Cleanup(func() {
		if !updateGolden {
			// Only cleanup temp outputs, keep golden files
			helpers.CleanupVHSOutputs(
				filepath.Join(projectRoot, "test/e2e/screenshots/cli_help.png"),
			)
		}
	})
}
