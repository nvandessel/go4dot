package dashboard

import (
	"errors"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/stow"
)

func TestSyncResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   *SyncResult
		expected bool
	}{
		{
			name:     "No errors",
			result:   &SyncResult{Success: []string{"vim", "zsh"}},
			expected: false,
		},
		{
			name: "With errors",
			result: &SyncResult{
				Failed: []stow.StowError{{ConfigName: "vim", Error: errors.New("test")}},
			},
			expected: true,
		},
		{
			name:     "Empty result",
			result:   &SyncResult{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.expected {
				t.Errorf("HasErrors() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestSyncResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   *SyncResult
		expected string
	}{
		{
			name:     "No configs",
			result:   &SyncResult{},
			expected: "No configs to sync",
		},
		{
			name:     "Success only",
			result:   &SyncResult{Success: []string{"vim", "zsh"}},
			expected: "2 synced",
		},
		{
			name: "With failed",
			result: &SyncResult{
				Success: []string{"vim"},
				Failed:  []stow.StowError{{ConfigName: "zsh", Error: errors.New("test")}},
			},
			expected: "1 synced, 1 failed",
		},
		{
			name: "With skipped",
			result: &SyncResult{
				Success: []string{"vim"},
				Skipped: []string{"zsh"},
			},
			expected: "1 synced, 1 skipped",
		},
		{
			name: "All types",
			result: &SyncResult{
				Success: []string{"vim", "tmux"},
				Failed:  []stow.StowError{{ConfigName: "zsh", Error: errors.New("test")}},
				Skipped: []string{"git"},
			},
			expected: "2 synced, 1 failed, 1 skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.Summary()
			if got != tt.expected {
				t.Errorf("Summary() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestUpdateResult_Summary(t *testing.T) {
	tests := []struct {
		name     string
		result   *UpdateResult
		expected string
	}{
		{
			name:     "No updates",
			result:   &UpdateResult{},
			expected: "No updates needed",
		},
		{
			name:     "Updates only",
			result:   &UpdateResult{Updated: []string{"repo1", "repo2"}},
			expected: "2 updated",
		},
		{
			name: "With failed",
			result: &UpdateResult{
				Updated: []string{"repo1"},
				Failed:  []string{"repo2"},
			},
			expected: "1 updated, 1 failed",
		},
		{
			name: "With skipped",
			result: &UpdateResult{
				Updated: []string{"repo1"},
				Skipped: []string{"repo2"},
			},
			expected: "1 updated, 1 skipped",
		},
		{
			name: "All types",
			result: &UpdateResult{
				Updated: []string{"repo1", "repo2"},
				Failed:  []string{"repo3"},
				Skipped: []string{"repo4"},
			},
			expected: "2 updated, 1 failed, 1 skipped",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.Summary()
			if got != tt.expected {
				t.Errorf("Summary() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

func TestCollectSyncErrors(t *testing.T) {
	permDeniedErr := errors.New("permission denied")
	fileExistsErr := errors.New("file exists")

	tests := []struct {
		name          string
		failed        []stow.StowError
		wantNil       bool
		wantSubstr    string
		wantWrappedIs error // Verify errors.Is works with wrapped error
	}{
		{
			name:    "Empty failed list",
			failed:  nil,
			wantNil: true,
		},
		{
			name: "Single error",
			failed: []stow.StowError{
				{ConfigName: "vim", Error: permDeniedErr},
			},
			wantSubstr:    "sync failed for vim",
			wantWrappedIs: permDeniedErr,
		},
		{
			name: "Multiple errors",
			failed: []stow.StowError{
				{ConfigName: "vim", Error: permDeniedErr},
				{ConfigName: "zsh", Error: fileExistsErr},
			},
			wantSubstr:    "sync failed for 2 configs",
			wantWrappedIs: permDeniedErr, // First error is wrapped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collectSyncErrors(tt.failed)
			if tt.wantNil {
				if err != nil {
					t.Errorf("collectSyncErrors() = %v, expected nil", err)
				}
				return
			}
			if err == nil {
				t.Error("collectSyncErrors() = nil, expected error")
				return
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error = %q, expected to contain %q", err.Error(), tt.wantSubstr)
			}
			if tt.wantWrappedIs != nil && !errors.Is(err, tt.wantWrappedIs) {
				t.Errorf("errors.Is() failed: wrapped error not accessible")
			}
		})
	}
}

func TestCollectUpdateErrors(t *testing.T) {
	cloneFailedErr := errors.New("clone failed")
	networkErr := errors.New("network error")

	tests := []struct {
		name          string
		failed        []deps.ExternalError
		wantNil       bool
		wantSubstr    string
		wantWrappedIs error // Verify errors.Is works with wrapped error
	}{
		{
			name:    "Empty failed list",
			failed:  nil,
			wantNil: true,
		},
		{
			name: "Single error",
			failed: []deps.ExternalError{
				{Dep: config.ExternalDep{Name: "repo1"}, Error: cloneFailedErr},
			},
			wantSubstr:    "update failed for repo1",
			wantWrappedIs: cloneFailedErr,
		},
		{
			name: "Multiple errors",
			failed: []deps.ExternalError{
				{Dep: config.ExternalDep{Name: "repo1"}, Error: cloneFailedErr},
				{Dep: config.ExternalDep{Name: "repo2"}, Error: networkErr},
			},
			wantSubstr:    "update failed for 2 dependencies",
			wantWrappedIs: cloneFailedErr, // First error is wrapped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := collectUpdateErrors(tt.failed)
			if tt.wantNil {
				if err != nil {
					t.Errorf("collectUpdateErrors() = %v, expected nil", err)
				}
				return
			}
			if err == nil {
				t.Error("collectUpdateErrors() = nil, expected error")
				return
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Errorf("error = %q, expected to contain %q", err.Error(), tt.wantSubstr)
			}
			if tt.wantWrappedIs != nil && !errors.Is(err, tt.wantWrappedIs) {
				t.Errorf("errors.Is() failed: wrapped error not accessible")
			}
		})
	}
}
