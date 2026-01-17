package config

// Config represents the complete .go4dot.yaml configuration
type Config struct {
	SchemaVersion string          `yaml:"schema_version"`
	Metadata      Metadata        `yaml:"metadata"`
	Dependencies  Dependencies    `yaml:"dependencies"`
	Configs       ConfigGroups    `yaml:"configs"`
	External      []ExternalDep   `yaml:"external"`
	MachineConfig []MachinePrompt `yaml:"machine_config"`
	Archived      []ConfigItem    `yaml:"archived"`
	PostInstall   string          `yaml:"post_install"`
}

// Metadata contains project information
type Metadata struct {
	Name        string `yaml:"name"`
	Author      string `yaml:"author"`
	Repository  string `yaml:"repository"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}

// Dependencies lists required system packages
type Dependencies struct {
	Critical []DependencyItem `yaml:"critical"`
	Core     []DependencyItem `yaml:"core"`
	Optional []DependencyItem `yaml:"optional"`
}

// DependencyItem represents a single dependency
// Can be a simple string or a complex object with package mappings
type DependencyItem struct {
	Name       string            `yaml:"name"`
	Binary     string            `yaml:"binary"`      // Binary name to check in PATH
	Package    map[string]string `yaml:"package"`     // Package name per manager
	Version    string            `yaml:"version"`     // Required version (e.g. "0.11+")
	VersionCmd string            `yaml:"version_cmd"` // Command to check version (defaults to --version)
}

// UnmarshalYAML allows DependencyItem to accept both string and object formats
func (d *DependencyItem) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Try to unmarshal as a string first
	var str string
	if err := unmarshal(&str); err == nil {
		d.Name = str
		d.Binary = str
		return nil
	}

	// Otherwise unmarshal as full struct
	type plain DependencyItem
	return unmarshal((*plain)(d))
}

// ConfigGroups organizes configs by category
type ConfigGroups struct {
	Core     []ConfigItem `yaml:"core"`
	Optional []ConfigItem `yaml:"optional"`
}

// ConfigItem represents a single dotfile configuration
type ConfigItem struct {
	Name                  string        `yaml:"name"`
	Path                  string        `yaml:"path"`
	Description           string        `yaml:"description"`
	Platforms             []string      `yaml:"platforms"`
	DependsOn             []string      `yaml:"depends_on"`
	ExternalDeps          []ExternalDep `yaml:"external_deps"`
	RequiresMachineConfig bool          `yaml:"requires_machine_config"`
}

// ExternalDep represents an external dependency to clone (plugins, themes, etc.)
type ExternalDep struct {
	Name          string            `yaml:"name"`
	ID            string            `yaml:"id"`
	URL           string            `yaml:"url"`
	Destination   string            `yaml:"destination"`
	Method        string            `yaml:"method"`         // "clone" or "copy"
	MergeStrategy string            `yaml:"merge_strategy"` // "overwrite" (default) or "keep_existing"
	Condition     map[string]string `yaml:"condition"`
}

// MachinePrompt represents machine-specific configuration prompts
type MachinePrompt struct {
	ID          string        `yaml:"id"`
	Description string        `yaml:"description"`
	Destination string        `yaml:"destination"`
	Prompts     []PromptField `yaml:"prompts"`
	Template    string        `yaml:"template"`
}

// PromptField represents a single prompt for user input
type PromptField struct {
	ID       string   `yaml:"id"`
	Prompt   string   `yaml:"prompt"`
	Type     string   `yaml:"type"` // text, password, confirm, select
	Required bool     `yaml:"required"`
	Default  string   `yaml:"default"`
	Options  []string `yaml:"options,omitempty"` // Options for select type
}
