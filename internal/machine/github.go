package machine

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// Commander abstracts command execution for testability.
type Commander interface {
	Run(name string, args ...string) ([]byte, error)
}

// ExecCommander is the default implementation using os/exec.
type ExecCommander struct{}

// Run executes a command and returns combined output.
func (e *ExecCommander) Run(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// GitHubClient handles GitHub operations via gh CLI.
type GitHubClient struct {
	Commander Commander
}

// NewGitHubClient creates a GitHubClient with default ExecCommander.
func NewGitHubClient() *GitHubClient {
	return &GitHubClient{Commander: &ExecCommander{}}
}

// getCommander returns the configured commander or default.
func (c *GitHubClient) getCommander() Commander {
	if c.Commander != nil {
		return c.Commander
	}
	return &ExecCommander{}
}

// GitHubSSHKey represents an SSH key registered on GitHub.
type GitHubSSHKey struct {
	ID    json.Number `json:"id"`
	Key   string      `json:"key"`
	Title string      `json:"title"`
}

// HasGHCLI checks if the gh CLI tool is installed.
func HasGHCLI() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}

// IsAuthenticated checks if gh is authenticated with GitHub.
// Stderr is discarded to prevent token leakage.
func (c *GitHubClient) IsAuthenticated() (bool, error) {
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = nil
	cmd.Stderr = nil // Discard stderr - may contain token info
	err := cmd.Run()
	if err != nil {
		return false, nil
	}
	return true, nil
}

// AddSSHKey registers an SSH public key with GitHub.
func (c *GitHubClient) AddSSHKey(pubkeyPath, title, sshDir string) error {
	if err := validation.ValidateSSHKeyPath(pubkeyPath, sshDir); err != nil {
		return fmt.Errorf("invalid public key path: %w", err)
	}
	if !strings.HasSuffix(pubkeyPath, ".pub") {
		return fmt.Errorf("path must be a .pub file: %q", pubkeyPath)
	}
	if err := validation.ValidateKeyTitle(title); err != nil {
		return fmt.Errorf("invalid key title: %w", err)
	}

	output, err := c.getCommander().Run("gh", "ssh-key", "add", pubkeyPath, "--title", title)
	if err != nil {
		return fmt.Errorf("failed to add SSH key to GitHub: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// AddGPGKey registers a GPG key with GitHub using pipe: gpg --export | gh gpg-key add.
func (c *GitHubClient) AddGPGKey(keyID string) error {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return fmt.Errorf("invalid GPG key ID: %w", err)
	}

	// Use exec.Command directly for pipe orchestration
	gpgCmd := exec.Command("gpg", "--armor", "--export", keyID)
	ghCmd := exec.Command("gh", "gpg-key", "add")

	gpgOut, err := gpgCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	ghCmd.Stdin = gpgOut
	gpgCmd.Stderr = os.Stderr
	ghCmd.Stderr = os.Stderr

	if err := gpgCmd.Start(); err != nil {
		return fmt.Errorf("failed to start gpg: %w", err)
	}
	if err := ghCmd.Start(); err != nil {
		_ = gpgCmd.Wait()
		return fmt.Errorf("failed to start gh: %w", err)
	}

	gpgErr := gpgCmd.Wait()
	ghErr := ghCmd.Wait()

	if gpgErr != nil {
		return fmt.Errorf("gpg export failed: %w", gpgErr)
	}
	if ghErr != nil {
		return fmt.Errorf("gh gpg-key add failed: %w", ghErr)
	}
	return nil
}

// ListSSHKeys returns SSH keys registered on GitHub.
func (c *GitHubClient) ListSSHKeys() ([]GitHubSSHKey, error) {
	output, err := c.getCommander().Run("gh", "ssh-key", "list", "--json", "id,key,title")
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub SSH keys: %w\nOutput: %s", err, string(output))
	}

	var keys []GitHubSSHKey
	if err := json.Unmarshal(output, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub SSH keys: %w", err)
	}
	return keys, nil
}

// IsKeyRegistered checks if a local SSH public key is already registered on GitHub.
func (c *GitHubClient) IsKeyRegistered(pubkeyPath, sshDir string) (bool, error) {
	// Read local public key
	localKey, err := GetSSHPublicKey(pubkeyPath, sshDir)
	if err != nil {
		return false, err
	}

	// Extract key material (second field: type base64 comment)
	localParts := strings.Fields(localKey)
	if len(localParts) < 2 {
		return false, fmt.Errorf("invalid public key format")
	}
	localMaterial := localParts[1]

	// Get GitHub keys
	ghKeys, err := c.ListSSHKeys()
	if err != nil {
		return false, err
	}

	// Compare key material
	for _, ghKey := range ghKeys {
		ghParts := strings.Fields(ghKey.Key)
		if len(ghParts) >= 2 && ghParts[1] == localMaterial {
			return true, nil
		}
	}
	return false, nil
}
