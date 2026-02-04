//go:build e2e

// Package scenarios contains visual E2E tests using VHS for TUI screenshot regression.
//
// These tests run VHS tapes inside Docker containers for complete isolation.
// Screenshots are saved to test/e2e/screenshots/ for multimodal review.
// Text output is compared against golden files in test/e2e/golden/.
package scenarios

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nvandessel/go4dot/test/e2e/helpers"
)

// vhsTestCase defines a VHS-based visual test
type vhsTestCase struct {
	name       string
	tapePath   string
	outputPath string
	goldenPath string
}

// TestVHS_DashboardStartup validates that the dashboard renders correctly on startup
func TestVHS_DashboardStartup(t *testing.T) {
	runVHSTest(t, vhsTestCase{
		name:       "dashboard startup",
		tapePath:   "test/e2e/tapes/dashboard_startup.tape",
		outputPath: "test/e2e/outputs/dashboard_startup.txt",
		goldenPath: "test/e2e/golden/dashboard_startup.txt",
	})
}

// TestVHS_DashboardNavigation validates panel navigation with Tab and number keys
func TestVHS_DashboardNavigation(t *testing.T) {
	runVHSTest(t, vhsTestCase{
		name:       "dashboard navigation",
		tapePath:   "test/e2e/tapes/dashboard_navigation.tape",
		outputPath: "test/e2e/outputs/dashboard_navigation.txt",
		goldenPath: "test/e2e/golden/dashboard_navigation.txt",
	})
}

// TestVHS_HealthPanel validates health panel focus and navigation
func TestVHS_HealthPanel(t *testing.T) {
	runVHSTest(t, vhsTestCase{
		name:       "health panel",
		tapePath:   "test/e2e/tapes/health_panel.tape",
		outputPath: "test/e2e/outputs/health_panel.txt",
		goldenPath: "test/e2e/golden/health_panel.txt",
	})
}

// TestVHS_FilterSelection validates filter mode and multi-select functionality
func TestVHS_FilterSelection(t *testing.T) {
	runVHSTest(t, vhsTestCase{
		name:       "filter and selection",
		tapePath:   "test/e2e/tapes/filter_selection.tape",
		outputPath: "test/e2e/outputs/filter_selection.txt",
		goldenPath: "test/e2e/golden/filter_selection.txt",
	})
}

// TestVHS_ConflictResolution validates that conflict resolution modal appears
// when installing with existing files that would conflict
func TestVHS_ConflictResolution(t *testing.T) {
	runVHSTest(t, vhsTestCase{
		name:       "conflict resolution",
		tapePath:   "test/e2e/tapes/conflict_resolution.tape",
		outputPath: "test/e2e/outputs/conflict_resolution.txt",
		goldenPath: "test/e2e/golden/conflict_resolution.txt",
	})
}

// runVHSTest executes a single VHS test case
func runVHSTest(t *testing.T, tc vhsTestCase) {
	t.Helper()

	updateGolden := os.Getenv("UPDATE_GOLDEN") == "1"
	projectRoot := getProjectRoot(t)

	// Build the test binary for Linux
	binaryPath := helpers.BuildTestBinary(t)

	// Copy fixtures to container
	fixturesDir := filepath.Join(projectRoot, "test/e2e/fixtures/dotfiles")

	// Create a VHS-enabled Docker container
	container := helpers.NewDockerTestContainer(t, helpers.DockerConfig{
		BinaryPath:  binaryPath,
		FixturesDir: fixturesDir,
		VHSEnabled:  true,
	})

	cfg := helpers.VHSConfig{
		TapePath:      filepath.Join(projectRoot, tc.tapePath),
		OutputPath:    filepath.Join(projectRoot, tc.outputPath),
		GoldenPath:    filepath.Join(projectRoot, tc.goldenPath),
		ScreenshotDir: filepath.Join(projectRoot, "test/e2e/screenshots"),
		UpdateGolden:  updateGolden,
	}

	if err := helpers.RunVHSTapeInContainer(t, container, cfg); err != nil {
		t.Fatalf("VHS test failed: %v", err)
	}

	// Cleanup test outputs (not golden files)
	t.Cleanup(func() {
		if !updateGolden {
			helpers.CleanupVHSOutputs(
				filepath.Join(projectRoot, tc.outputPath),
			)
		}
	})
}
