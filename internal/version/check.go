package version

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CheckResult holds version check info
type CheckResult struct {
	LatestVersion  string
	CurrentVersion string
	IsOutdated     bool
	ReleaseURL     string
}

// githubAPIURL is the endpoint for checking releases.
// Protected by githubAPIURLMu for safe concurrent test execution.
var (
	githubAPIURLMu sync.RWMutex
	githubAPIURL   = "https://api.github.com/repos/nvandessel/go4dot/releases/latest"
)

// CheckForUpdates queries GitHub for the latest release.
// The context allows the caller to cancel the request if needed.
func CheckForUpdates(ctx context.Context, currentVersion string) (*CheckResult, error) {
	if currentVersion == "dev" || currentVersion == "unknown" {
		return nil, nil // Don't check for dev builds
	}

	// Create request with context for cancellation support
	githubAPIURLMu.RLock()
	url := githubAPIURL
	githubAPIURLMu.RUnlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	latest := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(currentVersion, "v")

	if latest != current {
		return &CheckResult{
			LatestVersion:  latest,
			CurrentVersion: current,
			IsOutdated:     true,
			ReleaseURL:     release.HTMLURL,
		}, nil
	}

	return nil, nil
}
