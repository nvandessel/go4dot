package version

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// setTestURL safely sets the GitHub API URL for testing and returns a cleanup function.
func setTestURL(t *testing.T, url string) {
	t.Helper()
	githubAPIURLMu.Lock()
	original := githubAPIURL
	githubAPIURL = url
	githubAPIURLMu.Unlock()

	t.Cleanup(func() {
		githubAPIURLMu.Lock()
		githubAPIURL = original
		githubAPIURLMu.Unlock()
	})
}

func TestCheckForUpdates(t *testing.T) {
	tests := []struct {
		name           string
		version        string
		setupServer    func(t *testing.T) (url string, cleanup func())
		ctxBuilder     func() (context.Context, context.CancelFunc)
		wantResult     *CheckResult
		wantErr        bool
		skipServerURL  bool // true for dev/unknown version tests that don't need a server
	}{
		{
			name:          "dev version skips check",
			version:       "dev",
			skipServerURL: true,
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: nil,
			wantErr:    false,
		},
		{
			name:          "unknown version skips check",
			version:       "unknown",
			skipServerURL: true,
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: nil,
			wantErr:    false,
		},
		{
			name:    "context already cancelled",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(100 * time.Millisecond)
					w.WriteHeader(http.StatusOK)
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, func() {}
			},
			wantResult: nil,
			wantErr:    true,
		},
		{
			name:    "context timeout",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					time.Sleep(200 * time.Millisecond)
					w.WriteHeader(http.StatusOK)
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), 50*time.Millisecond)
			},
			wantResult: nil,
			wantErr:    true,
		},
		{
			name:    "outdated version detected",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := struct {
						TagName string `json:"tag_name"`
						HTMLURL string `json:"html_url"`
					}{
						TagName: "v2.0.0",
						HTMLURL: "https://github.com/nvandessel/go4dot/releases/tag/v2.0.0",
					}
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("failed to encode response: %v", err)
					}
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: &CheckResult{
				LatestVersion:  "2.0.0",
				CurrentVersion: "1.0.0",
				IsOutdated:     true,
				ReleaseURL:     "https://github.com/nvandessel/go4dot/releases/tag/v2.0.0",
			},
			wantErr: false,
		},
		{
			name:    "current version returns nil",
			version: "v1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := struct {
						TagName string `json:"tag_name"`
						HTMLURL string `json:"html_url"`
					}{
						TagName: "v1.0.0",
						HTMLURL: "https://github.com/nvandessel/go4dot/releases/tag/v1.0.0",
					}
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("failed to encode response: %v", err)
					}
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: nil,
			wantErr:    false,
		},
		{
			name:    "HTTP 500 error",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: nil,
			wantErr:    true,
		},
		{
			name:    "invalid JSON response",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte("invalid json"))
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: nil,
			wantErr:    true,
		},
		{
			name:    "version without v prefix",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					response := struct {
						TagName string `json:"tag_name"`
						HTMLURL string `json:"html_url"`
					}{
						TagName: "2.0.0", // No 'v' prefix
						HTMLURL: "https://github.com/nvandessel/go4dot/releases/tag/2.0.0",
					}
					w.Header().Set("Content-Type", "application/json")
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Errorf("failed to encode response: %v", err)
					}
				}))
				return server.URL, server.Close
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: &CheckResult{
				LatestVersion:  "2.0.0",
				CurrentVersion: "1.0.0",
				IsOutdated:     true,
				ReleaseURL:     "https://github.com/nvandessel/go4dot/releases/tag/2.0.0",
			},
			wantErr: false,
		},
		{
			name:    "server unreachable",
			version: "1.0.0",
			setupServer: func(t *testing.T) (string, func()) {
				// Return a URL that will refuse connection
				return "http://127.0.0.1:1", func() {}
			},
			ctxBuilder: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantResult: nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up server if needed
			if !tt.skipServerURL && tt.setupServer != nil {
				serverURL, cleanup := tt.setupServer(t)
				defer cleanup()
				setTestURL(t, serverURL)
			}

			// Build context
			ctx, cancel := tt.ctxBuilder()
			defer cancel()

			// Call function under test
			result, err := CheckForUpdates(ctx, tt.version)

			// Check error expectation
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check result expectation
			if tt.wantResult == nil {
				if result != nil {
					t.Errorf("expected nil result, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected result, got nil")
			}
			if result.IsOutdated != tt.wantResult.IsOutdated {
				t.Errorf("IsOutdated = %v, want %v", result.IsOutdated, tt.wantResult.IsOutdated)
			}
			if result.LatestVersion != tt.wantResult.LatestVersion {
				t.Errorf("LatestVersion = %q, want %q", result.LatestVersion, tt.wantResult.LatestVersion)
			}
			if result.CurrentVersion != tt.wantResult.CurrentVersion {
				t.Errorf("CurrentVersion = %q, want %q", result.CurrentVersion, tt.wantResult.CurrentVersion)
			}
			if result.ReleaseURL != tt.wantResult.ReleaseURL {
				t.Errorf("ReleaseURL = %q, want %q", result.ReleaseURL, tt.wantResult.ReleaseURL)
			}
		})
	}
}
