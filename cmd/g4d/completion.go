package main

import (
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for g4d.

To load completions:

Bash:
  # Linux:
  $ g4d completion bash > /etc/bash_completion.d/g4d

  # macOS:
  $ g4d completion bash > $(brew --prefix)/etc/bash_completion.d/g4d

  # Current session only:
  $ source <(g4d completion bash)

Zsh:
  # If shell completions are not already enabled, you need to enable them:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Load completions for every new session:
  $ g4d completion zsh > "${fpath[1]}/_g4d"

  # Current session only:
  $ source <(g4d completion zsh)

Fish:
  $ g4d completion fish > ~/.config/fish/completions/g4d.fish

  # Current session only:
  $ g4d completion fish | source`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()
		switch args[0] {
		case "bash":
			return cmd.Root().GenBashCompletionV2(out, true)
		case "zsh":
			return cmd.Root().GenZshCompletion(out)
		case "fish":
			return cmd.Root().GenFishCompletion(out, true)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
