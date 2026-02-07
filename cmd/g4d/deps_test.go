package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

func TestRunDepsInstall_ManualOnly(t *testing.T) {
	cfg := &config.Config{
		Dependencies: config.Dependencies{
			Core: []config.DependencyItem{
				{Name: "manual-tool", Binary: "definitely-not-installed-xyz", Manual: true},
			},
		},
	}
	p := &platform.Platform{}

	var stdout bytes.Buffer
	err := runDepsInstall(cfg, p, &stdout)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Manual (install required): 1 packages") {
		t.Fatalf("expected manual install summary, got %q", output)
	}
	if !strings.Contains(output, "manual-tool") {
		t.Fatalf("expected manual dep name in output, got %q", output)
	}
	if !strings.Contains(output, "All auto-installable dependencies are already installed.") {
		t.Fatalf("expected auto-installable summary, got %q", output)
	}
}

func TestRunDepsInstall_NoDeps(t *testing.T) {
	cfg := &config.Config{
		Dependencies: config.Dependencies{},
	}
	p := &platform.Platform{}

	var stdout bytes.Buffer
	err := runDepsInstall(cfg, p, &stdout)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "All dependencies are already installed!") {
		t.Fatalf("expected all installed message, got %q", output)
	}
}
