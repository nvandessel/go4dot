package machine

import (
	"testing"
)

func TestParseGPGOutput(t *testing.T) {
	// Sample GPG output
	output := `sec   rsa4096/ABCD1234EFGH5678 2020-01-01 [SC]
      FINGERPRINTFINGERPRINTFINGERPRINTFINGERPRINT
uid           [ultimate] John Doe <john@example.com>
ssb   rsa4096/IJKL9012MNOP3456 2020-01-01 [E]

sec   ed25519/QRST7890UVWX1234 2022-06-15 [SC]
      ANOTHERFINGERPRINTFINGERPRINT
uid           [ultimate] Jane Smith <jane@example.com>
ssb   cv25519/YZAB5678CDEF9012 2022-06-15 [E]`

	keys := parseGPGOutput(output)

	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys, got %d", len(keys))
	}

	// Check first key
	if keys[0].KeyID != "ABCD1234EFGH5678" {
		t.Errorf("First key ID mismatch: got %q", keys[0].KeyID)
	}
	if keys[0].Email != "john@example.com" {
		t.Errorf("First key email mismatch: got %q", keys[0].Email)
	}

	// Check second key
	if keys[1].KeyID != "QRST7890UVWX1234" {
		t.Errorf("Second key ID mismatch: got %q", keys[1].KeyID)
	}
	if keys[1].Email != "jane@example.com" {
		t.Errorf("Second key email mismatch: got %q", keys[1].Email)
	}
}

func TestParseGPGOutputEmpty(t *testing.T) {
	keys := parseGPGOutput("")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys for empty output, got %d", len(keys))
	}
}

func TestParseSSHOutput(t *testing.T) {
	output := `2048 SHA256:abcdefghijklmnop /home/user/.ssh/id_rsa (RSA)
256 SHA256:qrstuvwxyz123456 /home/user/.ssh/id_ed25519 (ED25519)`

	keys := parseSSHOutput(output)

	if len(keys) != 2 {
		t.Fatalf("Expected 2 keys, got %d", len(keys))
	}

	if keys[0].Type != "rsa" {
		t.Errorf("First key type mismatch: got %q", keys[0].Type)
	}
	if keys[0].Fingerprint != "SHA256:abcdefghijklmnop" {
		t.Errorf("First key fingerprint mismatch: got %q", keys[0].Fingerprint)
	}

	if keys[1].Type != "ed25519" {
		t.Errorf("Second key type mismatch: got %q", keys[1].Type)
	}

	// Verify Loaded and Source fields for all parsed keys
	for i, key := range keys {
		if !key.Loaded {
			t.Errorf("Key %d: expected Loaded=true, got false", i)
		}
		if key.Source != "agent" {
			t.Errorf("Key %d: expected Source=\"agent\", got %q", i, key.Source)
		}
	}
}

func TestParseSSHOutputNoIdentities(t *testing.T) {
	output := "The agent has no identities."
	keys := parseSSHOutput(output)
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
}

func TestParseSSHOutputEmpty(t *testing.T) {
	keys := parseSSHOutput("")
	if len(keys) != 0 {
		t.Errorf("Expected 0 keys, got %d", len(keys))
	}
}

func TestFormatGPGKeyChoice(t *testing.T) {
	key := GPGKey{
		KeyID:  "ABCD1234",
		UserID: "John Doe <john@example.com>",
		Email:  "john@example.com",
	}

	result := FormatGPGKeyChoice(key)
	expected := "John Doe <john@example.com> <john@example.com> (ABCD1234)"
	if result != expected {
		t.Errorf("FormatGPGKeyChoice mismatch: got %q, want %q", result, expected)
	}
}

func TestFormatSSHKeyChoice(t *testing.T) {
	key := SSHKey{
		Path:   "/home/user/.ssh/id_ed25519",
		Type:   "ed25519",
		Loaded: true,
		Source: "agent",
	}

	result := FormatSSHKeyChoice(key)
	expected := "/home/user/.ssh/id_ed25519 (ED25519)"
	if result != expected {
		t.Errorf("FormatSSHKeyChoice mismatch: got %q, want %q", result, expected)
	}
}

func TestGitDefaults(t *testing.T) {
	// This test may or may not have values depending on the system
	defaults := GitDefaults()

	// Just verify it returns a map without error
	if defaults == nil {
		t.Error("GitDefaults returned nil")
	}

	// Log what we found (informational)
	t.Logf("Git defaults found: %+v", defaults)
}

func TestGetSystemInfo(t *testing.T) {
	info, err := GetSystemInfo()
	if err != nil {
		t.Fatalf("GetSystemInfo failed: %v", err)
	}

	// Just verify basic fields are populated
	if info.HomeDir == "" {
		t.Error("HomeDir should not be empty")
	}

	// Log what we found (informational)
	t.Logf("System info: username=%s, hostname=%s, hasGPG=%v, hasSSH=%v",
		info.Username, info.Hostname, info.HasGPG, info.HasSSH)
}

func TestDetectGPGKeys(t *testing.T) {
	// This test depends on the system having gpg installed
	keys, err := DetectGPGKeys()
	if err != nil {
		t.Fatalf("DetectGPGKeys failed: %v", err)
	}

	// Just log what we found (may be empty)
	t.Logf("Found %d GPG keys", len(keys))
	for _, key := range keys {
		t.Logf("  - %s <%s> (%s)", key.UserID, key.Email, key.KeyID)
	}
}

func TestDetectSSHKeys(t *testing.T) {
	// This test depends on the system having ssh-agent running
	keys, err := DetectSSHKeys()
	if err != nil {
		t.Fatalf("DetectSSHKeys failed: %v", err)
	}

	// Just log what we found (may be empty)
	t.Logf("Found %d SSH keys", len(keys))
	for _, key := range keys {
		t.Logf("  - %s (%s)", key.Path, key.Type)
	}
}

func TestHasGPGKey(t *testing.T) {
	// This just verifies it doesn't panic
	hasGPG := HasGPGKey()
	t.Logf("HasGPGKey: %v", hasGPG)
}

func TestHasSSHKey(t *testing.T) {
	// This just verifies it doesn't panic
	hasSSH := HasSSHKey()
	t.Logf("HasSSHKey: %v", hasSSH)
}

func TestGetGitConfig(t *testing.T) {
	// Test getting a config value
	// This will return empty if not configured, which is fine
	name, err := GetGitConfig("user.name")
	if err != nil {
		t.Fatalf("GetGitConfig failed: %v", err)
	}
	t.Logf("user.name: %q", name)

	email, err := GetGitConfig("user.email")
	if err != nil {
		t.Fatalf("GetGitConfig failed: %v", err)
	}
	t.Logf("user.email: %q", email)
}

func TestGetGitUserName(t *testing.T) {
	name, err := GetGitUserName()
	if err != nil {
		t.Fatalf("GetGitUserName failed: %v", err)
	}
	t.Logf("Git user.name: %q", name)
}

func TestGetGitUserEmail(t *testing.T) {
	email, err := GetGitUserEmail()
	if err != nil {
		t.Fatalf("GetGitUserEmail failed: %v", err)
	}
	t.Logf("Git user.email: %q", email)
}

func TestGetGPGKeyByEmail(t *testing.T) {
	// This test depends on having GPG keys set up
	email := "test@example.com" // Unlikely to match
	key, err := GetGPGKeyByEmail(email)
	if err != nil {
		t.Fatalf("GetGPGKeyByEmail failed: %v", err)
	}

	// Most likely will be nil since the email doesn't exist
	if key != nil {
		t.Logf("Found key for %s: %s", email, key.KeyID)
	} else {
		t.Logf("No key found for %s (expected)", email)
	}
}
