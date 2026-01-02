package platform

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect() failed: %v", err)
	}

	if p == nil {
		t.Fatal("Detect() returned nil platform")
	}

	// Basic sanity checks
	if p.OS == "" {
		t.Error("OS should not be empty")
	}

	if p.Architecture == "" {
		t.Error("Architecture should not be empty")
	}

	// Verify OS matches runtime
	if p.OS != runtime.GOOS {
		t.Errorf("OS mismatch: got %s, want %s", p.OS, runtime.GOOS)
	}

	// Verify Architecture matches runtime
	if p.Architecture != runtime.GOARCH {
		t.Errorf("Architecture mismatch: got %s, want %s", p.Architecture, runtime.GOARCH)
	}
}

func TestPlatformString(t *testing.T) {
	p := &Platform{
		OS:             "linux",
		Distro:         "fedora",
		DistroVersion:  "43",
		IsWSL:          false,
		PackageManager: "dnf",
		Architecture:   "amd64",
	}

	s := p.String()

	// Check that output contains key information
	expectedStrings := []string{"linux", "fedora", "43", "dnf", "amd64"}
	for _, expected := range expectedStrings {
		if !strings.Contains(s, expected) {
			t.Errorf("String() output missing '%s': %s", expected, s)
		}
	}
}

func TestPlatformStringWSL(t *testing.T) {
	p := &Platform{
		OS:             "linux",
		Distro:         "ubuntu",
		DistroVersion:  "22.04",
		IsWSL:          true,
		PackageManager: "apt",
		Architecture:   "amd64",
	}

	s := p.String()

	if !strings.Contains(s, "WSL") {
		t.Errorf("String() should contain 'WSL' for WSL platform: %s", s)
	}
}

func TestPlatformStringMacOS(t *testing.T) {
	p := &Platform{
		OS:             "darwin",
		PackageManager: "brew",
		Architecture:   "arm64",
	}

	s := p.String()

	expectedStrings := []string{"darwin", "brew", "arm64"}
	for _, expected := range expectedStrings {
		if !strings.Contains(s, expected) {
			t.Errorf("String() output missing '%s': %s", expected, s)
		}
	}

	// Should not contain distro info for macOS
	if strings.Contains(s, "Distro:") {
		t.Errorf("String() should not contain distro info for macOS: %s", s)
	}
}

func TestIsLinux(t *testing.T) {
	tests := []struct {
		name string
		os   string
		want bool
	}{
		{"Linux", "linux", true},
		{"macOS", "darwin", false},
		{"Windows", "windows", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{OS: tt.os}
			if got := p.IsLinux(); got != tt.want {
				t.Errorf("IsLinux() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMacOS(t *testing.T) {
	tests := []struct {
		name string
		os   string
		want bool
	}{
		{"Linux", "linux", false},
		{"macOS", "darwin", true},
		{"Windows", "windows", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{OS: tt.os}
			if got := p.IsMacOS(); got != tt.want {
				t.Errorf("IsMacOS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsWindows(t *testing.T) {
	tests := []struct {
		name string
		os   string
		want bool
	}{
		{"Linux", "linux", false},
		{"macOS", "darwin", false},
		{"Windows", "windows", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{OS: tt.os}
			if got := p.IsWindows(); got != tt.want {
				t.Errorf("IsWindows() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSupportsPackageManager(t *testing.T) {
	tests := []struct {
		name   string
		pkgMgr string
		want   bool
	}{
		{"DNF", "dnf", true},
		{"APT", "apt", true},
		{"Brew", "brew", true},
		{"Unknown", "unknown", false},
		{"None", "none", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Platform{PackageManager: tt.pkgMgr}
			if got := p.SupportsPackageManager(); got != tt.want {
				t.Errorf("SupportsPackageManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectWSL(t *testing.T) {
	// This test will only be meaningful on Linux
	if runtime.GOOS != "linux" {
		t.Skip("Skipping WSL test on non-Linux platform")
	}

	isWSL := detectWSL()

	// Check if /proc/version exists
	_, err := os.ReadFile("/proc/version")
	if err != nil {
		t.Skip("Cannot read /proc/version, skipping WSL test")
	}

	// We can't definitively test this without being on WSL or not,
	// but we can at least verify it returns a bool without error
	if isWSL {
		t.Log("Detected WSL environment")
	} else {
		t.Log("Not running on WSL")
	}
}

func TestDetectLinuxDistro(t *testing.T) {
	// Only run on Linux
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux distro test on non-Linux platform")
	}

	p := &Platform{}
	err := detectLinuxDistro(p)
	if err != nil {
		t.Fatalf("detectLinuxDistro() failed: %v", err)
	}

	if p.Distro == "" {
		t.Error("Distro should not be empty on Linux")
	}

	t.Logf("Detected distro: %s %s", p.Distro, p.DistroVersion)
}

func TestDetectLinuxPackageManager(t *testing.T) {
	// Only run on Linux
	if runtime.GOOS != "linux" {
		t.Skip("Skipping package manager test on non-Linux platform")
	}

	p := &Platform{}
	detectLinuxPackageManager(p)

	if p.PackageManager == "" {
		t.Error("PackageManager should not be empty")
	}

	t.Logf("Detected package manager: %s", p.PackageManager)
}
