package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	ConfigFileName = ".go4dot.yaml"
)

var ErrConfigNotFound = errors.New("config not found")

func IsNotFound(err error) bool {
	return errors.Is(err, ErrConfigNotFound)
}

// Load reads and parses a .go4dot.yaml file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &cfg, nil
}

// FindConfig searches for .go4dot.yaml in common locations
func FindConfig() (string, error) {
	// Search locations in order of priority
	searchPaths := []string{
		// Current directory
		".",
		// Home dotfiles directory
		filepath.Join(os.Getenv("HOME"), "dotfiles"),
		// Hidden dotfiles directory
		filepath.Join(os.Getenv("HOME"), ".dotfiles"),
	}

	for _, basePath := range searchPaths {
		configPath := filepath.Join(basePath, ConfigFileName)
		if _, err := os.Stat(configPath); err == nil {
			// Found it!
			absPath, err := filepath.Abs(configPath)
			if err != nil {
				return configPath, nil
			}
			return absPath, nil
		}
	}

	return "", fmt.Errorf("%w: could not find %s in any standard location", ErrConfigNotFound, ConfigFileName)
}

// LoadFromDiscovery finds and loads the config file
func LoadFromDiscovery() (*Config, string, error) {
	configPath, err := FindConfig()
	if err != nil {
		return nil, "", err
	}

	cfg, err := Load(configPath)
	if err != nil {
		return nil, configPath, err
	}

	return cfg, configPath, nil
}

// LoadFromPath loads config from a specific path
func LoadFromPath(path string) (*Config, error) {
	// If path is a directory, append the config filename
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}

	if stat.IsDir() {
		path = filepath.Join(path, ConfigFileName)
	}

	return Load(path)
}

// ResolveRepoRoot determines the repository root from a path
func ResolveRepoRoot(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if stat.IsDir() {
		return absPath, nil
	}
	return filepath.Dir(absPath), nil
}
