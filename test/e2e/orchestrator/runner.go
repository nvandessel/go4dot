package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// RunnerConfig holds configuration for test execution.
type RunnerConfig struct {
	Suite        string
	Parallel     int
	UpdateGolden bool
	Verbose      bool
	Timeout      int
}

// Runner executes test suites.
type Runner struct {
	config      RunnerConfig
	projectRoot string
}

// TestResult represents the result of a single test.
type TestResult struct {
	Name     string        `json:"name"`
	Suite    string        `json:"suite"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// TestResults aggregates results from all tests.
type TestResults struct {
	Suite    string        `json:"suite"`
	Total    int           `json:"total"`
	Passed   int           `json:"passed"`
	Failed   int           `json:"failed"`
	Skipped  int           `json:"skipped"`
	Duration time.Duration `json:"duration"`
	Tests    []TestResult  `json:"tests"`
}

// NewRunner creates a new test runner with the given configuration.
func NewRunner(cfg RunnerConfig) *Runner {
	return &Runner{
		config:      cfg,
		projectRoot: findProjectRoot(),
	}
}

// Run executes the configured test suite and returns results.
func (r *Runner) Run() (*TestResults, error) {
	// Validate config to avoid zero-length semaphore or immediate timeout
	if r.config.Parallel <= 0 {
		r.config.Parallel = 1
	}
	if r.config.Timeout <= 0 {
		r.config.Timeout = 300 // Default 5 minutes
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.config.Timeout)*time.Second)
	defer cancel()

	startTime := time.Now()

	results := &TestResults{
		Suite: r.config.Suite,
		Tests: []TestResult{},
	}

	suites := r.getSuitesToRun()

	// Run suites
	for _, suite := range suites {
		suiteResults, err := r.runSuite(ctx, suite)
		if err != nil {
			// Context cancelled or timeout
			if ctx.Err() != nil {
				return results, fmt.Errorf("test run timed out after %d seconds", r.config.Timeout)
			}
			// Add failed result for the suite
			results.Tests = append(results.Tests, TestResult{
				Name:   suite,
				Suite:  suite,
				Passed: false,
				Error:  err.Error(),
			})
			results.Failed++
			results.Total++
			continue
		}

		results.Tests = append(results.Tests, suiteResults...)
		for _, tr := range suiteResults {
			results.Total++
			if tr.Passed {
				results.Passed++
			} else if tr.Error == "skipped" {
				results.Skipped++
			} else {
				results.Failed++
			}
		}
	}

	results.Duration = time.Since(startTime)

	return results, nil
}

// getSuitesToRun returns the list of suites to execute based on configuration.
func (r *Runner) getSuitesToRun() []string {
	if r.config.Suite == "all" {
		return []string{"visual", "docker", "tui"}
	}
	return []string{r.config.Suite}
}

// runSuite executes a specific test suite.
func (r *Runner) runSuite(ctx context.Context, suite string) ([]TestResult, error) {
	switch suite {
	case "visual":
		return r.runVisualTests(ctx)
	case "docker":
		return r.runDockerTests(ctx)
	case "tui":
		return r.runTUITests(ctx)
	default:
		return nil, fmt.Errorf("unknown suite: %s", suite)
	}
}

// runVisualTests executes VHS-based visual regression tests.
func (r *Runner) runVisualTests(ctx context.Context) ([]TestResult, error) {
	// Check if VHS is installed
	if _, err := exec.LookPath("vhs"); err != nil {
		return []TestResult{{
			Name:   "visual",
			Suite:  "visual",
			Passed: false,
			Error:  "skipped",
			Output: "VHS not installed, skipping visual tests",
		}}, nil
	}

	// Find VHS test files
	testFiles, err := r.findTestFiles("scenarios", "cli_test.go")
	if err != nil {
		return nil, fmt.Errorf("failed to find visual test files: %w", err)
	}

	if len(testFiles) == 0 {
		return []TestResult{{
			Name:   "visual",
			Suite:  "visual",
			Passed: true,
			Output: "No visual test files found",
		}}, nil
	}

	// Build environment for tests
	env := os.Environ()
	if r.config.UpdateGolden {
		env = append(env, "UPDATE_GOLDEN=1")
	}

	return r.runGoTests(ctx, testFiles, "visual", env)
}

// runDockerTests executes Docker/Podman container-based tests.
func (r *Runner) runDockerTests(ctx context.Context) ([]TestResult, error) {
	// Check if Docker or Podman is available (with timeout to avoid hanging)
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	dockerAvailable := exec.CommandContext(checkCtx, "docker", "info").Run() == nil
	podmanAvailable := exec.CommandContext(checkCtx, "podman", "info").Run() == nil

	if !dockerAvailable && !podmanAvailable {
		return []TestResult{{
			Name:   "docker",
			Suite:  "docker",
			Passed: false,
			Error:  "skipped",
			Output: "No container runtime found (docker or podman)",
		}}, nil
	}

	// Find Docker test files (doctor_checks_test.go, install_flow_test.go)
	testFiles, err := r.findTestFiles("scenarios", "doctor_checks_test.go", "install_flow_test.go")
	if err != nil {
		return nil, fmt.Errorf("failed to find docker test files: %w", err)
	}

	if len(testFiles) == 0 {
		return []TestResult{{
			Name:   "docker",
			Suite:  "docker",
			Passed: true,
			Output: "No Docker test files found",
		}}, nil
	}

	return r.runGoTests(ctx, testFiles, "docker", os.Environ())
}

// runTUITests executes headless TUI tests using teatest.
func (r *Runner) runTUITests(ctx context.Context) ([]TestResult, error) {
	// Find TUI test files
	testFiles, err := r.findTestFiles("scenarios", "dashboard_tui_test.go")
	if err != nil {
		return nil, fmt.Errorf("failed to find TUI test files: %w", err)
	}

	if len(testFiles) == 0 {
		return []TestResult{{
			Name:   "tui",
			Suite:  "tui",
			Passed: true,
			Output: "No TUI test files found",
		}}, nil
	}

	return r.runGoTests(ctx, testFiles, "tui", os.Environ())
}

// runGoTests executes Go tests for the specified files.
func (r *Runner) runGoTests(ctx context.Context, files []string, suite string, env []string) ([]TestResult, error) {
	results := []TestResult{}
	resultsChan := make(chan TestResult, len(files))

	// Create a wait group for parallel execution
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, r.config.Parallel)

	for _, file := range files {
		wg.Add(1)
		go func(testFile string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result := r.runSingleTestFile(ctx, testFile, suite, env)
			resultsChan <- result
		}(file)
	}

	// Wait for all tests to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	for result := range resultsChan {
		results = append(results, result)
	}

	return results, nil
}

// runSingleTestFile executes a single test file.
func (r *Runner) runSingleTestFile(ctx context.Context, file, suite string, env []string) TestResult {
	startTime := time.Now()

	// Get the directory containing the test file
	testDir := filepath.Dir(file)
	testFileName := filepath.Base(file)

	// Build the go test command
	args := []string{"test", "-v", "-tags=e2e", "-timeout", fmt.Sprintf("%ds", r.config.Timeout)}

	// Add parallel flag
	if r.config.Parallel > 1 {
		args = append(args, fmt.Sprintf("-parallel=%d", r.config.Parallel))
	}

	// Run specific test file
	args = append(args, "-run", testFileToPattern(testFileName), ".")

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = testDir
	cmd.Env = env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	result := TestResult{
		Name:     testFileName,
		Suite:    suite,
		Duration: duration,
	}

	combinedOutput := stdout.String() + stderr.String()
	if r.config.Verbose {
		result.Output = combinedOutput
	}

	if err != nil {
		result.Passed = false
		result.Error = err.Error()
		if !r.config.Verbose {
			// Include output on failure even if not verbose
			result.Output = combinedOutput
		}
	} else {
		result.Passed = true
	}

	return result
}

// findTestFiles finds test files in the e2e test directory.
func (r *Runner) findTestFiles(subdir string, patterns ...string) ([]string, error) {
	e2eDir := filepath.Join(r.projectRoot, "test", "e2e", subdir)

	var files []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(e2eDir, pattern))
		if err != nil {
			return nil, fmt.Errorf("glob %q: %w", pattern, err)
		}
		files = append(files, matches...)
	}

	return files, nil
}

// testFileToPattern converts a test file name to a test pattern.
// We use a broad "Test" pattern since Go test execution is already scoped
// to the package directory, and attempting to derive specific prefixes from
// filenames is fragile (e.g., "cli_test.go" doesn't match "TestCLI*").
func testFileToPattern(_ string) string {
	return "Test"
}

// findProjectRoot finds the project root by looking for go.mod.
func findProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}

	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return cwd
		}
		dir = parent
	}
}
