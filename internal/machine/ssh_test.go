package machine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectSSHKeyFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake .pub file
	pubContent := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test@example.com\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "id_ed25519.pub"), []byte(pubContent), 0644); err != nil {
		t.Fatalf("failed to write pub file: %v", err)
	}

	keys, err := DetectSSHKeyFiles(tmpDir)
	if err != nil {
		t.Fatalf("DetectSSHKeyFiles failed: %v", err)
	}

	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	key := keys[0]
	if key.Path != filepath.Join(tmpDir, "id_ed25519") {
		t.Errorf("Path = %q, want %q", key.Path, filepath.Join(tmpDir, "id_ed25519"))
	}
	if key.Type != "ed25519" {
		t.Errorf("Type = %q, want %q", key.Type, "ed25519")
	}
	if key.Comment != "test@example.com" {
		t.Errorf("Comment = %q, want %q", key.Comment, "test@example.com")
	}
	if key.Source != "file" {
		t.Errorf("Source = %q, want %q", key.Source, "file")
	}
	if key.Loaded {
		t.Error("Loaded should be false for file keys")
	}
}

func TestDetectSSHKeyFilesEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	keys, err := DetectSSHKeyFiles(tmpDir)
	if err != nil {
		t.Fatalf("DetectSSHKeyFiles failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected 0 keys for empty dir, got %d", len(keys))
	}
}

func TestDetectSSHKeyFilesNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does_not_exist")
	keys, err := DetectSSHKeyFiles(nonExistent)
	if err != nil {
		t.Fatalf("expected nil error for nonexistent dir, got: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for nonexistent dir, got %d", len(keys))
	}
}

func TestDetectSSHKeyFilesSkipsSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a real pub file in a separate location
	realDir := t.TempDir()
	realPub := filepath.Join(realDir, "real_key.pub")
	if err := os.WriteFile(realPub, []byte("ssh-ed25519 AAAAC3 test@example.com\n"), 0644); err != nil {
		t.Fatalf("failed to write real pub file: %v", err)
	}

	// Create a symlink to it in the SSH dir
	symlinkPath := filepath.Join(tmpDir, "linked_key.pub")
	if err := os.Symlink(realPub, symlinkPath); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}

	keys, err := DetectSSHKeyFiles(tmpDir)
	if err != nil {
		t.Fatalf("DetectSSHKeyFiles failed: %v", err)
	}

	// Symlinks should be skipped
	if len(keys) != 0 {
		t.Errorf("expected 0 keys (symlinks skipped), got %d", len(keys))
	}
}

func TestDetectSSHKeyFilesSkipsNonPub(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-.pub files
	if err := os.WriteFile(filepath.Join(tmpDir, "id_ed25519"), []byte("private key data"), 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config"), []byte("Host *"), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "known_hosts"), []byte("host key"), 0644); err != nil {
		t.Fatalf("failed to write known_hosts: %v", err)
	}

	keys, err := DetectSSHKeyFiles(tmpDir)
	if err != nil {
		t.Fatalf("DetectSSHKeyFiles failed: %v", err)
	}

	if len(keys) != 0 {
		t.Errorf("expected 0 keys (non-.pub skipped), got %d", len(keys))
	}
}

func TestDetectAllSSHKeys(t *testing.T) {
	t.Run("empty dir returns no error", func(t *testing.T) {
		tmpDir := t.TempDir()
		keys, err := DetectAllSSHKeys(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// May include agent keys from the host system
		t.Logf("Detected %d total SSH keys (agent keys may be included)", len(keys))
	})

	t.Run("with pub files", func(t *testing.T) {
		tmpDir := t.TempDir()
		// Create a .pub file
		pubContent := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test@example.com\n"
		if err := os.WriteFile(filepath.Join(tmpDir, "id_test.pub"), []byte(pubContent), 0644); err != nil {
			t.Fatalf("failed to create pub file: %v", err)
		}
		keys, err := DetectAllSSHKeys(tmpDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have at least the file key we created
		found := false
		for _, k := range keys {
			if filepath.Base(k.Path) == "id_test" {
				found = true
				if k.Source != "file" && k.Source != "both" {
					t.Errorf("expected Source 'file' or 'both', got %q", k.Source)
				}
			}
		}
		if !found {
			t.Error("expected to find id_test key in results")
		}
	})
}

func TestSSHKeygenDirectInvocation(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not available")
	}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "test_key")

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", "test@example.com", "-f", keyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		t.Fatalf("ssh-keygen failed: %v", err)
	}

	// Verify private key exists with correct permissions
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("private key not found: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("private key permissions = %v, want 0600", info.Mode().Perm())
	}

	// Verify public key exists
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Fatalf("public key not found: %v", err)
	}
}

func TestGenerateSSHKeyExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "existing_key")

	// Pre-create the file
	if err := os.WriteFile(keyPath, []byte("existing"), 0600); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	_, err := GenerateSSHKey(SSHKeygenOpts{
		Email:  "test@example.com",
		Name:   "existing_key",
		SSHDir: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error for existing key, got nil")
	}
	if !strings.Contains(err.Error(), "key already exists") {
		t.Errorf("expected 'key already exists' error, got: %v", err)
	}
}

func TestGenerateSSHKeyBadEmail(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := GenerateSSHKey(SSHKeygenOpts{
		Email:  "-evil@example.com",
		Name:   "test_key",
		SSHDir: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error for bad email, got nil")
	}
	if !strings.Contains(err.Error(), "invalid email") {
		t.Errorf("expected 'invalid email' error, got: %v", err)
	}
}

func TestGenerateSSHKeyTraversalPath(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := GenerateSSHKey(SSHKeygenOpts{
		Email:  "test@example.com",
		Name:   "../escape",
		SSHDir: tmpDir,
	})
	if err == nil {
		t.Fatal("expected error for traversal path, got nil")
	}
}

func TestGetSSHPublicKey(t *testing.T) {
	tmpDir := t.TempDir()
	pubContent := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey test@example.com"
	pubPath := filepath.Join(tmpDir, "test_key.pub")

	if err := os.WriteFile(pubPath, []byte(pubContent), 0644); err != nil {
		t.Fatalf("failed to write pub file: %v", err)
	}

	content, err := GetSSHPublicKey(pubPath, tmpDir)
	if err != nil {
		t.Fatalf("GetSSHPublicKey failed: %v", err)
	}

	if content != pubContent {
		t.Errorf("content = %q, want %q", content, pubContent)
	}
}

func TestGetSSHPublicKeyNotPub(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")

	_, err := GetSSHPublicKey(keyPath, tmpDir)
	if err == nil {
		t.Fatal("expected error for non-.pub path, got nil")
	}
	if !strings.Contains(err.Error(), ".pub") {
		t.Errorf("expected '.pub' in error, got: %v", err)
	}
}

func TestGetSSHPublicKeyTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	outsidePath := "/tmp/outside_key.pub"

	_, err := GetSSHPublicKey(outsidePath, tmpDir)
	if err == nil {
		t.Fatal("expected error for path outside sshDir, got nil")
	}
}

func TestAddKeyToAgentBadPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Path with traversal should fail validation
	err := AddKeyToAgent("/tmp/../etc/shadow", tmpDir)
	if err == nil {
		t.Fatal("expected error for bad path, got nil")
	}
	if !strings.Contains(err.Error(), "invalid key path") {
		t.Errorf("expected 'invalid key path' error, got: %v", err)
	}
}

func TestIsAgentRunning(t *testing.T) {
	// Smoke test - just verify it doesn't panic
	result := IsAgentRunning()
	t.Logf("IsAgentRunning: %v", result)
}

func TestExpandSSHDir(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantAbs bool
		wantErr bool
	}{
		{name: "tilde expansion", input: "~/.ssh", wantAbs: true, wantErr: false},
		{name: "bare tilde", input: "~", wantAbs: true, wantErr: false},
		{name: "absolute path", input: "/home/user/.ssh", wantAbs: true, wantErr: false},
		{name: "relative path", input: ".ssh", wantAbs: false, wantErr: true},
		{name: "relative with dots", input: "../.ssh", wantAbs: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandSSHDir(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandSSHDir(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.wantAbs && !filepath.IsAbs(result) {
				t.Errorf("expandSSHDir(%q) = %q, expected absolute path", tt.input, result)
			}
		})
	}
}

func TestParsePublicKey(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantType    string
		wantComment string
	}{
		{
			name:        "ed25519 with comment",
			content:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestKey user@host",
			wantType:    "ed25519",
			wantComment: "user@host",
		},
		{
			name:        "rsa with comment",
			content:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAA user@example.com",
			wantType:    "rsa",
			wantComment: "user@example.com",
		},
		{
			name:        "ecdsa",
			content:     "ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTY= comment here",
			wantType:    "ecdsa",
			wantComment: "comment here",
		},
		{
			name:        "no comment",
			content:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5",
			wantType:    "ed25519",
			wantComment: "",
		},
		{
			name:        "empty content",
			content:     "",
			wantType:    "",
			wantComment: "",
		},
		{
			name:        "multi-word comment",
			content:     "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAA John Doe (work)",
			wantType:    "rsa",
			wantComment: "John Doe (work)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := parsePublicKey(tt.content)
			if key.Type != tt.wantType {
				t.Errorf("Type = %q, want %q", key.Type, tt.wantType)
			}
			if key.Comment != tt.wantComment {
				t.Errorf("Comment = %q, want %q", key.Comment, tt.wantComment)
			}
		})
	}
}

func TestNormalizeKeyType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ssh-rsa", "rsa"},
		{"ssh-ed25519", "ed25519"},
		{"ecdsa-sha2-nistp256", "ecdsa"},
		{"ecdsa-sha2-nistp384", "ecdsa"},
		{"ecdsa-sha2-nistp521", "ecdsa"},
		{"ssh-dss", "dsa"},
		{"unknown-type", "unknown-type"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeKeyType(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeKeyType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
