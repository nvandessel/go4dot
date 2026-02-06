package platform

import "sync"

// PackageMapping represents a canonical package with its per-manager names.
type PackageMapping struct {
	// Canonical is the generic/canonical name used in configuration files.
	Canonical string

	// Description explains what this package provides.
	Description string

	// Managers maps package manager names to their specific package names.
	// Keys are manager names (e.g., "apt", "dnf", "brew", "pacman", "yum").
	Managers map[string]string
}

// PackageMappingRegistry holds a collection of package mappings and provides
// lookup capabilities for resolving canonical names to manager-specific names.
type PackageMappingRegistry struct {
	mu       sync.RWMutex
	mappings map[string]*PackageMapping
}

// NewPackageMappingRegistry creates an empty registry.
func NewPackageMappingRegistry() *PackageMappingRegistry {
	return &PackageMappingRegistry{
		mappings: make(map[string]*PackageMapping),
	}
}

// Register adds a PackageMapping to the registry. If a mapping with the same
// canonical name already exists, it is replaced.
func (r *PackageMappingRegistry) Register(m PackageMapping) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.mappings[m.Canonical] = &m
}

// Resolve translates a canonical package name into the manager-specific name.
// If no mapping exists for the canonical name, or the manager is not listed in
// the mapping, the original canonical name is returned unchanged.
func (r *PackageMappingRegistry) Resolve(canonical, manager string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.mappings[canonical]
	if !ok {
		return canonical
	}

	if name, ok := m.Managers[manager]; ok {
		return name
	}

	return canonical
}

// GetMapping returns the full PackageMapping for a canonical name, or nil if
// no mapping is registered.
func (r *PackageMappingRegistry) GetMapping(canonical string) *PackageMapping {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.mappings[canonical]
	if !ok {
		return nil
	}

	// Return a copy so callers cannot mutate internal state.
	cp := *m
	managers := make(map[string]string, len(m.Managers))
	for k, v := range m.Managers {
		managers[k] = v
	}
	cp.Managers = managers
	return &cp
}

// Len returns the number of registered mappings.
func (r *PackageMappingRegistry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.mappings)
}

// Canonicals returns a sorted slice of all registered canonical names.
func (r *PackageMappingRegistry) Canonicals() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.mappings))
	for k := range r.mappings {
		names = append(names, k)
	}
	return names
}

// ---------------------------------------------------------------------------
// Default registry (singleton)
// ---------------------------------------------------------------------------

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *PackageMappingRegistry
)

// GetDefaultRegistry returns the global, pre-populated PackageMappingRegistry.
// It is safe for concurrent use and is initialized only once.
func GetDefaultRegistry() *PackageMappingRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewPackageMappingRegistry()
		for _, m := range defaultMappings() {
			defaultRegistry.Register(m)
		}
	})
	return defaultRegistry
}

// ResolvePackageName is a convenience function that resolves a canonical
// package name using the default registry.
func ResolvePackageName(canonical, manager string) string {
	return GetDefaultRegistry().Resolve(canonical, manager)
}

// ---------------------------------------------------------------------------
// Default mappings data
// ---------------------------------------------------------------------------

// defaultMappings returns the built-in set of cross-manager package mappings.
//
//nolint:funlen // data table, length is expected
func defaultMappings() []PackageMapping {
	return []PackageMapping{
		// --- Editors ---
		{
			Canonical:   "neovim",
			Description: "Hyperextensible Vim-based text editor",
			Managers:    map[string]string{"apt": "neovim", "dnf": "neovim", "yum": "neovim", "pacman": "neovim", "brew": "neovim"},
		},
		{
			Canonical:   "vim",
			Description: "Vi IMproved text editor",
			Managers:    map[string]string{"apt": "vim", "dnf": "vim-enhanced", "yum": "vim-enhanced", "pacman": "vim", "brew": "vim"},
		},
		{
			Canonical:   "emacs",
			Description: "Extensible, customizable text editor",
			Managers:    map[string]string{"apt": "emacs", "dnf": "emacs", "yum": "emacs", "pacman": "emacs", "brew": "emacs"},
		},
		{
			Canonical:   "nano",
			Description: "Simple terminal text editor",
			Managers:    map[string]string{"apt": "nano", "dnf": "nano", "yum": "nano", "pacman": "nano", "brew": "nano"},
		},

		// --- Shells ---
		{
			Canonical:   "zsh",
			Description: "Z shell",
			Managers:    map[string]string{"apt": "zsh", "dnf": "zsh", "yum": "zsh", "pacman": "zsh", "brew": "zsh"},
		},
		{
			Canonical:   "fish",
			Description: "Friendly interactive shell",
			Managers:    map[string]string{"apt": "fish", "dnf": "fish", "yum": "fish", "pacman": "fish", "brew": "fish"},
		},
		{
			Canonical:   "bash",
			Description: "Bourne-Again SHell",
			Managers:    map[string]string{"apt": "bash", "dnf": "bash", "yum": "bash", "pacman": "bash", "brew": "bash"},
		},

		// --- Languages & Runtimes ---
		{
			Canonical:   "python3",
			Description: "Python 3 interpreter",
			Managers:    map[string]string{"apt": "python3", "dnf": "python3", "yum": "python3", "pacman": "python", "brew": "python@3"},
		},
		{
			Canonical:   "python3-pip",
			Description: "Python 3 package installer",
			Managers:    map[string]string{"apt": "python3-pip", "dnf": "python3-pip", "yum": "python3-pip", "pacman": "python-pip", "brew": "python@3"},
		},
		{
			Canonical:   "nodejs",
			Description: "JavaScript runtime built on V8",
			Managers:    map[string]string{"apt": "nodejs", "dnf": "nodejs", "yum": "nodejs", "pacman": "nodejs", "brew": "node"},
		},
		{
			Canonical:   "golang",
			Description: "Go programming language",
			Managers:    map[string]string{"apt": "golang", "dnf": "golang", "yum": "golang", "pacman": "go", "brew": "go"},
		},
		{
			Canonical:   "rust",
			Description: "Rust programming language (via rustup recommended)",
			Managers:    map[string]string{"apt": "rustc", "dnf": "rust", "yum": "rust", "pacman": "rust", "brew": "rust"},
		},
		{
			Canonical:   "ruby",
			Description: "Ruby programming language",
			Managers:    map[string]string{"apt": "ruby", "dnf": "ruby", "yum": "ruby", "pacman": "ruby", "brew": "ruby"},
		},
		{
			Canonical:   "lua",
			Description: "Lightweight scripting language",
			Managers:    map[string]string{"apt": "lua5.4", "dnf": "lua", "yum": "lua", "pacman": "lua", "brew": "lua"},
		},

		// --- Build Tools ---
		{
			Canonical:   "make",
			Description: "GNU Make build automation tool",
			Managers:    map[string]string{"apt": "make", "dnf": "make", "yum": "make", "pacman": "make", "brew": "make"},
		},
		{
			Canonical:   "cmake",
			Description: "Cross-platform build system generator",
			Managers:    map[string]string{"apt": "cmake", "dnf": "cmake", "yum": "cmake", "pacman": "cmake", "brew": "cmake"},
		},
		{
			Canonical:   "gcc",
			Description: "GNU Compiler Collection",
			Managers:    map[string]string{"apt": "gcc", "dnf": "gcc", "yum": "gcc", "pacman": "gcc", "brew": "gcc"},
		},
		{
			Canonical:   "build-essential",
			Description: "Essential build tools (compiler, make, etc.)",
			Managers:    map[string]string{"apt": "build-essential", "dnf": "gcc gcc-c++ make", "yum": "gcc gcc-c++ make", "pacman": "base-devel", "brew": "gcc"},
		},

		// --- CLI Tools ---
		{
			Canonical:   "fd",
			Description: "Fast and user-friendly alternative to find",
			Managers:    map[string]string{"apt": "fd-find", "dnf": "fd-find", "yum": "fd-find", "pacman": "fd", "brew": "fd"},
		},
		{
			Canonical:   "ripgrep",
			Description: "Fast recursive grep alternative",
			Managers:    map[string]string{"apt": "ripgrep", "dnf": "ripgrep", "yum": "ripgrep", "pacman": "ripgrep", "brew": "ripgrep"},
		},
		{
			Canonical:   "fzf",
			Description: "General-purpose command-line fuzzy finder",
			Managers:    map[string]string{"apt": "fzf", "dnf": "fzf", "yum": "fzf", "pacman": "fzf", "brew": "fzf"},
		},
		{
			Canonical:   "bat",
			Description: "Cat clone with syntax highlighting",
			Managers:    map[string]string{"apt": "bat", "dnf": "bat", "yum": "bat", "pacman": "bat", "brew": "bat"},
		},
		{
			Canonical:   "eza",
			Description: "Modern replacement for ls",
			Managers:    map[string]string{"apt": "eza", "dnf": "eza", "yum": "eza", "pacman": "eza", "brew": "eza"},
		},
		{
			Canonical:   "jq",
			Description: "Command-line JSON processor",
			Managers:    map[string]string{"apt": "jq", "dnf": "jq", "yum": "jq", "pacman": "jq", "brew": "jq"},
		},
		{
			Canonical:   "tree",
			Description: "Display directory structure as a tree",
			Managers:    map[string]string{"apt": "tree", "dnf": "tree", "yum": "tree", "pacman": "tree", "brew": "tree"},
		},
		{
			Canonical:   "htop",
			Description: "Interactive process viewer",
			Managers:    map[string]string{"apt": "htop", "dnf": "htop", "yum": "htop", "pacman": "htop", "brew": "htop"},
		},
		{
			Canonical:   "tmux",
			Description: "Terminal multiplexer",
			Managers:    map[string]string{"apt": "tmux", "dnf": "tmux", "yum": "tmux", "pacman": "tmux", "brew": "tmux"},
		},
		{
			Canonical:   "wget",
			Description: "Network downloader",
			Managers:    map[string]string{"apt": "wget", "dnf": "wget", "yum": "wget", "pacman": "wget", "brew": "wget"},
		},
		{
			Canonical:   "curl",
			Description: "Command-line URL transfer tool",
			Managers:    map[string]string{"apt": "curl", "dnf": "curl", "yum": "curl", "pacman": "curl", "brew": "curl"},
		},
		{
			Canonical:   "unzip",
			Description: "Extraction utility for ZIP archives",
			Managers:    map[string]string{"apt": "unzip", "dnf": "unzip", "yum": "unzip", "pacman": "unzip", "brew": "unzip"},
		},

		// --- Version Control ---
		{
			Canonical:   "git",
			Description: "Distributed version control system",
			Managers:    map[string]string{"apt": "git", "dnf": "git", "yum": "git", "pacman": "git", "brew": "git"},
		},
		{
			Canonical:   "lazygit",
			Description: "Simple terminal UI for git commands",
			Managers:    map[string]string{"apt": "lazygit", "dnf": "lazygit", "yum": "lazygit", "pacman": "lazygit", "brew": "lazygit"},
		},

		// --- Containers & Virtualisation ---
		{
			Canonical:   "docker",
			Description: "Container runtime",
			Managers:    map[string]string{"apt": "docker.io", "dnf": "docker", "yum": "docker", "pacman": "docker", "brew": "docker"},
		},
		{
			Canonical:   "podman",
			Description: "Daemonless container engine",
			Managers:    map[string]string{"apt": "podman", "dnf": "podman", "yum": "podman", "pacman": "podman", "brew": "podman"},
		},

		// --- Networking ---
		{
			Canonical:   "openssh",
			Description: "OpenSSH client and server",
			Managers:    map[string]string{"apt": "openssh-client", "dnf": "openssh-clients", "yum": "openssh-clients", "pacman": "openssh", "brew": "openssh"},
		},
		{
			Canonical:   "nmap",
			Description: "Network exploration and security auditing",
			Managers:    map[string]string{"apt": "nmap", "dnf": "nmap", "yum": "nmap", "pacman": "nmap", "brew": "nmap"},
		},

		// --- Web Servers ---
		{
			Canonical:   "httpd",
			Description: "Apache HTTP Server",
			Managers:    map[string]string{"apt": "apache2", "dnf": "httpd", "yum": "httpd", "pacman": "apache", "brew": "httpd"},
		},
		{
			Canonical:   "nginx",
			Description: "High-performance HTTP server and reverse proxy",
			Managers:    map[string]string{"apt": "nginx", "dnf": "nginx", "yum": "nginx", "pacman": "nginx", "brew": "nginx"},
		},

		// --- Stow (core dependency) ---
		{
			Canonical:   "stow",
			Description: "GNU Stow symlink farm manager",
			Managers:    map[string]string{"apt": "stow", "dnf": "stow", "yum": "stow", "pacman": "stow", "brew": "stow"},
		},

		// --- Terminal Emulators & Fonts ---
		{
			Canonical:   "kitty",
			Description: "GPU-accelerated terminal emulator",
			Managers:    map[string]string{"apt": "kitty", "dnf": "kitty", "yum": "kitty", "pacman": "kitty", "brew": "kitty"},
		},
		{
			Canonical:   "alacritty",
			Description: "GPU-accelerated terminal emulator",
			Managers:    map[string]string{"apt": "alacritty", "dnf": "alacritty", "yum": "alacritty", "pacman": "alacritty", "brew": "alacritty"},
		},

		// --- Miscellaneous ---
		{
			Canonical:   "shellcheck",
			Description: "Static analysis tool for shell scripts",
			Managers:    map[string]string{"apt": "shellcheck", "dnf": "ShellCheck", "yum": "ShellCheck", "pacman": "shellcheck", "brew": "shellcheck"},
		},
		{
			Canonical:   "the_silver_searcher",
			Description: "Code-searching tool similar to ack (ag)",
			Managers:    map[string]string{"apt": "silversearcher-ag", "dnf": "the_silver_searcher", "yum": "the_silver_searcher", "pacman": "the_silver_searcher", "brew": "the_silver_searcher"},
		},
		{
			Canonical:   "delta",
			Description: "Syntax-highlighting pager for git diffs",
			Managers:    map[string]string{"apt": "git-delta", "dnf": "git-delta", "yum": "git-delta", "pacman": "git-delta", "brew": "git-delta"},
		},
	}
}
