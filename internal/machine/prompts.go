package machine

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/nvandessel/go4dot/internal/config"
)

// PromptResult holds the collected values from prompts
type PromptResult struct {
	ID     string
	Values map[string]string
}

// PromptOptions configures prompt behavior
type PromptOptions struct {
	In           io.Reader                            // Input source (defaults to os.Stdin)
	Out          io.Writer                            // Output destination (defaults to os.Stdout)
	ProgressFunc func(current, total int, msg string) // Called for progress updates with item counts
	SkipPrompts  bool                                 // Use defaults without prompting
}

// CollectMachineConfig prompts the user for all machine-specific values
func CollectMachineConfig(cfg *config.Config, opts PromptOptions) ([]PromptResult, error) {
	// Set defaults if nil
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	var results []PromptResult

	for _, mc := range cfg.MachineConfig {
		result, err := collectPrompts(mc, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to collect prompts for %s: %w", mc.ID, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// CollectSingleConfig prompts for a single machine config by ID
func CollectSingleConfig(cfg *config.Config, id string, opts PromptOptions) (*PromptResult, error) {
	if opts.In == nil {
		opts.In = os.Stdin
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}

	var found *config.MachinePrompt
	for i := range cfg.MachineConfig {
		if cfg.MachineConfig[i].ID == id {
			found = &cfg.MachineConfig[i]
			break
		}
	}

	if found == nil {
		return nil, fmt.Errorf("machine config '%s' not found", id)
	}

	result, err := collectPrompts(*found, opts)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// collectPrompts collects values for a single MachinePrompt using Huh forms
func collectPrompts(mc config.MachinePrompt, opts PromptOptions) (PromptResult, error) {
	result := PromptResult{
		ID:     mc.ID,
		Values: make(map[string]string),
	}

	if opts.ProgressFunc != nil {
		opts.ProgressFunc(0, 0, fmt.Sprintf("Configuring %s...", mc.Description))
	}

	// Prepare fields for the form
	var groups []*huh.Group
	var fields []huh.Field
	valuePointers := make(map[string]interface{})

	for _, prompt := range mc.Prompts {
		// If skipping prompts, just use default/validate
		if opts.SkipPrompts {
			if prompt.Required && prompt.Default == "" {
				return result, fmt.Errorf("required field '%s' has no default value", prompt.ID)
			}
			result.Values[prompt.ID] = prompt.Default
			continue
		}

		switch prompt.Type {
		case "confirm":
			var val bool
			if prompt.Default == "true" || prompt.Default == "yes" || prompt.Default == "y" {
				val = true
			}
			valuePointers[prompt.ID] = &val

			fields = append(fields, huh.NewConfirm().
				Title(prompt.Prompt).
				Value(&val))

		case "select":
			val := prompt.Default
			valuePointers[prompt.ID] = &val

			var options []huh.Option[string]
			for _, opt := range prompt.Options {
				options = append(options, huh.NewOption(opt, opt))
			}

			if len(options) > 0 {
				fields = append(fields, huh.NewSelect[string]().
					Title(prompt.Prompt).
					Options(options...).
					Value(&val))
			} else {
				// Fallback to text input if no options provided
				f := huh.NewInput().
					Title(prompt.Prompt).
					Value(&val)
				if prompt.Required {
					f.Validate(requiredValidator)
				}
				fields = append(fields, f)
			}

		default: // text
			val := prompt.Default
			valuePointers[prompt.ID] = &val

			f := huh.NewInput().
				Title(prompt.Prompt).
				Value(&val)
			if prompt.Required {
				f.Validate(requiredValidator)
			}
			fields = append(fields, f)
		}
	}

	// If we skipped everything (or no prompts), return
	if opts.SkipPrompts || len(fields) == 0 {
		return result, nil
	}

	// Run the form
	// We put all fields in one group for now
	groups = append(groups, huh.NewGroup(fields...))

	form := huh.NewForm(groups...).
		WithInput(opts.In).
		WithOutput(opts.Out)

	err := form.Run()
	if err != nil {
		return result, err
	}

	// Extract values
	for id, ptr := range valuePointers {
		switch v := ptr.(type) {
		case *string:
			result.Values[id] = *v
		case *bool:
			result.Values[id] = strconv.FormatBool(*v)
		}
	}

	return result, nil
}

func requiredValidator(s string) error {
	if s == "" {
		return fmt.Errorf("this field is required")
	}
	return nil
}

// GetMachineConfigByID returns a machine config by its ID
func GetMachineConfigByID(cfg *config.Config, id string) *config.MachinePrompt {
	for i := range cfg.MachineConfig {
		if cfg.MachineConfig[i].ID == id {
			return &cfg.MachineConfig[i]
		}
	}
	return nil
}

// ListMachineConfigs returns all machine config IDs and descriptions
func ListMachineConfigs(cfg *config.Config) []struct {
	ID          string
	Description string
} {
	var list []struct {
		ID          string
		Description string
	}
	for _, mc := range cfg.MachineConfig {
		list = append(list, struct {
			ID          string
			Description string
		}{
			ID:          mc.ID,
			Description: mc.Description,
		})
	}
	return list
}
