package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// CheckResult holds version check info
type CheckResult struct {
	LatestVersion  string
	CurrentVersion string
	IsOutdated     bool
	ReleaseURL     string
}

// CheckForUpdates queries GitHub for the latest release
func CheckForUpdates(currentVersion string) (*CheckResult, error) {
	if currentVersion == "dev" || currentVersion == "unknown" {
		return nil, nil // Don't check for dev builds
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/nvandessel/go4dot/releases/latest")
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
