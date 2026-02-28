package machine

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nvandessel/go4dot/internal/validation"
)

// SSHKeygenOpts configures SSH key generation.
type SSHKeygenOpts struct {
	Email  string // Required. Validated with ValidateEmail().
	Name   string // Key filename. Default: "id_ed25519". No path separators allowed.
	SSHDir string // Base directory. Default: ~/.ssh. All paths validated within this.
}

// DefaultSSHDir is the default SSH directory.
const DefaultSSHDir = "~/.ssh"

// DefaultKeyName is the default SSH key filename.
const DefaultKeyName = "id_ed25519"

// DetectSSHKeyFiles scans sshDir for .pub files and returns enriched SSHKey structs.
// Symlinks are skipped entirely for security.
func DetectSSHKeyFiles(sshDir string) ([]SSHKey, error) {
	entries, err := os.ReadDir(sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read SSH directory: %w", err)
	}

	var keys []SSHKey
	for _, entry := range entries {
		// Skip symlinks entirely
		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		// Skip directories
		if entry.IsDir() {
			continue
		}
		// Only process .pub files
		if !strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}

		pubPath := filepath.Join(sshDir, entry.Name())

		// Validate path is within sshDir
		if err := validation.ValidateSSHKeyPath(pubPath, sshDir); err != nil {
			continue // Skip invalid paths
		}

		// Read and parse the pub key
		content, err := os.ReadFile(pubPath)
		if err != nil {
			continue
		}

		key := parsePublicKey(string(content))
		// Private key path is .pub suffix removed
		key.Path = strings.TrimSuffix(pubPath, ".pub")
		key.Source = "file"

		// Try to get fingerprint
		fingerprint := getKeyFingerprint(pubPath)
		if fingerprint != "" {
			key.Fingerprint = fingerprint
		}

		keys = append(keys, key)
	}

	return keys, nil
}

// parsePublicKey parses a single line from a .pub file: <type> <base64> [comment]
func parsePublicKey(content string) SSHKey {
	line := strings.TrimSpace(strings.SplitN(content, "\n", 2)[0])
	parts := strings.Fields(line)

	key := SSHKey{}
	if len(parts) >= 1 {
		// Map ssh key type names to short forms
		key.Type = normalizeKeyType(parts[0])
	}
	if len(parts) >= 3 {
		key.Comment = strings.Join(parts[2:], " ")
	}
	return key
}

// normalizeKeyType converts SSH key type identifiers to short form.
func normalizeKeyType(keyType string) string {
	switch keyType {
	case "ssh-rsa":
		return "rsa"
	case "ssh-ed25519":
		return "ed25519"
	case "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521":
		return "ecdsa"
	case "ssh-dss":
		return "dsa"
	default:
		return keyType
	}
}

// getKeyFingerprint runs ssh-keygen -lf to get a key's fingerprint.
func getKeyFingerprint(pubPath string) string {
	cmd := exec.Command("ssh-keygen", "-lf", pubPath)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	// Output format: 256 SHA256:xxx comment (TYPE)
	parts := strings.Fields(stdout.String())
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// DetectAllSSHKeys merges agent keys (from DetectSSHKeys in git.go) with file keys.
// Deduplicates by Path; agent keys win (set Loaded: true, Source: "both").
func DetectAllSSHKeys(sshDir string) ([]SSHKey, error) {
	agentKeys, _ := DetectSSHKeys() // from git.go
	fileKeys, err := DetectSSHKeyFiles(sshDir)
	if err != nil {
		return agentKeys, err
	}

	// Index agent keys by path
	agentMap := make(map[string]*SSHKey)
	for i := range agentKeys {
		agentMap[agentKeys[i].Path] = &agentKeys[i]
	}

	// Merge file keys
	var merged []SSHKey
	merged = append(merged, agentKeys...)

	for _, fk := range fileKeys {
		if ak, exists := agentMap[fk.Path]; exists {
			// Agent key wins, but mark as "both"
			ak.Source = "both"
			// Fill in missing fields from file key
			if ak.Comment == "" {
				ak.Comment = fk.Comment
			}
		} else {
			merged = append(merged, fk)
		}
	}

	return merged, nil
}

// GenerateSSHKey creates a new ed25519 SSH key pair.
// The function is interactive - ssh-keygen will prompt for passphrase.
func GenerateSSHKey(opts SSHKeygenOpts) (string, error) {
	// Validate email
	if err := validation.ValidateEmail(opts.Email); err != nil {
		return "", fmt.Errorf("invalid email: %w", err)
	}

	// Apply defaults
	if opts.Name == "" {
		opts.Name = DefaultKeyName
	}
	if opts.SSHDir == "" {
		opts.SSHDir = DefaultSSHDir
	}

	// Resolve and validate sshDir
	expandedSSHDir, err := expandSSHDir(opts.SSHDir)
	if err != nil {
		return "", fmt.Errorf("invalid SSH directory: %w", err)
	}

	// Build key path and validate
	keyPath := filepath.Join(expandedSSHDir, opts.Name)
	if err := validation.ValidateSSHKeyPath(keyPath, expandedSSHDir); err != nil {
		return "", fmt.Errorf("invalid key path: %w", err)
	}

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		return "", fmt.Errorf("key already exists: %s", keyPath)
	}

	// Ensure sshDir exists with proper permissions
	if err := os.MkdirAll(expandedSSHDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create SSH directory: %w", err)
	}

	// Generate the key - interactive (ssh-keygen handles passphrase prompt)
	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", opts.Email, "-f", keyPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen failed: %w", err)
	}

	// Verify permissions on generated key
	info, err := os.Stat(keyPath)
	if err != nil {
		return "", fmt.Errorf("generated key not found: %w", err)
	}
	if info.Mode().Perm() != 0600 {
		return "", fmt.Errorf("generated key has unexpected permissions: %v (expected 0600)", info.Mode().Perm())
	}

	// Verify .pub exists
	pubPath := keyPath + ".pub"
	if _, err := os.Stat(pubPath); err != nil {
		return "", fmt.Errorf("generated public key not found: %w", err)
	}

	return keyPath, nil
}

// AddKeyToAgent adds an SSH key to the running ssh-agent.
// Interactive - ssh-add will prompt for passphrase if the key is protected.
func AddKeyToAgent(keyPath, sshDir string) error {
	if err := validation.ValidateSSHKeyPath(keyPath, sshDir); err != nil {
		return fmt.Errorf("invalid key path: %w", err)
	}

	if !IsAgentRunning() {
		return fmt.Errorf("SSH agent not running. Start with: eval $(ssh-agent -s)")
	}

	cmd := exec.Command("ssh-add", keyPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-add failed: %w", err)
	}
	return nil
}

// GetSSHPublicKey reads an SSH public key file. Path must be within sshDir and end in .pub.
func GetSSHPublicKey(keyPath, sshDir string) (string, error) {
	if !strings.HasSuffix(keyPath, ".pub") {
		return "", fmt.Errorf("path must be a .pub file: %q", keyPath)
	}
	if err := validation.ValidateSSHKeyPath(keyPath, sshDir); err != nil {
		return "", fmt.Errorf("invalid key path: %w", err)
	}

	content, err := os.ReadFile(keyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}
	return strings.TrimSpace(string(content)), nil
}

// IsAgentRunning checks if an SSH agent is running and accessible.
func IsAgentRunning() bool {
	cmd := exec.Command("ssh-add", "-l")
	err := cmd.Run()
	if err != nil {
		// Exit code 2 means no agent; exit code 1 means agent running but no keys
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode() != 2
		}
		return false
	}
	return true // Exit code 0 means agent running with keys
}

// expandSSHDir expands ~ prefix and ensures the directory path is valid.
func expandSSHDir(sshDir string) (string, error) {
	if strings.HasPrefix(sshDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		sshDir = filepath.Join(home, sshDir[2:])
	}

	sshDir = filepath.Clean(sshDir)

	if !filepath.IsAbs(sshDir) {
		return "", fmt.Errorf("SSH directory must be absolute: %q", sshDir)
	}

	return sshDir, nil
}
