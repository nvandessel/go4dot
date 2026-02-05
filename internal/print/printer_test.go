package print

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the function
	f()

	// Close writer and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read the captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestSuccess(t *testing.T) {
	output := captureOutput(func() {
		Success("operation completed")
	})

	if !strings.Contains(output, "✓") {
		t.Error("Success() output should contain checkmark (✓)")
	}
	if !strings.Contains(output, "operation completed") {
		t.Error("Success() output should contain the message")
	}
}

func TestSuccessFormatted(t *testing.T) {
	output := captureOutput(func() {
		Success("processed %d items", 42)
	})

	if !strings.Contains(output, "42") {
		t.Error("Success() should support format strings")
	}
}

func TestError(t *testing.T) {
	output := captureOutput(func() {
		Error("something went wrong")
	})

	if !strings.Contains(output, "✖") {
		t.Error("Error() output should contain cross mark (✖)")
	}
	if !strings.Contains(output, "something went wrong") {
		t.Error("Error() output should contain the message")
	}
}

func TestErrorFormatted(t *testing.T) {
	output := captureOutput(func() {
		Error("failed with code %d: %s", 500, "server error")
	})

	if !strings.Contains(output, "500") {
		t.Error("Error() should support format strings")
	}
	if !strings.Contains(output, "server error") {
		t.Error("Error() should support format strings")
	}
}

func TestWarning(t *testing.T) {
	output := captureOutput(func() {
		Warning("this might be a problem")
	})

	if !strings.Contains(output, "⚠") {
		t.Error("Warning() output should contain warning sign (⚠)")
	}
	if !strings.Contains(output, "this might be a problem") {
		t.Error("Warning() output should contain the message")
	}
}

func TestWarningFormatted(t *testing.T) {
	output := captureOutput(func() {
		Warning("found %d issues", 3)
	})

	if !strings.Contains(output, "3") {
		t.Error("Warning() should support format strings")
	}
}

func TestInfo(t *testing.T) {
	output := captureOutput(func() {
		Info("for your information")
	})

	if !strings.Contains(output, "ℹ") {
		t.Error("Info() output should contain info sign (ℹ)")
	}
	if !strings.Contains(output, "for your information") {
		t.Error("Info() output should contain the message")
	}
}

func TestInfoFormatted(t *testing.T) {
	output := captureOutput(func() {
		Info("version %s", "1.0.0")
	})

	if !strings.Contains(output, "1.0.0") {
		t.Error("Info() should support format strings")
	}
}

func TestSection(t *testing.T) {
	output := captureOutput(func() {
		Section("Configuration")
	})

	if !strings.Contains(output, "Configuration") {
		t.Error("Section() output should contain the title")
	}
	// Section adds an empty line before the title
	if !strings.HasPrefix(output, "\n") {
		t.Error("Section() should start with a newline")
	}
}

func TestColors(t *testing.T) {
	// Test that color constants are defined
	if PrimaryColor == "" {
		t.Error("PrimaryColor should be defined")
	}
	if SecondaryColor == "" {
		t.Error("SecondaryColor should be defined")
	}
	if ErrorColor == "" {
		t.Error("ErrorColor should be defined")
	}
	if WarningColor == "" {
		t.Error("WarningColor should be defined")
	}
	if TextColor == "" {
		t.Error("TextColor should be defined")
	}
}

func TestStyles(t *testing.T) {
	// Test that styles are usable
	rendered := TitleStyle.Render("Title")
	if rendered == "" {
		t.Error("TitleStyle should render content")
	}

	rendered = ErrorStyle.Render("Error")
	if rendered == "" {
		t.Error("ErrorStyle should render content")
	}

	rendered = SuccessStyle.Render("Success")
	if rendered == "" {
		t.Error("SuccessStyle should render content")
	}
}
