package machine

import (
	"bytes"
	"encoding/json"
	"fmt"
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
// Output is discarded to prevent token leakage.
func (c *GitHubClient) IsAuthenticated() (bool, error) {
	_, err := c.getCommander().Run("gh", "auth", "status")
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

// AddGPGKey registers a GPG key with GitHub.
// Exports the key via gpg, validates it's non-empty, then adds via gh.
func (c *GitHubClient) AddGPGKey(keyID string) error {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return fmt.Errorf("invalid GPG key ID: %w", err)
	}

	// Export GPG key first to detect missing-key case
	gpgOut, err := exec.Command("gpg", "--armor", "--export", keyID).Output()
	if err != nil {
		return fmt.Errorf("gpg export failed: %w", err)
	}
	if len(strings.TrimSpace(string(gpgOut))) == 0 {
		return fmt.Errorf("gpg key %q not found in local keyring", keyID)
	}

	// Add to GitHub via gh CLI
	ghCmd := exec.Command("gh", "gpg-key", "add")
	ghCmd.Stdin = strings.NewReader(string(gpgOut))
	var ghStderr bytes.Buffer
	ghCmd.Stderr = &ghStderr
	if err := ghCmd.Run(); err != nil {
		return fmt.Errorf("gh gpg-key add failed: %w\nStderr: %s", err, ghStderr.String())
	}
	return nil
}

// GitHubGPGSubkey represents a subkey within a GitHub GPG key.
type GitHubGPGSubkey struct {
	ID    json.Number `json:"id"`
	KeyID string      `json:"key_id"`
}

// GitHubGPGKey represents a GPG key registered on GitHub.
type GitHubGPGKey struct {
	ID      json.Number       `json:"id"`
	KeyID   string            `json:"key_id"`
	Email   string            `json:"email"`
	Subkeys []GitHubGPGSubkey `json:"subkeys"`
}

// ListGPGKeys returns GPG keys registered on GitHub.
// Uses gh api because gh gpg-key list does not support --json output.
func (c *GitHubClient) ListGPGKeys() ([]GitHubGPGKey, error) {
	output, err := c.getCommander().Run("gh", "api", "/user/gpg_keys")
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub GPG keys: %w\nOutput: %s", err, string(output))
	}

	var keys []GitHubGPGKey
	if err := json.Unmarshal(output, &keys); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub GPG keys: %w", err)
	}
	return keys, nil
}

// IsGPGKeyRegistered checks if a local GPG key is already registered on GitHub.
func (c *GitHubClient) IsGPGKeyRegistered(keyID string) (bool, error) {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return false, fmt.Errorf("invalid GPG key ID: %w", err)
	}

	ghKeys, err := c.ListGPGKeys()
	if err != nil {
		return false, fmt.Errorf("list GitHub GPG keys: %w", err)
	}

	// Normalize to uppercase for comparison (GPG key IDs are hex).
	// Only exact match — suffix matching risks false positives (Evil32 attack).
	normalizedKeyID := strings.ToUpper(keyID)
	for _, ghKey := range ghKeys {
		if strings.ToUpper(ghKey.KeyID) == normalizedKeyID {
			return true, nil
		}
	}
	return false, nil
}

// NeedsGPGKeyReupload checks if the local key has subkeys not yet on GitHub.
// Returns true if local subkeys don't match what GitHub has registered.
func (c *GitHubClient) NeedsGPGKeyReupload(localKeyID string, localSubkeyIDs []string) (bool, error) {
	if err := validation.ValidateGPGKeyID(localKeyID); err != nil {
		return false, fmt.Errorf("invalid GPG key ID: %w", err)
	}

	ghKeys, err := c.ListGPGKeys()
	if err != nil {
		return false, fmt.Errorf("list GitHub GPG keys: %w", err)
	}

	// Find the matching GitHub key
	normalizedKeyID := strings.ToUpper(localKeyID)
	var matchingKey *GitHubGPGKey
	for i, ghKey := range ghKeys {
		if strings.ToUpper(ghKey.KeyID) == normalizedKeyID {
			matchingKey = &ghKeys[i]
			break
		}
	}

	if matchingKey == nil {
		// Key not on GitHub at all — needs upload, not re-upload
		return false, nil
	}

	// Build set of GitHub subkey IDs (uppercase)
	ghSubkeySet := make(map[string]bool)
	for _, sub := range matchingKey.Subkeys {
		ghSubkeySet[strings.ToUpper(sub.KeyID)] = true
	}

	// Check if any local subkey is missing from GitHub
	for _, localSubFP := range localSubkeyIDs {
		// Extract the short key ID (last 16 chars) from fingerprint for comparison
		// GitHub returns short key IDs, local gpg returns full fingerprints
		localID := strings.ToUpper(localSubFP)
		if len(localID) > 16 {
			localID = localID[len(localID)-16:]
		}
		if !ghSubkeySet[localID] {
			return true, nil
		}
	}

	return false, nil
}

// DeleteGPGKey removes a GPG key from GitHub by its GitHub key ID.
func (c *GitHubClient) DeleteGPGKey(githubKeyID string) error {
	output, err := c.getCommander().Run("gh", "api", "-X", "DELETE",
		fmt.Sprintf("/user/gpg_keys/%s", githubKeyID))
	if err != nil {
		return fmt.Errorf("failed to delete GitHub GPG key %s: %w\nOutput: %s", githubKeyID, err, string(output))
	}
	return nil
}

// ReuploadGPGKey deletes and re-adds a GPG key on GitHub.
// GitHub's API doesn't support updating GPG keys, so we delete and re-add.
func (c *GitHubClient) ReuploadGPGKey(keyID string) error {
	if err := validation.ValidateGPGKeyID(keyID); err != nil {
		return fmt.Errorf("invalid GPG key ID: %w", err)
	}

	// Find the GitHub key ID for this GPG key
	ghKeys, err := c.ListGPGKeys()
	if err != nil {
		return fmt.Errorf("list GitHub GPG keys: %w", err)
	}

	normalizedKeyID := strings.ToUpper(keyID)
	var githubID string
	for _, ghKey := range ghKeys {
		if strings.ToUpper(ghKey.KeyID) == normalizedKeyID {
			githubID = ghKey.ID.String()
			break
		}
	}

	if githubID == "" {
		return fmt.Errorf("GPG key %s not found on GitHub", keyID)
	}

	// Delete existing
	if err := c.DeleteGPGKey(githubID); err != nil {
		return fmt.Errorf("delete existing key: %w", err)
	}

	// Re-add
	if err := c.AddGPGKey(keyID); err != nil {
		return fmt.Errorf("re-add key after delete: %w", err)
	}

	return nil
}

// ListSSHKeys returns SSH keys registered on GitHub.
// Uses gh api because gh ssh-key list does not support --json output.
func (c *GitHubClient) ListSSHKeys() ([]GitHubSSHKey, error) {
	output, err := c.getCommander().Run("gh", "api", "user/keys")
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
		return false, fmt.Errorf("read local SSH public key: %w", err)
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
		return false, fmt.Errorf("list GitHub SSH keys: %w", err)
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
