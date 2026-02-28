package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// mockCommander is a test double for Commander.
type mockCommander struct {
	output []byte
	err    error
}

func (m *mockCommander) Run(name string, args ...string) ([]byte, error) {
	return m.output, m.err
}

func TestHasGHCLI(t *testing.T) {
	// Smoke test - just verify no panic
	result := HasGHCLI()
	t.Logf("HasGHCLI: %v", result)
}

func TestGitHubClient_AddSSHKey_BadPath(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{}}
	err := client.AddSSHKey("/invalid/../path.pub", "title", "/ssh")
	if err == nil {
		t.Error("expected error for bad path")
	}
}

func TestGitHubClient_AddSSHKey_NotPubFile(t *testing.T) {
	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "id_ed25519")
	if err := os.WriteFile(keyPath, []byte("ssh-ed25519 AAAA test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &GitHubClient{Commander: &mockCommander{}}
	err := client.AddSSHKey(keyPath, "title", tmpDir)
	if err == nil {
		t.Error("expected error for non-.pub file")
	}
}

func TestGitHubClient_AddSSHKey_BadTitle(t *testing.T) {
	// Create a valid .pub file in tmpdir
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &GitHubClient{Commander: &mockCommander{}}
	err := client.AddSSHKey(pubPath, "-evil", tmpDir)
	if err == nil {
		t.Error("expected error for bad title")
	}
}

func TestGitHubClient_AddSSHKey_Success(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &GitHubClient{Commander: &mockCommander{output: []byte("ok")}}
	err := client.AddSSHKey(pubPath, "my-key", tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGitHubClient_AddSSHKey_CommandFailure(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &GitHubClient{Commander: &mockCommander{output: []byte("error"), err: fmt.Errorf("gh failed")}}
	err := client.AddSSHKey(pubPath, "my-key", tmpDir)
	if err == nil {
		t.Error("expected error for command failure")
	}
}

func TestGitHubClient_AddGPGKey_BadKeyID(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{}}
	err := client.AddGPGKey("not-hex!")
	if err == nil {
		t.Error("expected error for non-hex key ID")
	}
}

func TestGitHubClient_AddGPGKey_Empty(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{}}
	err := client.AddGPGKey("")
	if err == nil {
		t.Error("expected error for empty key ID")
	}
}

func TestGitHubClient_AddGPGKey_TooShort(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{}}
	err := client.AddGPGKey("ABCDEF0")
	if err == nil {
		t.Error("expected error for too-short key ID")
	}
}

func TestGitHubClient_ListSSHKeys_Parse(t *testing.T) {
	jsonData := `[{"id": 123, "key": "ssh-ed25519 AAAA test@example.com", "title": "my-key"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	keys, err := client.ListSSHKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Title != "my-key" {
		t.Errorf("title = %q, want %q", keys[0].Title, "my-key")
	}
	if keys[0].Key != "ssh-ed25519 AAAA test@example.com" {
		t.Errorf("key = %q, want full key string", keys[0].Key)
	}
}

func TestGitHubClient_ListSSHKeys_Empty(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{output: []byte("[]")}}
	keys, err := client.ListSSHKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestGitHubClient_ListSSHKeys_CommandError(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{output: []byte("error"), err: fmt.Errorf("gh failed")}}
	_, err := client.ListSSHKeys()
	if err == nil {
		t.Error("expected error for command failure")
	}
}

func TestGitHubClient_ListSSHKeys_BadJSON(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{output: []byte("not json")}}
	_, err := client.ListSSHKeys()
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestGitHubClient_IsKeyRegistered_Match(t *testing.T) {
	// Create a pub key file
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA_MATCH test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock gh returning a matching key
	jsonData := `[{"id": 1, "key": "ssh-ed25519 AAAA_MATCH test@example.com", "title": "matched"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	registered, err := client.IsKeyRegistered(pubPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registered {
		t.Error("expected key to be registered")
	}
}

func TestGitHubClient_IsKeyRegistered_NoMatch(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA_MATCH test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Mock gh returning a non-matching key
	jsonNoMatch := `[{"id": 1, "key": "ssh-ed25519 BBBB_NOMATCH other@example.com", "title": "other"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonNoMatch)}}

	registered, err := client.IsKeyRegistered(pubPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registered {
		t.Error("expected key to NOT be registered")
	}
}

func TestGitHubClient_IsKeyRegistered_EmptyGitHubKeys(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	if err := os.WriteFile(pubPath, []byte("ssh-ed25519 AAAA_MATCH test@example.com\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &GitHubClient{Commander: &mockCommander{output: []byte("[]")}}

	registered, err := client.IsKeyRegistered(pubPath, tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registered {
		t.Error("expected key to NOT be registered when no GitHub keys exist")
	}
}

func TestGitHubClient_IsKeyRegistered_BadPubKey(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "test.pub")
	// Write a malformed public key with only one field
	if err := os.WriteFile(pubPath, []byte("justoneword\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &GitHubClient{Commander: &mockCommander{output: []byte("[]")}}

	_, err := client.IsKeyRegistered(pubPath, tmpDir)
	if err == nil {
		t.Error("expected error for invalid public key format")
	}
}

func TestGitHubClient_IsAuthenticated_Success(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{output: []byte("logged in")}}
	auth, err := client.IsAuthenticated()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !auth {
		t.Error("expected authenticated to be true")
	}
}

func TestGitHubClient_IsAuthenticated_NotAuth(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{err: fmt.Errorf("not logged in")}}
	auth, err := client.IsAuthenticated()
	if err != nil {
		t.Fatalf("unexpected error: %v (should return false, not error)", err)
	}
	if auth {
		t.Error("expected authenticated to be false")
	}
}

func TestGitHubClient_IsAuthenticated_Smoke(t *testing.T) {
	if !HasGHCLI() {
		t.Skip("gh CLI not available")
	}
	client := NewGitHubClient()
	auth, err := client.IsAuthenticated()
	if err != nil {
		t.Logf("IsAuthenticated error: %v", err)
	}
	t.Logf("IsAuthenticated: %v", auth)
}

func TestNewGitHubClient(t *testing.T) {
	client := NewGitHubClient()
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.Commander == nil {
		t.Error("expected non-nil commander")
	}
}

func TestGitHubClient_getCommander_NilFallback(t *testing.T) {
	client := &GitHubClient{Commander: nil}
	cmd := client.getCommander()
	if cmd == nil {
		t.Error("expected non-nil commander from fallback")
	}
}

func TestGitHubClient_ListGPGKeys_Parse(t *testing.T) {
	jsonData := `[{"id": 1, "key_id": "ABCDEF1234567890", "email": "test@example.com"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	keys, err := client.ListGPGKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].KeyID != "ABCDEF1234567890" {
		t.Errorf("key_id = %q, want %q", keys[0].KeyID, "ABCDEF1234567890")
	}
}

func TestGitHubClient_ListGPGKeys_Empty(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{output: []byte("[]")}}
	keys, err := client.ListGPGKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestGitHubClient_ListGPGKeys_CommandError(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{err: fmt.Errorf("gh failed")}}
	_, err := client.ListGPGKeys()
	if err == nil {
		t.Error("expected error for command failure")
	}
}

func TestGitHubClient_IsGPGKeyRegistered_Match(t *testing.T) {
	jsonData := `[{"id": 1, "key_id": "ABCDEF1234567890", "email": "test@example.com"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	registered, err := client.IsGPGKeyRegistered("ABCDEF1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registered {
		t.Error("expected key to be registered")
	}
}

func TestGitHubClient_IsGPGKeyRegistered_CaseInsensitive(t *testing.T) {
	jsonData := `[{"id": 1, "key_id": "ABCDEF1234567890", "email": "test@example.com"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	registered, err := client.IsGPGKeyRegistered("abcdef1234567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !registered {
		t.Error("expected case-insensitive match")
	}
}

func TestGitHubClient_IsGPGKeyRegistered_ShortIDNoMatch(t *testing.T) {
	// Short form key IDs should NOT match long form (risk of Evil32 false positives)
	jsonData := `[{"id": 1, "key_id": "ABCDEF1234567890", "email": "test@example.com"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	registered, err := client.IsGPGKeyRegistered("34567890")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registered {
		t.Error("short key IDs should not match via suffix (Evil32 risk)")
	}
}

func TestGitHubClient_IsGPGKeyRegistered_NoMatch(t *testing.T) {
	jsonData := `[{"id": 1, "key_id": "ABCDEF1234567890", "email": "test@example.com"}]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	registered, err := client.IsGPGKeyRegistered("11111111")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registered {
		t.Error("expected key to NOT be registered")
	}
}

func TestGitHubClient_IsGPGKeyRegistered_BadKeyID(t *testing.T) {
	client := &GitHubClient{Commander: &mockCommander{output: []byte("[]")}}
	_, err := client.IsGPGKeyRegistered("not-hex!")
	if err == nil {
		t.Error("expected error for invalid key ID")
	}
}

func TestGitHubClient_ListSSHKeys_MultipleKeys(t *testing.T) {
	jsonData := `[
		{"id": 1, "key": "ssh-ed25519 AAAA first@example.com", "title": "key-1"},
		{"id": 2, "key": "ssh-rsa BBBB second@example.com", "title": "key-2"},
		{"id": 3, "key": "ssh-ed25519 CCCC third@example.com", "title": "key-3"}
	]`
	client := &GitHubClient{Commander: &mockCommander{output: []byte(jsonData)}}

	keys, err := client.ListSSHKeys()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[1].Title != "key-2" {
		t.Errorf("second key title = %q, want %q", keys[1].Title, "key-2")
	}
}
