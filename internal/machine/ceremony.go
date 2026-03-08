package machine

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// CeremonyConfig holds the configuration for a new-signing ceremony.
type CeremonyConfig struct {
	GPGRunner    GPGRunner
	GitHubClient *GitHubClient
	KeyID        string
	Fingerprint  string
}

// StepResult represents the outcome of a ceremony step.
type StepResult struct {
	Name    string
	Success bool
	Message string
}

// CeremonyResult holds the results of a complete ceremony run.
type CeremonyResult struct {
	Steps []StepResult
}

// PreflightChecks verifies the prerequisites for a signing ceremony.
func PreflightChecks(ghClient *GitHubClient) []StepResult {
	var results []StepResult

	// Check gpg is installed
	if _, err := exec.LookPath("gpg"); err == nil {
		results = append(results, StepResult{
			Name: "gpg-installed", Success: true, Message: "gpg is installed",
		})
	} else {
		results = append(results, StepResult{
			Name: "gpg-installed", Success: false, Message: "gpg is not installed",
		})
	}

	// Check gh is authenticated
	if ghClient != nil && HasGHCLI() {
		auth, _ := ghClient.IsAuthenticated()
		if auth {
			results = append(results, StepResult{
				Name: "gh-authenticated", Success: true, Message: "gh CLI is authenticated",
			})
		} else {
			results = append(results, StepResult{
				Name: "gh-authenticated", Success: false, Message: "gh CLI is not authenticated — run `gh auth login`",
			})
		}
	} else {
		results = append(results, StepResult{
			Name: "gh-authenticated", Success: false, Message: "gh CLI is not installed",
		})
	}

	return results
}

// ImportMasterKey imports a GPG key from a file (e.g., USB drive).
func ImportMasterKey(runner GPGRunner, keyPath string) error {
	if keyPath == "" {
		return fmt.Errorf("key path must not be empty")
	}

	// Validate the path is a regular file
	info, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("cannot access key file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("key path is not a regular file: %s", keyPath)
	}

	output, err := runner.Run("--import", keyPath)
	if err != nil {
		return fmt.Errorf("gpg --import failed: %w\nOutput: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// RunEditKeyInteractive launches gpg --edit-key with full terminal passthrough.
// This allows the user to interactively add subkeys, set expiry, etc.
// Not unit tested — thin terminal passthrough.
func RunEditKeyInteractive(keyID string) error {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}

	cmd := exec.Command("gpg", "--edit-key", keyID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpg --edit-key failed: %w", err)
	}
	return nil
}

// ConfigureGitSigning writes GPG signing configuration to ~/.gitconfig.local.
// Uses git config --file to avoid modifying the main ~/.gitconfig.
func ConfigureGitSigning(keyID, email string) error {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return fmt.Errorf("invalid key ID: %w", err)
	}
	if err := validation.ValidateEmail(email); err != nil {
		return fmt.Errorf("invalid email: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	configFile := filepath.Join(home, ".gitconfig.local")

	configs := []struct{ key, value string }{
		{"user.signingkey", keyID},
		{"user.email", email},
		{"commit.gpgsign", "true"},
		{"tag.gpgsign", "true"},
	}

	for _, cfg := range configs {
		cmd := exec.Command("git", "config", "--file", configFile, cfg.key, cfg.value)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git config %s failed: %w\nOutput: %s", cfg.key, err, strings.TrimSpace(string(output)))
		}
	}

	return nil
}

// RunNewSigningCeremony orchestrates all steps of a new-signing key ceremony.
func RunNewSigningCeremony(cfg CeremonyConfig, masterKeyPath string, email string, skipEditKey bool) (*CeremonyResult, error) {
	result := &CeremonyResult{}

	// Step 1: Pre-flight checks
	preflights := PreflightChecks(cfg.GitHubClient)
	result.Steps = append(result.Steps, preflights...)
	for _, pf := range preflights {
		if !pf.Success {
			return result, fmt.Errorf("pre-flight check failed: %s — %s", pf.Name, pf.Message)
		}
	}

	// Step 2: Import master key
	if err := ImportMasterKey(cfg.GPGRunner, masterKeyPath); err != nil {
		result.Steps = append(result.Steps, StepResult{
			Name: "import-master", Success: false, Message: err.Error(),
		})
		return result, fmt.Errorf("import master key: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name: "import-master", Success: true, Message: "Master key imported",
	})

	// Step 3: Interactive gpg --edit-key (optional — skipped in non-interactive mode)
	if !skipEditKey {
		if err := RunEditKeyInteractive(cfg.KeyID); err != nil {
			result.Steps = append(result.Steps, StepResult{
				Name: "edit-key", Success: false, Message: err.Error(),
			})
			return result, fmt.Errorf("edit key: %w", err)
		}
		result.Steps = append(result.Steps, StepResult{
			Name: "edit-key", Success: true, Message: "Key editing complete",
		})
	}

	// Step 4: Strip master key
	stripResult, err := StripMasterKey(cfg.GPGRunner, cfg.KeyID)
	if err != nil {
		result.Steps = append(result.Steps, StepResult{
			Name: "strip-master", Success: false, Message: err.Error(),
		})
		return result, fmt.Errorf("strip master key: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name:    "strip-master",
		Success: true,
		Message: fmt.Sprintf("Master key stripped, %d subkeys preserved", stripResult.SubkeysPreserved),
	})

	// Step 5: Set ultimate trust
	if err := SetUltimateTrust(cfg.GPGRunner, cfg.Fingerprint); err != nil {
		result.Steps = append(result.Steps, StepResult{
			Name: "set-trust", Success: false, Message: err.Error(),
		})
		return result, fmt.Errorf("set trust: %w", err)
	}
	result.Steps = append(result.Steps, StepResult{
		Name: "set-trust", Success: true, Message: "Ultimate trust set",
	})

	// Step 6: Upload to GitHub
	if cfg.GitHubClient != nil {
		registered, err := cfg.GitHubClient.IsGPGKeyRegistered(cfg.KeyID)
		if err != nil {
			result.Steps = append(result.Steps, StepResult{
				Name: "github-upload", Success: false, Message: err.Error(),
			})
			return result, fmt.Errorf("check GitHub registration: %w", err)
		}

		if registered {
			// Re-upload with updated subkeys
			if err := cfg.GitHubClient.ReuploadGPGKey(cfg.KeyID); err != nil {
				result.Steps = append(result.Steps, StepResult{
					Name: "github-upload", Success: false, Message: err.Error(),
				})
				return result, fmt.Errorf("re-upload to GitHub: %w", err)
			}
			result.Steps = append(result.Steps, StepResult{
				Name: "github-upload", Success: true, Message: "Key re-uploaded to GitHub with new subkeys",
			})
		} else {
			if err := cfg.GitHubClient.AddGPGKey(cfg.KeyID); err != nil {
				result.Steps = append(result.Steps, StepResult{
					Name: "github-upload", Success: false, Message: err.Error(),
				})
				return result, fmt.Errorf("upload to GitHub: %w", err)
			}
			result.Steps = append(result.Steps, StepResult{
				Name: "github-upload", Success: true, Message: "Key uploaded to GitHub",
			})
		}
	}

	// Step 7: Configure git signing
	if email != "" {
		if err := ConfigureGitSigning(cfg.KeyID, email); err != nil {
			result.Steps = append(result.Steps, StepResult{
				Name: "git-signing", Success: false, Message: err.Error(),
			})
			return result, fmt.Errorf("configure git signing: %w", err)
		}
		result.Steps = append(result.Steps, StepResult{
			Name: "git-signing", Success: true, Message: "Git signing configured in ~/.gitconfig.local",
		})
	}

	return result, nil
}
