package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration files",
	Long:  "Commands for working with .go4dot.yaml configuration files",
}

var configValidateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate a .go4dot.yaml file",
	Long:  "Validate the syntax and structure of a .go4dot.yaml configuration file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var configPath string
		var err error

		if len(args) > 0 {
			// Load from specified path
			configPath = args[0]
			cfg, err = config.LoadFromPath(configPath)
		} else {
			// Discover config
			cfg, configPath, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Loaded config from: %s\n", configPath)

		// Validate
		if err := cfg.Validate(filepath.Dir(configPath)); err != nil {
			fmt.Fprintf(os.Stderr, "Validation failed:\n%v\n", err)
			os.Exit(1)
		}

		fmt.Println("Configuration is valid")
		fmt.Printf("  Schema version: %s\n", cfg.SchemaVersion)
		fmt.Printf("  Name: %s\n", cfg.Metadata.Name)
		fmt.Printf("  Configs: %d core, %d optional\n", len(cfg.Configs.Core), len(cfg.Configs.Optional))
		fmt.Printf("  Dependencies: %d total\n", len(cfg.GetAllDependencies()))
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show [path]",
	Short: "Display configuration contents",
	Long:  "Display the full contents of a .go4dot.yaml configuration file",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var configPath string
		var err error

		if len(args) > 0 {
			configPath = args[0]
			cfg, err = config.LoadFromPath(configPath)
		} else {
			cfg, configPath, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Configuration from: %s\n", configPath)
		fmt.Println("---------------------------------")

		// Convert to YAML and print
		data, err := yaml.Marshal(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(string(data))
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)
}
