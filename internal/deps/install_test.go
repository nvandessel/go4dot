package deps

import (
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

func TestInstall_ManualOnly(t *testing.T) {
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Core: []config.DependencyItem{
				{Name: "manual-tool", Binary: "definitely-not-installed-xyz", Manual: true},
			},
		},
	}

	var messages []string
	opts := InstallOptions{
		ProgressFunc: func(current, total int, msg string) {
			messages = append(messages, msg)
		},
	}

	result, err := Install(cfg, &platform.Platform{}, opts)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(result.ManualSkipped) != 1 {
		t.Fatalf("expected 1 manual skipped dep, got %d", len(result.ManualSkipped))
	}
	if result.ManualSkipped[0].Name != "manual-tool" {
		t.Fatalf("expected manual dep name %q, got %q", "manual-tool", result.ManualSkipped[0].Name)
	}
	if len(result.Installed) != 0 {
		t.Fatalf("expected no installed deps, got %d", len(result.Installed))
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected no failed deps, got %d", len(result.Failed))
	}

	found := false
	for _, msg := range messages {
		if strings.Contains(msg, "Skipping manual dependency") {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected progress message for manual dependency skip")
	}
}
