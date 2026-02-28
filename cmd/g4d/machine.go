package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/validation"
	"github.com/spf13/cobra"
)

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "Manage machine-specific configuration",
	Long:  "Commands for configuring machine-specific settings like git user, GPG keys, etc.",
}

var machineStatusCmd = &cobra.Command{
	Use:   "status [config-path]",
	Short: "Show status of machine configurations",
	Long:  "Display which machine-specific configurations are set up and which are missing",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var err error

		if len(args) > 0 {
			cfg, err = config.LoadFromPath(args[0])
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.MachineConfig) == 0 {
			fmt.Println("No machine configurations defined in config")
			return
		}

		statuses := machine.CheckMachineConfigStatus(cfg)
		machine.PrintStatus(statuses)
	},
}

var machineConfigureCmd = &cobra.Command{
	Use:   "configure [id] [config-path]",
	Short: "Configure machine-specific settings",
	Long: `Interactively configure machine-specific settings.

Without arguments, configures all machine settings.
With an ID argument, configures only that specific setting.`,
	Args: cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var cfg *config.Config
		var err error
		var specificID string

		configPathArg := ""
		if len(args) >= 1 {
			if _, statErr := os.Stat(args[0]); statErr == nil || filepath.Ext(args[0]) == ".yaml" || filepath.Ext(args[0]) == ".yml" {
				configPathArg = args[0]
			} else {
				specificID = args[0]
				if len(args) >= 2 {
					configPathArg = args[1]
				}
			}
		}

		if configPathArg != "" {
			cfg, err = config.LoadFromPath(configPathArg)
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		if len(cfg.MachineConfig) == 0 {
			fmt.Println("No machine configurations defined in config")
			return
		}

		skipPrompts, _ := cmd.Flags().GetBool("defaults")
		overwrite, _ := cmd.Flags().GetBool("overwrite")

		promptOpts := machine.PromptOptions{
			SkipPrompts: skipPrompts,
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		renderOpts := machine.RenderOptions{
			Overwrite: overwrite,
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		if specificID != "" {
			// Configure single
			fmt.Printf("Configuring %s...\n\n", specificID)

			result, err := machine.CollectSingleConfig(cfg, specificID, promptOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			mc := machine.GetMachineConfigByID(cfg, specificID)
			_, err = machine.RenderAndWrite(mc, result.Values, renderOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Configure all
			fmt.Printf("Configuring %d machine settings...\n\n", len(cfg.MachineConfig))

			results, err := machine.CollectMachineConfig(cfg, promptOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}

			_, err = machine.RenderAll(cfg, results, renderOpts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		}

		fmt.Println("\nConfiguration complete")
	},
}

var machineShowCmd = &cobra.Command{
	Use:   "show <id> [config-path]",
	Short: "Preview a machine configuration",
	Long:  "Show what a machine configuration would generate without writing it",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		var cfg *config.Config
		var err error

		if len(args) > 1 {
			cfg, err = config.LoadFromPath(args[1])
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		mc := machine.GetMachineConfigByID(cfg, id)
		if mc == nil {
			fmt.Fprintf(os.Stderr, "Error: machine config '%s' not found\n", id)
			os.Exit(1)
		}

		// Collect values (use defaults)
		result, err := machine.CollectSingleConfig(cfg, id, machine.PromptOptions{SkipPrompts: true})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error collecting defaults: %v\n", err)
			os.Exit(1)
		}

		content, err := machine.PreviewRender(mc, result.Values)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rendering preview: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Preview of %s (destination: %s):\n", mc.Description, mc.Destination)
		fmt.Println("------------------------------------")
		fmt.Println(content)
	},
}

var machineRemoveCmd = &cobra.Command{
	Use:   "remove <id> [config-path]",
	Short: "Remove a machine configuration file",
	Long:  "Remove a generated machine configuration file",
	Args:  cobra.RangeArgs(1, 2),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		var cfg *config.Config
		var err error

		if len(args) > 1 {
			cfg, err = config.LoadFromPath(args[1])
		} else {
			cfg, _, err = config.LoadFromDiscovery()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		mc := machine.GetMachineConfigByID(cfg, id)
		if mc == nil {
			fmt.Fprintf(os.Stderr, "Error: machine config '%s' not found\n", id)
			os.Exit(1)
		}

		opts := machine.RenderOptions{
			ProgressFunc: func(current, total int, msg string) {
				if total > 0 && current > 0 {
					fmt.Printf("[%d/%d] %s\n", current, total, msg)
				} else {
					fmt.Println(msg)
				}
			},
		}

		err = machine.RemoveMachineConfig(mc, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var machineInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show system information for machine config",
	Long:  "Display detected system information useful for machine configuration",
	Run: func(cmd *cobra.Command, args []string) {
		info, err := machine.GetSystemInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting system info: %v\n", err)
			os.Exit(1)
		}

		machine.PrintSystemInfo(info)
	},
}

var machineKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage SSH and GPG keys",
	Long:  "Commands for listing, generating, and registering SSH and GPG keys.",
}

var machineKeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all detected SSH and GPG keys",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("-- SSH Keys --")

		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		sshDir := filepath.Join(home, ".ssh")

		keys, err := machine.DetectAllSSHKeys(sshDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		if len(keys) == 0 {
			fmt.Println("  No SSH keys found")
		} else {
			for _, key := range keys {
				status := ""
				if key.Loaded {
					status = " [loaded in agent]"
				}
				fmt.Printf("  %s (%s)%s\n", key.Path, strings.ToUpper(key.Type), status)
				if key.Fingerprint != "" {
					fmt.Printf("    Fingerprint: %s\n", key.Fingerprint)
				}
				if key.Comment != "" {
					fmt.Printf("    Comment: %s\n", key.Comment)
				}
			}
		}

		fmt.Println("\n-- GPG Keys --")
		gpgKeys, err := machine.DetectGPGKeys()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		if len(gpgKeys) == 0 {
			fmt.Println("  No GPG keys found")
		} else {
			for _, key := range gpgKeys {
				fmt.Printf("  %s <%s> (%s)\n", key.UserID, key.Email, key.KeyID)
			}
		}
	},
}

var machineKeysGenerateSSHCmd = &cobra.Command{
	Use:   "generate-ssh",
	Short: "Generate a new ed25519 SSH key",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		sshDir := filepath.Join(home, ".ssh")

		// Show existing keys
		keys, err := machine.DetectAllSSHKeys(sshDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not detect existing keys: %v\n", err)
		}
		if len(keys) > 0 {
			fmt.Println("Existing SSH keys:")
			for _, key := range keys {
				fmt.Printf("  %s (%s)\n", key.Path, strings.ToUpper(key.Type))
			}
			fmt.Println()
		}

		// Get email
		email, _ := cmd.Flags().GetString("email")
		if email == "" {
			// Try git config
			gitEmail, _ := machine.GetGitUserEmail()
			if gitEmail != "" {
				email = gitEmail
			}
		}

		if email == "" {
			fmt.Fprintf(os.Stderr, "Error: email is required (use --email flag or configure git user.email)\n")
			os.Exit(1)
		}

		if err := validation.ValidateEmail(email); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		name, _ := cmd.Flags().GetString("name")
		if strings.ContainsAny(name, "/\\") {
			fmt.Fprintf(os.Stderr, "Error: key name must not contain path separators\n")
			os.Exit(1)
		}

		keyPath, err := machine.GenerateSSHKey(machine.SSHKeygenOpts{
			Email:  email,
			Name:   name,
			SSHDir: sshDir,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nGenerated SSH key: %s\n", keyPath)

		// Try to add to agent
		if machine.IsAgentRunning() {
			fmt.Println("Adding key to SSH agent...")
			if err := machine.AddKeyToAgent(keyPath, sshDir); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not add key to agent: %v\n", err)
			} else {
				fmt.Println("Key added to SSH agent")
			}
		}

		// Print public key
		pubKey, err := machine.GetSSHPublicKey(keyPath+".pub", sshDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read public key: %v\n", err)
		} else {
			fmt.Printf("\nPublic key:\n%s\n", pubKey)
			fmt.Println("\nCopy the key above to add it to GitHub")
		}
	},
}

func init() {
	rootCmd.AddCommand(machineCmd)
	machineCmd.AddCommand(machineStatusCmd)
	machineCmd.AddCommand(machineConfigureCmd)
	machineCmd.AddCommand(machineShowCmd)
	machineCmd.AddCommand(machineRemoveCmd)
	machineCmd.AddCommand(machineInfoCmd)

	machineCmd.AddCommand(machineKeysCmd)
	machineKeysCmd.AddCommand(machineKeysListCmd)
	machineKeysCmd.AddCommand(machineKeysGenerateSSHCmd)

	// Flags for machine configure
	machineConfigureCmd.Flags().Bool("defaults", false, "Use default values without prompting")
	machineConfigureCmd.Flags().Bool("overwrite", false, "Overwrite existing configuration files")

	// Flags for generate-ssh
	machineKeysGenerateSSHCmd.Flags().String("email", "", "Email for key comment")
	machineKeysGenerateSSHCmd.Flags().String("name", "id_ed25519", "Key filename")
}
