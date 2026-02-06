package platform

import (
	"sort"
	"testing"
)

func TestResolve_AcrossManagers(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		manager   string
		want      string
	}{
		// --- fd: different names on different managers ---
		{"fd on apt", "fd", "apt", "fd-find"},
		{"fd on dnf", "fd", "dnf", "fd-find"},
		{"fd on yum", "fd", "yum", "fd-find"},
		{"fd on pacman", "fd", "pacman", "fd"},
		{"fd on brew", "fd", "brew", "fd"},

		// --- vim: enhanced variant on Red Hat family ---
		{"vim on apt", "vim", "apt", "vim"},
		{"vim on dnf", "vim", "dnf", "vim-enhanced"},
		{"vim on yum", "vim", "yum", "vim-enhanced"},
		{"vim on pacman", "vim", "pacman", "vim"},
		{"vim on brew", "vim", "brew", "vim"},

		// --- httpd: different across Debian vs Red Hat vs Arch ---
		{"httpd on apt", "httpd", "apt", "apache2"},
		{"httpd on dnf", "httpd", "dnf", "httpd"},
		{"httpd on yum", "httpd", "yum", "httpd"},
		{"httpd on pacman", "httpd", "pacman", "apache"},
		{"httpd on brew", "httpd", "brew", "httpd"},

		// --- python3: varies on Arch and Brew ---
		{"python3 on apt", "python3", "apt", "python3"},
		{"python3 on dnf", "python3", "dnf", "python3"},
		{"python3 on pacman", "python3", "pacman", "python"},
		{"python3 on brew", "python3", "brew", "python@3"},

		// --- openssh: client package naming varies ---
		{"openssh on apt", "openssh", "apt", "openssh-client"},
		{"openssh on dnf", "openssh", "dnf", "openssh-clients"},
		{"openssh on pacman", "openssh", "pacman", "openssh"},

		// --- docker: docker.io on Debian ---
		{"docker on apt", "docker", "apt", "docker.io"},
		{"docker on dnf", "docker", "dnf", "docker"},

		// --- nodejs: node on brew ---
		{"nodejs on apt", "nodejs", "apt", "nodejs"},
		{"nodejs on brew", "nodejs", "brew", "node"},

		// --- golang: go on Arch and brew ---
		{"golang on apt", "golang", "apt", "golang"},
		{"golang on pacman", "golang", "pacman", "go"},
		{"golang on brew", "golang", "brew", "go"},

		// --- rust: rustc on apt ---
		{"rust on apt", "rust", "apt", "rustc"},
		{"rust on dnf", "rust", "dnf", "rust"},

		// --- the_silver_searcher: silversearcher-ag on apt ---
		{"ag on apt", "the_silver_searcher", "apt", "silversearcher-ag"},
		{"ag on dnf", "the_silver_searcher", "dnf", "the_silver_searcher"},

		// --- shellcheck: capitalised on Red Hat ---
		{"shellcheck on apt", "shellcheck", "apt", "shellcheck"},
		{"shellcheck on dnf", "shellcheck", "dnf", "ShellCheck"},

		// --- build-essential: meta-package differences ---
		{"build-essential on apt", "build-essential", "apt", "build-essential"},
		{"build-essential on dnf", "build-essential", "dnf", "@development-tools"},
		{"build-essential on yum", "build-essential", "yum", "@development-tools"},
		{"build-essential on pacman", "build-essential", "pacman", "base-devel"},

		// --- delta: git-delta everywhere except canonical ---
		{"delta on apt", "delta", "apt", "git-delta"},
		{"delta on brew", "delta", "brew", "git-delta"},

		// --- lua: versioned on apt ---
		{"lua on apt", "lua", "apt", "lua5.4"},
		{"lua on dnf", "lua", "dnf", "lua"},

		// --- python3-pip: python-pip on Arch ---
		{"python3-pip on apt", "python3-pip", "apt", "python3-pip"},
		{"python3-pip on pacman", "python3-pip", "pacman", "python-pip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePackageName(tt.canonical, tt.manager)
			if got != tt.want {
				t.Errorf("ResolvePackageName(%q, %q) = %q, want %q",
					tt.canonical, tt.manager, got, tt.want)
			}
		})
	}
}

func TestResolve_FallbackForUnknownPackage(t *testing.T) {
	tests := []struct {
		name      string
		canonical string
		manager   string
	}{
		{"completely unknown on apt", "some-obscure-tool", "apt"},
		{"completely unknown on dnf", "another-unknown-pkg", "dnf"},
		{"completely unknown on brew", "not-in-registry", "brew"},
		{"completely unknown on pacman", "mystery-pkg", "pacman"},
		{"completely unknown on yum", "no-such-thing", "yum"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePackageName(tt.canonical, tt.manager)
			if got != tt.canonical {
				t.Errorf("ResolvePackageName(%q, %q) = %q, want original %q",
					tt.canonical, tt.manager, got, tt.canonical)
			}
		})
	}
}

func TestResolve_FallbackForUnknownManager(t *testing.T) {
	// A known canonical name but an unsupported manager should return the
	// canonical name unchanged.
	got := ResolvePackageName("fd", "zypper")
	if got != "fd" {
		t.Errorf("ResolvePackageName(fd, zypper) = %q, want %q", got, "fd")
	}
}

func TestMapPackageName_DelegatesToRegistry(t *testing.T) {
	// Ensure the existing public API still works after refactoring.
	tests := []struct {
		name        string
		genericName string
		manager     string
		want        string
	}{
		{"neovim on dnf", "neovim", "dnf", "neovim"},
		{"fd on apt", "fd", "apt", "fd-find"},
		{"fd on brew", "fd", "brew", "fd"},
		{"ripgrep on pacman", "ripgrep", "pacman", "ripgrep"},
		{"unmapped package", "some-random-pkg", "dnf", "some-random-pkg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MapPackageName(tt.genericName, tt.manager)
			if got != tt.want {
				t.Errorf("MapPackageName(%q, %q) = %q, want %q",
					tt.genericName, tt.manager, got, tt.want)
			}
		})
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewPackageMappingRegistry()

	if r.Len() != 0 {
		t.Fatalf("new registry should be empty, got %d", r.Len())
	}

	r.Register(PackageMapping{
		Canonical:   "test-pkg",
		Description: "a test package",
		Managers:    map[string]string{"apt": "test-pkg-apt"},
	})

	if r.Len() != 1 {
		t.Fatalf("registry should have 1 mapping, got %d", r.Len())
	}

	got := r.Resolve("test-pkg", "apt")
	if got != "test-pkg-apt" {
		t.Errorf("Resolve(test-pkg, apt) = %q, want %q", got, "test-pkg-apt")
	}
}

func TestRegistry_RegisterOverwrite(t *testing.T) {
	r := NewPackageMappingRegistry()

	r.Register(PackageMapping{
		Canonical: "mypkg",
		Managers:  map[string]string{"apt": "old-name"},
	})
	r.Register(PackageMapping{
		Canonical: "mypkg",
		Managers:  map[string]string{"apt": "new-name"},
	})

	if r.Len() != 1 {
		t.Fatalf("overwriting should not increase count, got %d", r.Len())
	}

	got := r.Resolve("mypkg", "apt")
	if got != "new-name" {
		t.Errorf("Resolve after overwrite = %q, want %q", got, "new-name")
	}
}

func TestRegistry_GetMapping(t *testing.T) {
	r := NewPackageMappingRegistry()

	r.Register(PackageMapping{
		Canonical:   "getme",
		Description: "test description",
		Managers:    map[string]string{"apt": "getme-apt", "dnf": "getme-dnf"},
	})

	m := r.GetMapping("getme")
	if m == nil {
		t.Fatal("GetMapping returned nil for a registered package")
	}

	if m.Canonical != "getme" {
		t.Errorf("Canonical = %q, want %q", m.Canonical, "getme")
	}
	if m.Description != "test description" {
		t.Errorf("Description = %q, want %q", m.Description, "test description")
	}
	if m.Managers["apt"] != "getme-apt" {
		t.Errorf("Managers[apt] = %q, want %q", m.Managers["apt"], "getme-apt")
	}
	if m.Managers["dnf"] != "getme-dnf" {
		t.Errorf("Managers[dnf] = %q, want %q", m.Managers["dnf"], "getme-dnf")
	}
}

func TestRegistry_GetMapping_ReturnsNilForUnknown(t *testing.T) {
	r := NewPackageMappingRegistry()

	m := r.GetMapping("nonexistent")
	if m != nil {
		t.Errorf("GetMapping for unknown package should return nil, got %+v", m)
	}
}

func TestRegistry_GetMapping_ReturnsCopy(t *testing.T) {
	r := NewPackageMappingRegistry()

	r.Register(PackageMapping{
		Canonical: "immutable",
		Managers:  map[string]string{"apt": "original"},
	})

	// Mutate the returned copy.
	m := r.GetMapping("immutable")
	m.Managers["apt"] = "mutated"

	// The registry's internal state should be unaffected.
	got := r.Resolve("immutable", "apt")
	if got != "original" {
		t.Errorf("registry was mutated via GetMapping copy: Resolve = %q, want %q", got, "original")
	}
}

func TestRegistry_Canonicals(t *testing.T) {
	r := NewPackageMappingRegistry()
	r.Register(PackageMapping{Canonical: "beta"})
	r.Register(PackageMapping{Canonical: "alpha"})
	r.Register(PackageMapping{Canonical: "gamma"})

	names := r.Canonicals()

	expected := []string{"alpha", "beta", "gamma"}
	if len(names) != len(expected) {
		t.Fatalf("Canonicals() len = %d, want %d", len(names), len(expected))
	}
	for i, n := range names {
		if n != expected[i] {
			t.Errorf("Canonicals()[%d] = %q, want %q", i, n, expected[i])
		}
	}
}

func TestRegistry_Canonicals_ReturnsSorted(t *testing.T) {
	r := NewPackageMappingRegistry()
	r.Register(PackageMapping{Canonical: "zulu"})
	r.Register(PackageMapping{Canonical: "mike"})
	r.Register(PackageMapping{Canonical: "alpha"})
	r.Register(PackageMapping{Canonical: "delta"})

	names := r.Canonicals()

	if !sort.StringsAreSorted(names) {
		t.Errorf("Canonicals() should return sorted names, got %v", names)
	}
}

func TestDefaultRegistry_IsPopulated(t *testing.T) {
	r := GetDefaultRegistry()

	// The default registry should have at least 30 mappings.
	if r.Len() < 30 {
		t.Errorf("default registry has %d mappings, expected at least 30", r.Len())
	}
}

func TestDefaultRegistry_ContainsExpectedPackages(t *testing.T) {
	r := GetDefaultRegistry()

	expected := []string{
		"neovim", "vim", "zsh", "fish", "python3", "nodejs", "golang",
		"fd", "ripgrep", "fzf", "bat", "jq", "git", "docker", "stow",
		"httpd", "openssh", "tmux", "curl", "wget", "make", "cmake",
	}

	for _, canonical := range expected {
		m := r.GetMapping(canonical)
		if m == nil {
			t.Errorf("default registry missing expected package %q", canonical)
		}
	}
}

func TestDefaultRegistry_AllMappingsHaveFiveManagers(t *testing.T) {
	r := GetDefaultRegistry()
	managers := []string{"apt", "dnf", "yum", "pacman", "brew"}

	for _, canonical := range r.Canonicals() {
		m := r.GetMapping(canonical)
		for _, mgr := range managers {
			if _, ok := m.Managers[mgr]; !ok {
				t.Errorf("mapping %q missing manager %q", canonical, mgr)
			}
		}
	}
}
