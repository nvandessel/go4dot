package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// Test with the example minimal config
	examplePath := "../../examples/minimal/.go4dot.yaml"

	cfg, err := Load(examplePath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify basic structure
	if cfg.SchemaVersion != "1.0" {
		t.Errorf("SchemaVersion = %s, want 1.0", cfg.SchemaVersion)
	}

	if cfg.Metadata.Name != "minimal-dotfiles" {
		t.Errorf("Metadata.Name = %s, want minimal-dotfiles", cfg.Metadata.Name)
	}

	// Verify dependencies
	if len(cfg.Dependencies.Critical) != 2 {
		t.Errorf("len(Critical) = %d, want 2", len(cfg.Dependencies.Critical))
	}

	if len(cfg.Dependencies.Core) != 1 {
		t.Errorf("len(Core) = %d, want 1", len(cfg.Dependencies.Core))
	}

	// Verify configs
	if len(cfg.Configs.Core) != 2 {
		t.Errorf("len(Configs.Core) = %d, want 2", len(cfg.Configs.Core))
	}

	if len(cfg.Configs.Optional) != 1 {
		t.Errorf("len(Configs.Optional) = %d, want 1", len(cfg.Configs.Optional))
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/path/that/does/not/exist/.go4dot.yaml")
	if err == nil {
		t.Error("Load() should fail for non-existent file")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	// Create a temp file with invalid YAML
	tmpfile, err := os.CreateTemp("", "invalid-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_, err = tmpfile.Write([]byte("invalid: yaml: content:\n  - this is\n wrong"))
	if err != nil {
		t.Fatal(err)
	}
	_ = tmpfile.Close()

	_, err = Load(tmpfile.Name())
	if err == nil {
		t.Error("Load() should fail for invalid YAML")
	}
}

func TestLoadFromPath(t *testing.T) {
	// Test loading from directory
	exampleDir := "../../examples/minimal"
	cfg, err := LoadFromPath(exampleDir)
	if err != nil {
		t.Fatalf("LoadFromPath() with directory failed: %v", err)
	}

	if cfg.Metadata.Name != "minimal-dotfiles" {
		t.Errorf("Metadata.Name = %s, want minimal-dotfiles", cfg.Metadata.Name)
	}

	// Test loading from file
	exampleFile := filepath.Join(exampleDir, ".go4dot.yaml")
	cfg, err = LoadFromPath(exampleFile)
	if err != nil {
		t.Fatalf("LoadFromPath() with file failed: %v", err)
	}

	if cfg.Metadata.Name != "minimal-dotfiles" {
		t.Errorf("Metadata.Name = %s, want minimal-dotfiles", cfg.Metadata.Name)
	}
}

func TestDependencyItemUnmarshal(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		wantName string
	}{
		{
			name:     "Simple string",
			yaml:     "dependencies:\n  critical:\n    - git\n",
			wantName: "git",
		},
		{
			name: "Complex object",
			yaml: `dependencies:
  core:
    - name: neovim
      binary: nvim
      package:
        dnf: neovim
        apt: neovim
`,
			wantName: "neovim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpfile, err := os.CreateTemp("", "test-*.yaml")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = os.Remove(tmpfile.Name()) }()

			// Write minimal valid config
			content := "schema_version: \"1.0\"\nmetadata:\n  name: test\n" + tt.yaml
			_, err = tmpfile.Write([]byte(content))
			if err != nil {
				t.Fatal(err)
			}
			_ = tmpfile.Close()

			cfg, err := Load(tmpfile.Name())
			if err != nil {
				t.Fatalf("Load() failed: %v", err)
			}

			// Check if dependency was parsed
			var found bool
			for _, dep := range cfg.GetAllDependencies() {
				if dep.Name == tt.wantName {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("Dependency %s not found", tt.wantName)
			}
		})
	}
}
