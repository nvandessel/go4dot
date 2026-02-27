package machine

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/validation"
)

// RenderResult holds the result of rendering a template
type RenderResult struct {
	ID          string
	Destination string
	Content     string
}

// RenderOptions configures template rendering
type RenderOptions struct {
	DryRun       bool                                 // Don't write files, just return content
	Overwrite    bool                                 // Overwrite existing files
	ProgressFunc func(current, total int, msg string) // Called for progress updates with item counts
}

// RenderMachineConfig renders a machine config template with the given values
func RenderMachineConfig(mc *config.MachinePrompt, values map[string]string) (*RenderResult, error) {
	// Parse the template
	tmpl, err := template.New(mc.ID).Parse(mc.Template)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	// Expand destination path
	dest, err := expandPath(mc.Destination)
	if err != nil {
		return nil, fmt.Errorf("failed to expand destination path: %w", err)
	}

	return &RenderResult{
		ID:          mc.ID,
		Destination: dest,
		Content:     buf.String(),
	}, nil
}

// RenderAndWrite renders a template and writes it to the destination
func RenderAndWrite(mc *config.MachinePrompt, values map[string]string, opts RenderOptions) (*RenderResult, error) {
	result, err := RenderMachineConfig(mc, values)
	if err != nil {
		return nil, err
	}

	if opts.ProgressFunc != nil {
		if opts.DryRun {
			opts.ProgressFunc(0, 0, fmt.Sprintf("Would write %s to %s", mc.ID, result.Destination))
		} else {
			opts.ProgressFunc(0, 0, fmt.Sprintf("Writing %s to %s", mc.ID, result.Destination))
		}
	}

	if opts.DryRun {
		return result, nil
	}

	// Check if file exists
	if _, err := os.Stat(result.Destination); err == nil && !opts.Overwrite {
		return nil, fmt.Errorf("file already exists: %s (use --overwrite to replace)", result.Destination)
	}

	// Create parent directory if needed
	parentDir := filepath.Dir(result.Destination)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", parentDir, err)
	}

	// Write the file
	if err := os.WriteFile(result.Destination, []byte(result.Content), 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, 0, fmt.Sprintf("✓ Created %s", result.Destination))
	}

	return result, nil
}

// RenderAll renders all machine configs with collected values
func RenderAll(cfg *config.Config, results []PromptResult, opts RenderOptions) ([]RenderResult, error) {
	var rendered []RenderResult

	for _, pr := range results {
		mc := GetMachineConfigByID(cfg, pr.ID)
		if mc == nil {
			return nil, fmt.Errorf("machine config '%s' not found", pr.ID)
		}

		result, err := RenderAndWrite(mc, pr.Values, opts)
		if err != nil {
			return rendered, fmt.Errorf("failed to render %s: %w", pr.ID, err)
		}
		rendered = append(rendered, *result)
	}

	return rendered, nil
}

// CheckMachineConfigStatus checks if machine config files exist
func CheckMachineConfigStatus(cfg *config.Config) []MachineConfigStatus {
	var statuses []MachineConfigStatus

	for _, mc := range cfg.MachineConfig {
		status := MachineConfigStatus{
			ID:          mc.ID,
			Description: mc.Description,
		}

		dest, err := expandPath(mc.Destination)
		if err != nil {
			status.Status = "error"
			status.Error = err.Error()
			statuses = append(statuses, status)
			continue
		}

		status.Destination = dest

		if _, err := os.Stat(dest); os.IsNotExist(err) {
			status.Status = "missing"
		} else if err != nil {
			status.Status = "error"
			status.Error = err.Error()
		} else {
			status.Status = "configured"
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// MachineConfigStatus represents the status of a machine config
type MachineConfigStatus struct {
	ID          string
	Description string
	Destination string
	Status      string // "configured", "missing", "error"
	Error       string
}

// RemoveMachineConfig removes a generated machine config file
func RemoveMachineConfig(mc *config.MachinePrompt, opts RenderOptions) error {
	dest, err := expandPath(mc.Destination)
	if err != nil {
		return fmt.Errorf("failed to expand path: %w", err)
	}

	if _, err := os.Stat(dest); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", dest)
	}

	if opts.ProgressFunc != nil {
		if opts.DryRun {
			opts.ProgressFunc(0, 0, fmt.Sprintf("Would remove %s", dest))
		} else {
			opts.ProgressFunc(0, 0, fmt.Sprintf("Removing %s", dest))
		}
	}

	if opts.DryRun {
		return nil
	}

	if err := os.Remove(dest); err != nil {
		return fmt.Errorf("failed to remove file: %w", err)
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, 0, fmt.Sprintf("✓ Removed %s", dest))
	}

	return nil
}

// expandPath expands ~ to home directory.
// Only paths starting with ~/ are accepted for security.
func expandPath(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return "", fmt.Errorf("destination path must start with ~/: %q", path)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	expanded := filepath.Clean(filepath.Join(home, path[2:]))

	if err := validation.ValidateDestinationPath(expanded, home); err != nil {
		return "", fmt.Errorf("invalid destination path: %w", err)
	}

	return expanded, nil
}

// ValidateTemplate checks if a template is valid
func ValidateTemplate(templateStr string) error {
	_, err := template.New("validate").Parse(templateStr)
	if err != nil {
		return fmt.Errorf("invalid template: %w", err)
	}
	return nil
}

// PreviewRender renders a template without writing, for preview purposes
func PreviewRender(mc *config.MachinePrompt, values map[string]string) (string, error) {
	result, err := RenderMachineConfig(mc, values)
	if err != nil {
		return "", err
	}
	return result.Content, nil
}
