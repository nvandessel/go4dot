package machine

import (
	"context"
	"testing"
	"time"
)

func TestVerifyGitConfig_Smoke(t *testing.T) {
	// Smoke test - just verify no panic
	results := VerifyGitConfig()
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	for _, r := range results {
		t.Logf("  %s: %s — %s", r.Name, r.Status, r.Message)
	}
}

func TestVerifyGPGSign_InvalidKeyID(t *testing.T) {
	result := VerifyGPGSign("not-hex!")
	if result.Status != VerifyFail {
		t.Errorf("expected fail for invalid key ID, got %q", result.Status)
	}
	if result.Name != "gpg-sign" {
		t.Errorf("expected name gpg-sign, got %q", result.Name)
	}
}

func TestVerifyGPGSign_EmptyKeyID(t *testing.T) {
	result := VerifyGPGSign("")
	if result.Status != VerifyFail {
		t.Errorf("expected fail for empty key ID, got %q", result.Status)
	}
}

func TestVerifySSHGitHub_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := VerifySSHGitHub(ctx)
	// Should be skip or fail (not pass since context is cancelled)
	if result.Status == VerifyPass {
		t.Errorf("expected skip or fail for cancelled context, got pass")
	}
	if result.Name != "ssh-github" {
		t.Errorf("expected name ssh-github, got %q", result.Name)
	}
	t.Logf("Result: %s — %s", result.Status, result.Message)
}

func TestVerifySSHGitHub_Timeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()
	time.Sleep(5 * time.Millisecond) // Let timeout expire

	result := VerifySSHGitHub(ctx)
	if result.Status == VerifyPass {
		t.Logf("Surprisingly passed (SSH might be configured) — %s", result.Message)
	} else {
		t.Logf("Result: %s — %s", result.Status, result.Message)
	}
}

func TestRunAllVerifications(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	results := RunAllVerifications(ctx, "")
	// Should have at least git config checks (3) + ssh check (1) = 4
	if len(results) < 4 {
		t.Fatalf("expected at least 4 results, got %d", len(results))
	}
	for _, r := range results {
		t.Logf("  %s: %s — %s", r.Name, r.Status, r.Message)
	}
}

func TestRunAllVerifications_WithGPG(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Use a fake key ID - won't actually work but should add a gpg-sign result
	results := RunAllVerifications(ctx, "ABCD1234")
	hasGPG := false
	for _, r := range results {
		if r.Name == "gpg-sign" {
			hasGPG = true
		}
	}
	if !hasGPG {
		t.Error("expected gpg-sign result when key ID provided")
	}
}
