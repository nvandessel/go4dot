package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the string representation of the validation error
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors is a collection of validation errors
type ValidationErrors []ValidationError

// Error returns the string representation of all validation errors
func (e ValidationErrors) Error() string {
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// Validate checks if the configuration is valid
func (c *Config) Validate(configDir string) error {
	var errors ValidationErrors

	// Check schema version
	if c.SchemaVersion == "" {
		errors = append(errors, ValidationError{
			Field:   "schema_version",
			Message: "schema_version is required",
		})
	}

	// Validate metadata
	if c.Metadata.Name == "" {
		errors = append(errors, ValidationError{
			Field:   "metadata.name",
			Message: "name is required",
		})
	}

	// Validate configs
	configNames := make(map[string]bool)

	// Check core configs
	for i, cfg := range c.Configs.Core {
		if cfg.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.core[%d].name", i),
				Message: "name is required",
			})
		}
		if cfg.Path == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.core[%d].path", i),
				Message: "path is required",
			})
		}

		// Check for duplicate names
		if configNames[cfg.Name] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.core[%d].name", i),
				Message: fmt.Sprintf("duplicate config name: %s", cfg.Name),
			})
		}
		configNames[cfg.Name] = true

		// Check if path exists
		if !filepath.IsAbs(cfg.Path) {
			cfg.Path = filepath.Join(configDir, cfg.Path)
		}
		if _, err := os.Stat(cfg.Path); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.core[%d].path", i),
				Message: fmt.Sprintf("path does not exist: %s", cfg.Path),
			})
		}

		// Validate per-config external dependencies
		for j, ext := range cfg.ExternalDeps {
			extErrors := validateExternalDep(ext, fmt.Sprintf("configs.core[%d].external_deps[%d]", i, j))
			errors = append(errors, extErrors...)
		}
	}

	// Check optional configs
	for i, cfg := range c.Configs.Optional {
		if cfg.Name == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.optional[%d].name", i),
				Message: "name is required",
			})
		}
		if cfg.Path == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.optional[%d].path", i),
				Message: "path is required",
			})
		}

		// Check for duplicate names
		if configNames[cfg.Name] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.optional[%d].name", i),
				Message: fmt.Sprintf("duplicate config name: %s", cfg.Name),
			})
		}
		configNames[cfg.Name] = true

		// Check if path exists
		if !filepath.IsAbs(cfg.Path) {
			cfg.Path = filepath.Join(configDir, cfg.Path)
		}
		if _, err := os.Stat(cfg.Path); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.optional[%d].path", i),
				Message: fmt.Sprintf("path does not exist: %s", cfg.Path),
			})
		}

		// Validate per-config external dependencies
		for j, ext := range cfg.ExternalDeps {
			extErrors := validateExternalDep(ext, fmt.Sprintf("configs.optional[%d].external_deps[%d]", i, j))
			errors = append(errors, extErrors...)
		}
	}

	// Validate external dependencies
	for i, ext := range c.External {
		extErrors := validateExternalDep(ext, fmt.Sprintf("external[%d]", i))
		errors = append(errors, extErrors...)
	}

	// Validate machine config
	for i, mc := range c.MachineConfig {
		if mc.ID == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("machine_config[%d].id", i),
				Message: "id is required",
			})
		}
		if mc.Destination == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("machine_config[%d].destination", i),
				Message: "destination is required",
			})
		}
		if mc.Template == "" {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("machine_config[%d].template", i),
				Message: "template is required",
			})
		}
	}

	// Validate post_install script
	if c.PostInstall != "" {
		scriptPath := c.PostInstall
		if !filepath.IsAbs(scriptPath) {
			scriptPath = filepath.Join(configDir, scriptPath)
		}
		info, err := os.Stat(scriptPath)
		if os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   "post_install",
				Message: fmt.Sprintf("script does not exist: %s", scriptPath),
			})
		} else if err == nil {
			// Check if it's a file and executable
			if info.IsDir() {
				errors = append(errors, ValidationError{
					Field:   "post_install",
					Message: fmt.Sprintf("path is a directory, not a file: %s", scriptPath),
				})
			} else if info.Mode()&0111 == 0 {
				errors = append(errors, ValidationError{
					Field:   "post_install",
					Message: fmt.Sprintf("script is not executable: %s", scriptPath),
				})
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}

	// Check for circular dependencies
	if err := c.validateCircularDependencies(); err != nil {
		errors = append(errors, ValidationError{
			Field:   "configs",
			Message: err.Error(),
		})
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// GetAllDependencies returns all dependencies (critical + core + optional)
func (c *Config) GetAllDependencies() []DependencyItem {
	var all []DependencyItem
	all = append(all, c.Dependencies.Critical...)
	all = append(all, c.Dependencies.Core...)
	all = append(all, c.Dependencies.Optional...)
	return all
}

// GetAllConfigs returns all configs (core + optional)
func (c *Config) GetAllConfigs() []ConfigItem {
	var all []ConfigItem
	all = append(all, c.Configs.Core...)
	all = append(all, c.Configs.Optional...)
	return all
}

// GetConfigByName finds a config by name
// GetConfigByName returns a config item by its name
func (c *Config) GetConfigByName(name string) *ConfigItem {
	for _, cfg := range c.Configs.Core {
		if cfg.Name == name {
			return &cfg
		}
	}
	for _, cfg := range c.Configs.Optional {
		if cfg.Name == name {
			return &cfg
		}
	}
	return nil
}

// validateExternalDep validates a single external dependency
func validateExternalDep(ext ExternalDep, prefix string) []ValidationError {
	var errors []ValidationError
	if ext.ID == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".id",
			Message: "id is required",
		})
	}
	if ext.URL == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".url",
			Message: "url is required",
		})
	}
	if ext.Destination == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".destination",
			Message: "destination is required",
		})
	}
	method := strings.ToLower(strings.TrimSpace(ext.Method))
	if method != "" && method != "clone" && method != "copy" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".method",
			Message: "method must be \"clone\" or \"copy\"",
		})
	}

	merge := strings.ToLower(strings.TrimSpace(ext.MergeStrategy))
	if merge != "" && merge != "overwrite" && merge != "keep_existing" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".merge_strategy",
			Message: "merge_strategy must be \"overwrite\" or \"keep_existing\"",
		})
	}
	return errors
}

func (c *Config) validateCircularDependencies() error {
	allConfigs := c.GetAllConfigs()
	graph := make(map[string][]string)
	for _, cfg := range allConfigs {
		graph[cfg.Name] = cfg.DependsOn
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)

	var dfs func(name string) error
	dfs = func(name string) error {
		visiting[name] = true
		for _, dep := range graph[name] {
			if visiting[dep] {
				return fmt.Errorf("circular dependency detected: %s -> %s", name, dep)
			}
			if !visited[dep] {
				if err := dfs(dep); err != nil {
					return err
				}
			}
		}
		visiting[name] = false
		visited[name] = true
		return nil
	}

	for name := range graph {
		if !visited[name] {
			if err := dfs(name); err != nil {
				return err
			}
		}
	}

	return nil
}
