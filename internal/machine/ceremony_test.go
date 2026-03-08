package machine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreflightChecks(t *testing.T) {
	t.Run("with nil client", func(t *testing.T) {
		results := PreflightChecks(nil)
		// Should have gpg check and gh check
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
		// gh check should fail with nil client
		ghResult := results[1]
		if ghResult.Name != "gh-authenticated" {
			t.Errorf("expected gh-authenticated check, got %q", ghResult.Name)
		}
		if ghResult.Success {
			t.Error("expected gh check to fail with nil client")
		}
	})

	t.Run("with mock client authenticated", func(t *testing.T) {
		client := &GitHubClient{Commander: &mockCommander{output: []byte("logged in")}}
		results := PreflightChecks(client)
		if len(results) < 2 {
			t.Fatalf("expected at least 2 results, got %d", len(results))
		}
	})
}

func TestImportMasterKey(t *testing.T) {
	tests := []struct {
		name      string
		keyPath   string
		setup     func(t *testing.T) string // returns path
		runErr    error
		wantErr   bool
		errSubstr string
	}{
		{
			name: "empty path",
			setup: func(t *testing.T) string {
				return ""
			},
			wantErr:   true,
			errSubstr: "must not be empty",
		},
		{
			name: "nonexistent file",
			setup: func(t *testing.T) string {
				return "/tmp/nonexistent-gpg-key-12345"
			},
			wantErr:   true,
			errSubstr: "cannot access",
		},
		{
			name: "directory instead of file",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr:   true,
			errSubstr: "not a regular file",
		},
		{
			name: "valid file, gpg succeeds",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "master.key")
				if err := os.WriteFile(path, []byte("fake key data"), 0600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr: false,
		},
		{
			name: "valid file, gpg fails",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "master.key")
				if err := os.WriteFile(path, []byte("fake key data"), 0600); err != nil {
					t.Fatal(err)
				}
				return path
			},
			runErr:    fmt.Errorf("gpg: no valid data"),
			wantErr:   true,
			errSubstr: "gpg --import failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := ""
			if tt.setup != nil {
				path = tt.setup(t)
			}
			if tt.keyPath != "" {
				path = tt.keyPath
			}

			mock := &mockGPGRunner{
				runOutput: []byte("ok"),
				runErr:    tt.runErr,
			}

			err := ImportMasterKey(mock, path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImportMasterKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errSubstr != "" {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}

func TestConfigureGitSigning(t *testing.T) {
	t.Run("invalid key ID", func(t *testing.T) {
		err := ConfigureGitSigning("", "user@example.com")
		if err == nil {
			t.Error("expected error for empty key ID")
		}
	})

	t.Run("invalid email", func(t *testing.T) {
		err := ConfigureGitSigning("ABCDEF1234567890", "")
		if err == nil {
			t.Error("expected error for empty email")
		}
	})

	t.Run("invalid email format", func(t *testing.T) {
		err := ConfigureGitSigning("ABCDEF1234567890", "not-an-email")
		if err == nil {
			t.Error("expected error for invalid email format")
		}
	})
}

func TestCeremonyResult_StepTracking(t *testing.T) {
	result := &CeremonyResult{}

	result.Steps = append(result.Steps, StepResult{
		Name: "step-1", Success: true, Message: "done",
	})
	result.Steps = append(result.Steps, StepResult{
		Name: "step-2", Success: false, Message: "failed",
	})

	if len(result.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(result.Steps))
	}
	if result.Steps[0].Name != "step-1" || !result.Steps[0].Success {
		t.Errorf("step 1 unexpected: %+v", result.Steps[0])
	}
	if result.Steps[1].Name != "step-2" || result.Steps[1].Success {
		t.Errorf("step 2 unexpected: %+v", result.Steps[1])
	}
}
