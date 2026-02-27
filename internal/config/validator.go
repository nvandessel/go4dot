package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
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
		} else if err := validation.ValidateConfigName(cfg.Name); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.core[%d].name", i),
				Message: err.Error(),
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

		// Validate path
		pathErrors := validateConfigPath(cfg.Path, configDir, fmt.Sprintf("configs.core[%d].path", i))
		errors = append(errors, pathErrors...)

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
		} else if err := validation.ValidateConfigName(cfg.Name); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("configs.optional[%d].name", i),
				Message: err.Error(),
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

		// Validate path
		pathErrors := validateConfigPath(cfg.Path, configDir, fmt.Sprintf("configs.optional[%d].path", i))
		errors = append(errors, pathErrors...)

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

	// Validate dependency items for security
	for i, dep := range c.Dependencies.Critical {
		depErrors := validateDependencyItem(dep, fmt.Sprintf("dependencies.critical[%d]", i))
		errors = append(errors, depErrors...)
	}
	for i, dep := range c.Dependencies.Core {
		depErrors := validateDependencyItem(dep, fmt.Sprintf("dependencies.core[%d]", i))
		errors = append(errors, depErrors...)
	}
	for i, dep := range c.Dependencies.Optional {
		depErrors := validateDependencyItem(dep, fmt.Sprintf("dependencies.optional[%d]", i))
		errors = append(errors, depErrors...)
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

		// Security: validate machine config destination prefix
		mcErrors := validateMachineConfig(mc, fmt.Sprintf("machine_config[%d]", i))
		errors = append(errors, mcErrors...)
	}

	// PostInstall is a display-only string shown to the user after installation.
	// It is not executed by go4dot, so no executable-bit validation is needed.

	// After all other validation, check for circular dependencies
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

// validateConfigPath validates a single config path
func validateConfigPath(path, configDir, fieldPrefix string) []ValidationError {
	var errors []ValidationError
	if path == "" {
		// This is already checked in the main validation loop,
		// but we keep it here for robustness.
		return errors
	}

	absPath := path
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(configDir, absPath)
	}

	if _, err := os.Stat(absPath); err != nil {
		if os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Field:   fieldPrefix,
				Message: fmt.Sprintf("path does not exist: %s", absPath),
			})
		} else {
			errors = append(errors, ValidationError{
				Field:   fieldPrefix,
				Message: fmt.Sprintf("invalid path: %s (%v)", absPath, err),
			})
		}
	}
	return errors
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
	} else if err := validation.ValidateGitURL(ext.URL); err != nil {
		errors = append(errors, ValidationError{
			Field:   prefix + ".url",
			Message: err.Error(),
		})
	}
	if ext.Destination == "" {
		errors = append(errors, ValidationError{
			Field:   prefix + ".destination",
			Message: "destination is required",
		})
	} else if !strings.HasPrefix(ext.Destination, "~/") && !strings.HasPrefix(ext.Destination, "@repoRoot/") {
		errors = append(errors, ValidationError{
			Field:   prefix + ".destination",
			Message: "destination must start with ~/ or @repoRoot/",
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

// validateDependencyItem validates a single dependency item's fields for security.
func validateDependencyItem(dep DependencyItem, prefix string) []ValidationError {
	var errors []ValidationError

	// Validate Binary field if set
	if dep.Binary != "" {
		if err := validation.ValidateBinaryName(dep.Binary); err != nil {
			errors = append(errors, ValidationError{
				Field:   prefix + ".binary",
				Message: err.Error(),
			})
		}
	}

	// Validate VersionCmd field if set
	if dep.VersionCmd != "" {
		if err := validation.ValidateVersionCmd(dep.VersionCmd); err != nil {
			errors = append(errors, ValidationError{
				Field:   prefix + ".version_cmd",
				Message: err.Error(),
			})
		}
	}

	// Validate Package map values
	for mgr, pkgName := range dep.Package {
		if err := validation.ValidatePackageName(pkgName); err != nil {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("%s.package[%s]", prefix, mgr),
				Message: err.Error(),
			})
		}
	}

	return errors
}

// validateMachineConfig validates machine config fields for security.
func validateMachineConfig(mc MachinePrompt, prefix string) []ValidationError {
	var errors []ValidationError

	// Validate destination prefix (must start with ~/)
	if mc.Destination != "" && !strings.HasPrefix(mc.Destination, "~/") {
		errors = append(errors, ValidationError{
			Field:   prefix + ".destination",
			Message: "machine config destination must start with ~/",
		})
	}

	return errors
}

func (c *Config) validateCircularDependencies() error {
	allConfigs := c.GetAllConfigs()
	graph := make(map[string][]string)
	configSet := make(map[string]struct{})
	for _, cfg := range allConfigs {
		graph[cfg.Name] = cfg.DependsOn
		configSet[cfg.Name] = struct{}{}
	}

	for name, deps := range graph {
		for _, dep := range deps {
			if _, ok := configSet[dep]; !ok {
				return fmt.Errorf("unknown dependency '%s' referenced by '%s'", dep, name)
			}
		}
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
