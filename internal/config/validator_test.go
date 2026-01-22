package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	// Create a dummy directory for path validation
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Fatalf("Failed to remove temp dir: %v", err)
		}
	}()

	// Create a dummy executable file for hook validation
	dummyHook := filepath.Join(tempDir, "hook.sh")
	if err := os.WriteFile(dummyHook, []byte("#!/bin/sh\necho 'hello'"), 0755); err != nil {
		t.Fatalf("Failed to create dummy hook: %v", err)
	}

	// Create a dummy non-executable file
	nonExecutableHook := filepath.Join(tempDir, "non-executable-hook.sh")
	if err := os.WriteFile(nonExecutableHook, []byte("#!/bin/sh\necho 'hello'"), 0644); err != nil {
		t.Fatalf("Failed to create non-executable hook: %v", err)
	}

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Configs: ConfigGroups{
					Core: []ConfigItem{
						{Name: "git", Path: tempDir},
					},
				},
				PostInstall: dummyHook,
			},
			wantErr: false,
		},
		{
			name: "Missing schema_version",
			config: &Config{
				Metadata: Metadata{Name: "test"},
			},
			wantErr: true,
		},
		{
			name: "Missing metadata.name",
			config: &Config{
				SchemaVersion: "1.0",
			},
			wantErr: true,
		},
		{
			name: "Config path does not exist",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Configs: ConfigGroups{
					Core: []ConfigItem{
						{Name: "git", Path: "/path/that/does/not/exist"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Post-install script does not exist",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				PostInstall:   "/path/that/does/not/exist",
			},
			wantErr: true,
		},
		{
			name: "Post-install script not executable",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				PostInstall:   nonExecutableHook,
			},
			wantErr: true,
		},
		{
			name: "Circular dependency",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Configs: ConfigGroups{
					Core: []ConfigItem{
						{Name: "a", Path: tempDir, DependsOn: []string{"b"}},
						{Name: "b", Path: tempDir, DependsOn: []string{"a"}},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "No circular dependency",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Configs: ConfigGroups{
					Core: []ConfigItem{
						{Name: "a", Path: tempDir, DependsOn: []string{"b"}},
						{Name: "b", Path: tempDir, DependsOn: []string{"c"}},
						{Name: "c", Path: tempDir},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(tempDir); (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
