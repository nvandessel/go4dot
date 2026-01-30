package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Reporter handles formatting and outputting test results.
type Reporter struct {
	format  string
	verbose bool
	output  io.Writer
}

// NewReporter creates a new reporter with the specified format.
func NewReporter(format string, verbose bool) *Reporter {
	return &Reporter{
		format:  format,
		verbose: verbose,
		output:  os.Stdout,
	}
}

// Report outputs the test results in the configured format.
func (r *Reporter) Report(results *TestResults) error {
	switch r.format {
	case "json":
		return r.reportJSON(results)
	case "text":
		return r.reportText(results)
	default:
		return fmt.Errorf("unknown format: %s", r.format)
	}
}

// reportJSON outputs results in JSON format.
func (r *Reporter) reportJSON(results *TestResults) error {
	encoder := json.NewEncoder(r.output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// reportText outputs results in human-readable text format.
func (r *Reporter) reportText(results *TestResults) error {
	r.printHeader(results)
	r.printTestResults(results)
	r.printSummary(results)

	return nil
}

// printf is a helper that writes formatted output, ignoring errors.
// This is standard practice for console output where write errors aren't recoverable.
func (r *Reporter) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(r.output, format, args...)
}

// printHeader prints the test run header.
func (r *Reporter) printHeader(results *TestResults) {
	r.printf("\n")
	r.printf("================================================================================\n")
	r.printf("  E2E Test Results - Suite: %s\n", results.Suite)
	r.printf("================================================================================\n")
	r.printf("\n")
}

// printTestResults prints individual test results.
func (r *Reporter) printTestResults(results *TestResults) {
	for _, test := range results.Tests {
		r.printTestResult(test)
	}
	r.printf("\n")
}

// printTestResult prints a single test result.
func (r *Reporter) printTestResult(test TestResult) {
	statusIcon := r.getStatusIcon(test)
	statusText := r.getStatusText(test)

	r.printf("%s [%s] %s (%s)\n",
		statusIcon,
		test.Suite,
		test.Name,
		formatDuration(test.Duration),
	)

	// Print details if verbose or if test failed
	if r.verbose || !test.Passed {
		if test.Error != "" && test.Error != "skipped" {
			r.printf("    Error: %s\n", test.Error)
		}
		if test.Output != "" {
			r.printOutput(test.Output, statusText)
		}
	}
}

// getStatusIcon returns the appropriate icon for a test result.
func (r *Reporter) getStatusIcon(test TestResult) string {
	if test.Error == "skipped" {
		return "SKIP"
	}
	if test.Passed {
		return "PASS"
	}
	return "FAIL"
}

// getStatusText returns the status text for a test result.
func (r *Reporter) getStatusText(test TestResult) string {
	if test.Error == "skipped" {
		return "skipped"
	}
	if test.Passed {
		return "passed"
	}
	return "failed"
}

// printOutput prints test output with indentation.
func (r *Reporter) printOutput(output, status string) {
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Limit output lines for readability
	maxLines := 50
	if status == "failed" {
		maxLines = 100 // Show more lines for failures
	}

	if len(lines) > maxLines {
		r.printf("    Output (truncated to %d lines):\n", maxLines)
		lines = lines[:maxLines]
	} else if len(lines) > 0 {
		r.printf("    Output:\n")
	}

	for _, line := range lines {
		r.printf("      %s\n", line)
	}
}

// printSummary prints the test run summary.
func (r *Reporter) printSummary(results *TestResults) {
	r.printf("--------------------------------------------------------------------------------\n")
	r.printf("  Summary\n")
	r.printf("--------------------------------------------------------------------------------\n")

	r.printf("  Total:    %d\n", results.Total)
	r.printf("  Passed:   %d\n", results.Passed)
	r.printf("  Failed:   %d\n", results.Failed)
	r.printf("  Skipped:  %d\n", results.Skipped)
	r.printf("  Duration: %s\n", formatDuration(results.Duration))
	r.printf("\n")

	// Final status message
	if results.Failed > 0 {
		r.printf("  Status: FAILED\n")
	} else if results.Passed == 0 && results.Skipped > 0 {
		r.printf("  Status: SKIPPED (no tests ran)\n")
	} else {
		r.printf("  Status: PASSED\n")
	}
	r.printf("\n")
}

// formatDuration formats a duration for display.
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}

// JSONResult is a simplified result structure for JSON output.
type JSONResult struct {
	Suite    string       `json:"suite"`
	Status   string       `json:"status"`
	Total    int          `json:"total"`
	Passed   int          `json:"passed"`
	Failed   int          `json:"failed"`
	Skipped  int          `json:"skipped"`
	Duration string       `json:"duration"`
	Tests    []JSONTest   `json:"tests"`
}

// JSONTest represents a single test in JSON output.
type JSONTest struct {
	Name     string `json:"name"`
	Suite    string `json:"suite"`
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Error    string `json:"error,omitempty"`
	Output   string `json:"output,omitempty"`
}

// ToJSON converts TestResults to a JSON-friendly format.
func (results *TestResults) ToJSON() JSONResult {
	status := "passed"
	if results.Failed > 0 {
		status = "failed"
	} else if results.Passed == 0 && results.Skipped > 0 {
		status = "skipped"
	}

	tests := make([]JSONTest, len(results.Tests))
	for i, t := range results.Tests {
		testStatus := "passed"
		if t.Error == "skipped" {
			testStatus = "skipped"
		} else if !t.Passed {
			testStatus = "failed"
		}

		tests[i] = JSONTest{
			Name:     t.Name,
			Suite:    t.Suite,
			Status:   testStatus,
			Duration: formatDuration(t.Duration),
			Error:    t.Error,
			Output:   t.Output,
		}
	}

	return JSONResult{
		Suite:    results.Suite,
		Status:   status,
		Total:    results.Total,
		Passed:   results.Passed,
		Failed:   results.Failed,
		Skipped:  results.Skipped,
		Duration: formatDuration(results.Duration),
		Tests:    tests,
	}
}
