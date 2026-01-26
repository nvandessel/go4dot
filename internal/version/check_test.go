package version

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckForUpdates_DevVersion(t *testing.T) {
	ctx := context.Background()

	result, err := CheckForUpdates(ctx, "dev")
	if err != nil {
		t.Errorf("expected no error for dev version, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for dev version, got %v", result)
	}
}

func TestCheckForUpdates_UnknownVersion(t *testing.T) {
	ctx := context.Background()

	result, err := CheckForUpdates(ctx, "unknown")
	if err != nil {
		t.Errorf("expected no error for unknown version, got %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for unknown version, got %v", result)
	}
}

func TestCheckForUpdates_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Override the URL for testing
	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := CheckForUpdates(ctx, "1.0.0")
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
}

func TestCheckForUpdates_ContextTimeout(t *testing.T) {
	// Create a server that delays response longer than timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Override the URL for testing
	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := CheckForUpdates(ctx, "1.0.0")
	if err == nil {
		t.Error("expected error for context timeout, got nil")
	}
}

func TestCheckForUpdates_OutdatedVersion(t *testing.T) {
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
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ctx := context.Background()
	result, err := CheckForUpdates(ctx, "1.0.0")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if !result.IsOutdated {
		t.Error("expected IsOutdated to be true")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("expected LatestVersion '2.0.0', got '%s'", result.LatestVersion)
	}
	if result.CurrentVersion != "1.0.0" {
		t.Errorf("expected CurrentVersion '1.0.0', got '%s'", result.CurrentVersion)
	}
}

func TestCheckForUpdates_CurrentVersion(t *testing.T) {
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
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ctx := context.Background()
	result, err := CheckForUpdates(ctx, "v1.0.0")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for current version, got %v", result)
	}
}

func TestCheckForUpdates_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ctx := context.Background()
	_, err := CheckForUpdates(ctx, "1.0.0")

	if err == nil {
		t.Error("expected error for HTTP 500, got nil")
	}
}

func TestCheckForUpdates_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ctx := context.Background()
	_, err := CheckForUpdates(ctx, "1.0.0")

	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestCheckForUpdates_VersionWithoutPrefix(t *testing.T) {
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
	defer server.Close()

	originalURL := githubAPIURL
	githubAPIURL = server.URL
	defer func() { githubAPIURL = originalURL }()

	ctx := context.Background()
	result, err := CheckForUpdates(ctx, "1.0.0")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected result, got nil")
	}
	if result.LatestVersion != "2.0.0" {
		t.Errorf("expected LatestVersion '2.0.0', got '%s'", result.LatestVersion)
	}
}
