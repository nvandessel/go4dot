package machine

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GPGKey represents a GPG key
type GPGKey struct {
	KeyID       string
	UserID      string
	Email       string
	Fingerprint string
}

// DetectGPGKeys returns a list of available GPG signing keys
func DetectGPGKeys() ([]GPGKey, error) {
	// Check if gpg is available
	if _, err := exec.LookPath("gpg"); err != nil {
		return nil, nil // No GPG, not an error
	}

	// List secret keys
	cmd := exec.Command("gpg", "--list-secret-keys", "--keyid-format", "long")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// No keys or GPG not configured properly
		return nil, nil
	}

	return parseGPGOutput(stdout.String()), nil
}

// parseGPGOutput parses the output of gpg --list-secret-keys
func parseGPGOutput(output string) []GPGKey {
	var keys []GPGKey
	lines := strings.Split(output, "\n")

	var currentKey *GPGKey
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for sec line: sec   rsa4096/KEYID 2020-01-01 [SC]
		if strings.HasPrefix(line, "sec") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Extract key ID from "rsa4096/KEYID"
				keyPart := parts[1]
				if idx := strings.Index(keyPart, "/"); idx >= 0 {
					keyID := keyPart[idx+1:]
					currentKey = &GPGKey{KeyID: keyID}
				}
			}
		}

		// Look for uid line: uid           [ultimate] User Name <email@example.com>
		if strings.HasPrefix(line, "uid") && currentKey != nil {
			// Extract name and email
			if idx := strings.Index(line, "]"); idx >= 0 {
				userPart := strings.TrimSpace(line[idx+1:])
				currentKey.UserID = userPart

				// Extract email from <email>
				if emailStart := strings.Index(userPart, "<"); emailStart >= 0 {
					if emailEnd := strings.Index(userPart, ">"); emailEnd > emailStart {
						currentKey.Email = userPart[emailStart+1 : emailEnd]
					}
				}
			}

			keys = append(keys, *currentKey)
			currentKey = nil
		}
	}

	return keys
}

// GetGPGKeyByEmail finds a GPG key matching the given email
func GetGPGKeyByEmail(email string) (*GPGKey, error) {
	keys, err := DetectGPGKeys()
	if err != nil {
		return nil, err
	}

	email = strings.ToLower(email)
	for _, key := range keys {
		if strings.ToLower(key.Email) == email {
			return &key, nil
		}
	}

	return nil, nil
}

// HasGPGKey checks if any GPG keys are available
func HasGPGKey() bool {
	keys, err := DetectGPGKeys()
	if err != nil {
		return false
	}
	return len(keys) > 0
}

// GitConfigValue represents a git config value
type GitConfigValue struct {
	Key   string
	Value string
	Scope string // "global", "local", "system"
}

// GetGitConfig gets a git config value
func GetGitConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", "--global", "--get", key)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// Key not found is not an error
		return "", nil
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetGitUserName returns the configured git user.name
func GetGitUserName() (string, error) {
	return GetGitConfig("user.name")
}

// GetGitUserEmail returns the configured git user.email
func GetGitUserEmail() (string, error) {
	return GetGitConfig("user.email")
}

// GetGitSigningKey returns the configured git user.signingkey
func GetGitSigningKey() (string, error) {
	return GetGitConfig("user.signingkey")
}

// GitDefaults returns default values for git configuration based on current settings
func GitDefaults() map[string]string {
	defaults := make(map[string]string)

	if name, _ := GetGitUserName(); name != "" {
		defaults["user_name"] = name
	}

	if email, _ := GetGitUserEmail(); email != "" {
		defaults["user_email"] = email
	}

	if signingKey, _ := GetGitSigningKey(); signingKey != "" {
		defaults["signing_key"] = signingKey
	}

	return defaults
}

// SSHKey represents an SSH key
type SSHKey struct {
	Path        string
	Type        string // "rsa", "ed25519", etc.
	Fingerprint string
	Comment     string
	Loaded      bool   // true if key is in ssh-agent
	Source      string // "agent", "file", or "both"
}

// DetectSSHKeys returns a list of available SSH keys
func DetectSSHKeys() ([]SSHKey, error) {
	// Check if ssh-add is available
	if _, err := exec.LookPath("ssh-add"); err != nil {
		return nil, nil
	}

	// List loaded keys
	cmd := exec.Command("ssh-add", "-l")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		// No keys loaded or agent not running
		return nil, nil
	}

	return parseSSHOutput(stdout.String()), nil
}

// parseSSHOutput parses the output of ssh-add -l
func parseSSHOutput(output string) []SSHKey {
	var keys []SSHKey
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "no identities") {
			continue
		}

		// Format: 2048 SHA256:... /path/to/key (RSA)
		parts := strings.Fields(line)
		if len(parts) >= 4 {
			key := SSHKey{
				Fingerprint: parts[1],
				Loaded:      true,
				Source:      "agent",
			}

			// Path is second to last, type is in parentheses at end
			if len(parts) >= 3 {
				key.Path = parts[2]
			}

			// Extract type from last part (RSA), (ED25519), etc.
			lastPart := parts[len(parts)-1]
			if strings.HasPrefix(lastPart, "(") && strings.HasSuffix(lastPart, ")") {
				key.Type = strings.ToLower(lastPart[1 : len(lastPart)-1])
			}

			// Comment might be in the path or separate
			if len(parts) > 4 {
				key.Comment = strings.Join(parts[3:len(parts)-1], " ")
			}

			keys = append(keys, key)
		}
	}

	return keys
}

// HasSSHKey checks if any SSH keys are loaded
func HasSSHKey() bool {
	keys, err := DetectSSHKeys()
	if err != nil {
		return false
	}
	return len(keys) > 0
}

// SystemInfo returns useful system information for machine config
type SystemInfo struct {
	Username    string
	HomeDir     string
	Hostname    string
	GitUserName string
	GitEmail    string
	HasGPG      bool
	HasSSH      bool
}

// GetSystemInfo gathers system information useful for machine config
func GetSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{}

	// Get username
	cmd := exec.Command("whoami")
	if out, err := cmd.Output(); err == nil {
		info.Username = strings.TrimSpace(string(out))
	}

	// Get hostname
	cmd = exec.Command("hostname")
	if out, err := cmd.Output(); err == nil {
		info.Hostname = strings.TrimSpace(string(out))
	}

	// Get home directory
	if home, err := expandPath("~/"); err == nil {
		info.HomeDir = home
	}

	// Get git config
	info.GitUserName, _ = GetGitUserName()
	info.GitEmail, _ = GetGitUserEmail()

	// Check for GPG and SSH
	info.HasGPG = HasGPGKey()
	info.HasSSH = HasSSHKey()

	return info, nil
}

// FormatGPGKeyChoice formats a GPG key for display in a selection prompt
func FormatGPGKeyChoice(key GPGKey) string {
	return fmt.Sprintf("%s <%s> (%s)", key.UserID, key.Email, key.KeyID)
}

// FormatSSHKeyChoice formats an SSH key for display in a selection prompt
func FormatSSHKeyChoice(key SSHKey) string {
	return fmt.Sprintf("%s (%s)", key.Path, strings.ToUpper(key.Type))
}
