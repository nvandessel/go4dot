# 🐹 GopherDot - Complete Implementation Plan

**Project:** A Go-based CLI tool for managing dotfiles across multiple machines with interactive setup, platform detection, and dependency management.

**Repository:** `github.com/nvandessel/gopherdot`

**Status:** 🚧 Under Active Development - **36% Complete (5/14 phases)**

---

## 📊 Current Status (2026-01-02)

### ✅ Completed Phases (5/14)
- **Phase 0**: Project Setup - Full Go project structure, dependencies, Makefile
- **Phase 1**: Platform Detection - OS/distro/package manager detection  
- **Phase 2**: Package Managers - DNF, APT, Brew, Pacman, YUM implementations
- **Phase 3**: Config Loading - YAML parsing, validation, discovery
- **Phase 4**: Dependencies - Checking and installation of system packages
- **Phase 5**: Stow Management - Symlink creation/removal with GNU stow

### 🎯 What Works Now
```bash
# Commands available:
gopherdot detect                          # Show platform info
gopherdot config validate [path]          # Validate .gopherdot.yaml
gopherdot config show [path]              # Display config
gopherdot deps check [path]               # Check dependency status
gopherdot deps install [path]             # Install missing deps
gopherdot stow add <config> [path]        # Stow a config
gopherdot stow remove <config> [path]     # Unstow a config
gopherdot stow refresh [path]             # Refresh all configs
```

### 📈 Project Stats
- **Lines of Code**: ~4,500+
- **Tests**: 39 passing (25-80% coverage per module)
- **Commands**: 12 working commands
- **Platforms**: Linux (Fedora, Ubuntu, Arch), macOS, WSL

### ⏳ Next Up - Phase 6: External Dependencies
Clone external repos (plugin managers, themes, etc.) from GitHub.

---

## Table of Contents

- [Executive Summary](#executive-summary)
- [Project Architecture](#project-architecture)
- [Core Components](#core-components)
- [Configuration File Specification](#configuration-file-specification)
- [Command Structure](#command-structure)
- [Go Project Structure](#go-project-structure)
- [Implementation Phases](#implementation-phases)
- [Testing Strategy](#testing-strategy)
- [Go Learning Resources](#go-learning-resources)
- [Timeline Estimate](#timeline-estimate)
- [Success Metrics](#success-metrics)

---

## Executive Summary

### What We're Building

A **standalone CLI tool** (GopherDot) that manages dotfiles repositories with the following features:

- ✅ **Platform detection** - Automatically detect OS, distro, and package manager
- ✅ **Dependency management** - Check for and install required tools
- ✅ **Interactive setup** - Beautiful TUI with prompts and progress indicators
- ✅ **Machine-specific config** - Prompt for values that differ per machine (git name, email, GPG keys)
- ✅ **Stow management** - Safely symlink configs with conflict detection
- ✅ **External dependencies** - Clone plugin managers, themes, etc. from GitHub
- ✅ **Health checking** - Doctor command to validate installation
- ✅ **Universal** - Works with ANY dotfiles repo that has a `.gopherdot.yaml` config file

### Key Design Decisions

- **Name:** GopherDot (playful, memorable, Go-themed)
- **Two separate repos:** CLI tool + your dotfiles (keeps concerns separated)
- **Zero-cost hosting:** GitHub Pages + GitHub Releases
- **Versioning:** Semantic versioning (v1.0.0, v1.1.0, etc.)
- **Testing:** Unit tests, integration tests, and example dotfiles
- **Distribution:** Bootstrap script (`curl | bash`), GitHub Releases, and `go install`
- **Init command:** Generate `.gopherdot.yaml` by scanning existing dotfiles

### User Experience Flow

**For you (or anyone using GopherDot):**

```bash
# Clone your dotfiles
git clone https://github.com/nvandessel/dotfiles.git ~/dotfiles
cd ~/dotfiles

# Run the bootstrap script (installs GopherDot + runs setup)
./install.sh

# Or manually:
curl -fsSL https://raw.githubusercontent.com/nvandessel/gopherdot/main/install.sh | bash
gopherdot install
```

**For someone creating their own dotfiles with GopherDot:**

```bash
cd ~/my-dotfiles

# Initialize GopherDot config (scans your dotfiles, generates .gopherdot.yaml)
gopherdot init

# Edit .gopherdot.yaml to customize
vim .gopherdot.yaml

# Run setup
gopherdot install
```

---

## Project Architecture

### Two Separate Repositories

```
┌─────────────────────────────────────────────┐
│  github.com/nvandessel/gopherdot           │  ← The CLI Tool
│                                             │
│  • Go binary that manages dotfiles         │
│  • Distributed via GitHub Releases         │
│  • Works with ANY dotfiles repo            │
└─────────────────────────────────────────────┘
                    ↓ manages
┌─────────────────────────────────────────────┐
│  github.com/nvandessel/dotfiles            │  ← Your Dotfiles
│  (and anyone else's dotfiles!)             │
│                                             │
│  • Config files (git, nvim, tmux, etc.)    │
│  • .gopherdot.yaml (manifest)              │
│  • install.sh (bootstraps GopherDot)       │
└─────────────────────────────────────────────┘
```

### Why Separate Repos?

**Advantages:**

- ✅ **Reusable** - Anyone can use GopherDot with their own dotfiles
- ✅ **Versioned independently** - Release CLI tool separately from your dotfiles
- ✅ **Cleaner** - Your dotfiles stay pure config files
- ✅ **Community project** - Others can contribute to the tool
- ✅ **Standard practice** - Common pattern (Homebrew manages formulae, not itself)

### How They Integrate

1. **GopherDot discovers dotfiles:**
   - Checks current directory for `.gopherdot.yaml`
   - Checks `~/dotfiles`
   - Checks `~/.dotfiles`
   - Prompts if not found

2. **GopherDot reads `.gopherdot.yaml`** from your dotfiles repo

3. **GopherDot manages everything:**
   - Platform detection
   - Dependency installation
   - Machine-specific prompts
   - Stowing configs
   - Cloning external deps
   - Health checking

---

## Core Components

### 1. Configuration File: `.gopherdot.yaml`

**Location:** Root of dotfiles repository

**Purpose:** Declarative manifest that tells GopherDot:
- What configs exist and where
- What dependencies are needed
- Platform compatibility
- Machine-specific prompts
- External dependencies to clone
- Post-install instructions

**Schema Version:** `1.0` (allows evolution without breaking old configs)

**Example:**

```yaml
schema_version: "1.0"

metadata:
  name: my-dotfiles
  author: Your Name
  repository: https://github.com/user/dotfiles
  description: My personal dotfiles
  version: 1.0.0

dependencies:
  critical:
    - git
    - stow
    - zsh
  
  core:
    - name: nvim
      binary: nvim
      package:
        dnf: neovim
        apt: neovim
        brew: neovim

configs:
  core:
    - name: git
      path: git
      description: Git configuration
      platforms: [linux, macos, windows]
      requires_machine_config: true
    
    - name: nvim
      path: nvim
      description: Neovim configuration
      platforms: [linux, macos, windows]
      depends_on: [nvim]

external:
  - name: Pure Prompt
    id: pure
    url: https://github.com/sindresorhus/pure.git
    destination: ~/.zsh/pure

machine_config:
  git:
    description: Git user configuration
    destination: ~/.gitconfig.local
    prompts:
      - id: user_name
        prompt: Full name for git commits
        type: text
        required: true
      - id: user_email
        prompt: Email for git commits
        type: text
        required: true
    template: |
      [user]
          name = {{ .user_name }}
          email = {{ .user_email }}
```

### 2. Command Structure

```
gopherdot
├── install         # Interactive setup (main command)
├── init            # Generate .gopherdot.yaml from existing dotfiles
├── doctor          # Health check and troubleshooting
├── update          # Pull latest dotfiles and restow
├── list            # Show installed configs
├── reconfigure     # Re-run machine-specific prompts
├── stow            # Manual stow operations
│   ├── add         # Stow a specific config
│   ├── remove      # Unstow a specific config
│   └── refresh     # Restow all active configs
├── uninstall       # Remove all symlinks
├── version         # Show version
└── help            # Help documentation
```

### 3. State Management

**Location:** `~/.config/gopherdot/state.json`

**Purpose:** Track what's installed, when, and where.

**Contents:**

```json
{
  "version": "1.0.0",
  "installed_at": "2026-01-02T10:30:00Z",
  "last_update": "2026-01-02T10:30:00Z",
  "dotfiles_path": "/home/nic/dotfiles",
  "platform": {
    "os": "linux",
    "distro": "fedora",
    "distro_version": "43",
    "package_manager": "dnf"
  },
  "configs_installed": [
    "git",
    "nvim",
    "tmux",
    "zsh"
  ],
  "machine_config": {
    "git": {
      "local_config_path": "~/.gitconfig.local",
      "has_gpg": true
    }
  },
  "external_deps": {
    "pure": {
      "installed": true,
      "path": "~/.zsh/pure",
      "last_update": "2026-01-02T10:30:00Z"
    }
  }
}
```

---

## Configuration File Specification

See the full `.gopherdot.yaml` specification with detailed examples in [CONFIG_SPEC.md](./docs/CONFIG_SPEC.md) (to be created).

### Key Sections

1. **`schema_version`** - Config format version (currently "1.0")
2. **`metadata`** - Project info (name, author, repo, description, version)
3. **`dependencies`** - System packages needed (critical, core, optional)
4. **`configs`** - Dotfile modules to stow (core, optional, platform-specific)
5. **`external`** - External repos to clone (plugin managers, themes)
6. **`machine_config`** - Machine-specific prompts and templates
7. **`archived`** - Old configs not installed (for documentation)
8. **`post_install`** - Message shown after successful install

---

## Command Structure

### `gopherdot install`

Interactive first-time setup.

**Flow:**
1. Welcome screen
2. Dependency check + install
3. Git configuration prompts
4. Config selection (checkboxes)
5. Additional config (Obsidian path, etc.)
6. Stow configs with progress
7. Install external deps (NvChad, TPM, Pure)
8. Success + next steps

**Flags:**
- `--auto` - Non-interactive, use defaults
- `--minimal` - Only core configs, no prompts
- `--skip-deps` - Skip dependency installation

### `gopherdot init`

Generate `.gopherdot.yaml` from existing dotfiles.

**Flow:**
1. Scan directory for configs
2. Detect what each directory is (git, nvim, tmux, etc.)
3. Interactive prompts for unknowns
4. Prompt for metadata (name, author, description)
5. Prompt for platform support
6. Generate `.gopherdot.yaml` with helpful comments
7. Show next steps

### `gopherdot doctor`

Health check and troubleshooting.

**Checks:**
- System dependencies present and correct version
- Stowed symlinks valid (not broken)
- `~/.gitconfig.local` exists and has required fields
- External dependencies exist (NvChad, TPM, Pure)
- Font availability (optional)
- Platform-specific checks

### `gopherdot update`

Pull latest dotfiles and update.

**Flow:**
1. Git pull in dotfiles directory
2. Show what changed
3. Check if `.gopherdot.yaml` changed
4. Offer to install new configs
5. Restow configs
6. Update external deps
7. Show migration notes if any

### `gopherdot list`

Show installed configs.

**Output:**
- Installed configs
- Available but not installed
- Platform-specific (not available on this platform)
- Archived configs

### `gopherdot reconfigure`

Re-run machine-specific prompts.

**Options:**
- Reconfigure everything
- Reconfigure specific parts (git, paths, etc.)

### `gopherdot stow`

Manual stow operations.

**Subcommands:**
- `add <config>` - Stow a specific config
- `remove <config>` - Unstow a specific config
- `refresh` - Restow all active configs
- `list` - Show available configs

### `gopherdot uninstall`

Remove dotfiles.

**Flow:**
1. Confirm action
2. Unstow all configs
3. Optionally remove external deps
4. Optionally remove machine config
5. Remove state file

### `gopherdot version`

Show version information.

**Output:**
- Version number
- Build time
- Go version
- Platform

---

## Go Project Structure

```
gopherdot/
├── cmd/
│   └── gopherdot/
│       └── main.go                 # Entry point
│
├── internal/                       # Private application code
│   ├── config/
│   │   ├── loader.go              # Load .gopherdot.yaml
│   │   ├── schema.go              # Struct definitions
│   │   ├── validator.go           # Validate config
│   │   └── init.go                # Generate config (for init command)
│   │
│   ├── platform/
│   │   ├── detect.go              # OS/distro detection
│   │   ├── packages.go            # Package manager abstraction
│   │   ├── packages_linux.go      # Linux-specific (dnf, apt, etc.)
│   │   └── packages_darwin.go     # macOS-specific (brew)
│   │
│   ├── deps/
│   │   ├── check.go               # Check if deps installed
│   │   ├── install.go             # Install system packages
│   │   ├── external.go            # Git clone external deps
│   │   └── recommend.go           # Recommend optional tools
│   │
│   ├── stow/
│   │   ├── manager.go             # Stow operations
│   │   ├── conflicts.go           # Handle stow conflicts
│   │   └── validate.go            # Validate stow success
│   │
│   ├── machine/
│   │   ├── prompts.go             # Machine-specific prompts
│   │   ├── templates.go           # Generate configs from templates
│   │   └── git.go                 # Git-specific helpers (GPG detection)
│   │
│   ├── doctor/
│   │   ├── check.go               # Health check logic
│   │   ├── report.go              # Generate health report
│   │   └── fixes.go               # Suggest fixes for issues
│   │
│   ├── state/
│   │   ├── state.go               # State management
│   │   ├── load.go                # Load state
│   │   └── save.go                # Save state
│   │
│   └── ui/
│       ├── prompts.go             # Huh-based prompts
│       ├── progress.go            # Progress bars and spinners
│       ├── styles.go              # Lipgloss styles
│       └── messages.go            # Success/error messages
│
├── pkg/                            # Public library code (if any)
│   └── gopherdot/
│       └── api.go                 # Public API (for future extensions)
│
├── docs/
│   ├── README.md
│   ├── installation.md            # How to install GopherDot
│   ├── getting-started.md         # Quick start guide
│   ├── config-reference.md        # .gopherdot.yaml specification
│   ├── commands.md                # Command reference
│   └── creating-dotfiles.md       # Guide for creating dotfiles
│
├── examples/
│   ├── minimal/                   # Minimal example dotfiles
│   │   ├── git/.gitconfig
│   │   ├── zsh/.zshrc
│   │   └── .gopherdot.yaml
│   │
│   └── advanced/                  # Full-featured example
│       ├── git/
│       ├── nvim/
│       ├── tmux/
│       ├── .gopherdot.yaml
│       └── README.md
│
├── scripts/
│   ├── install.sh                 # Bootstrap installer (curl | bash)
│   ├── build.sh                   # Build all platforms
│   └── release.sh                 # Create GitHub release
│
├── .github/
│   └── workflows/
│       ├── release.yml            # Build & release on tags
│       ├── test.yml               # Run tests on PR
│       └── lint.yml               # Go linting
│
├── go.mod
├── go.sum
├── Makefile                       # Build automation
├── README.md                      # Main documentation
├── PLAN.md                        # This file!
├── LICENSE                        # MIT license
└── .gitignore
```

---

## Implementation Phases

### Phase 0: Project Setup & Foundation (2-3 hours)

**Goal:** Get the basic Go project structure in place and testable.

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create GitHub repository
- [x] Create project directory
- [x] Write PLAN.md
- [x] Initialize Go module
- [x] Add dependencies (Cobra, Bubbletea, Huh, Bubbles, Lipgloss, YAML)
- [x] Create project structure (directories)
- [x] Create basic `main.go` with version command
- [x] Create Makefile
- [x] Test build: `make build && ./gopherdot version`
- [x] Create basic README.md
- [x] Initialize git and make first commit

**Deliverables:**
- Buildable Go project
- Basic CLI with version command
- Project structure in place
- Can run `make build` and `./gopherdot version`

**What you'll learn:**
- Go module initialization
- Cobra CLI framework basics
- Go project structure conventions
- Makefile for Go projects

**Detailed Implementation:** See [PHASE_0.md](./docs/PHASE_0.md) (to be created)

---

### Phase 1: Platform Detection (3-4 hours)

**Goal:** Detect OS, distro, and package manager reliably.

**Status:** ✅ COMPLETED

**Tasks:**

- [ ] Create `internal/platform/detect.go`
- [ ] Implement OS detection (`runtime.GOOS`)
- [ ] Implement Linux distro detection (parse `/etc/os-release`)
- [ ] Implement WSL detection (check `/proc/version`)
- [ ] Implement package manager detection (check for binaries)
- [ ] Add `gopherdot detect` command for testing
- [ ] Write unit tests for detection logic

**Deliverables:**
- Platform detection working on Linux/macOS/WSL
- `gopherdot detect` command shows platform info
- Unit tests passing

**What you'll learn:**
- Reading files in Go
- String parsing
- Structs and methods
- Unit testing in Go

---

### Phase 2: Package Manager Abstraction (4-5 hours)

**Goal:** Abstract package installation across different package managers.

**Status:** ✅ COMPLETED

**Tasks:**

- [ ] Create `PackageManager` interface
- [ ] Implement DNF manager (Fedora/RHEL)
- [ ] Implement APT manager (Ubuntu/Debian)
- [ ] Implement Brew manager (macOS)
- [ ] Handle sudo caching
- [ ] Add package name mapping
- [ ] Write tests with mocked commands

**Deliverables:**
- Package manager abstraction working
- Can install packages on DNF/APT/Brew
- Tests for package operations

**What you'll learn:**
- Interfaces in Go
- Running shell commands (`os/exec`)
- Error handling
- Polymorphism

---

### Phase 3: Config Schema & Loading (3-4 hours)

**Goal:** Parse `.gopherdot.yaml` files.

**Status:** ✅ COMPLETED

**Tasks:**

- [ ] Create config structs in `internal/config/schema.go`
- [ ] Implement YAML loading in `internal/config/loader.go`
- [ ] Implement validation in `internal/config/validator.go`
- [ ] Add `gopherdot config validate` command
- [ ] Add `gopherdot config show` command
- [ ] Write tests for loading and validation

**Deliverables:**
- Can load `.gopherdot.yaml` files
- Validation working
- `gopherdot config` commands
- Tests passing

**What you'll learn:**
- YAML parsing with `gopkg.in/yaml.v3`
- Struct tags
- Error handling
- File I/O

---

### Phase 4: Dependency Checking & Installation (5-6 hours)

**Goal:** Check for required tools and install if missing.

**Status:** ✅ COMPLETED

**Tasks:**

- [ ] Create `internal/deps/check.go` for checking deps
- [ ] Create `internal/deps/install.go` for installing deps
- [ ] Implement interactive installation flow
- [ ] Add progress indicators
- [ ] Handle installation failures gracefully
- [ ] Add `gopherdot doctor --deps-only` command
- [ ] Write tests with mocked installs

**Deliverables:**
- Can check for dependencies
- Can install missing packages interactively
- Progress indicators working
- Tests passing

**What you'll learn:**
- Working with slices
- Interactive prompts with Huh (Charm's form library)
- Progress indicators
- Error aggregation

---

### Phase 5: Stow Management (4-5 hours)

**Goal:** Stow and unstow configs safely.

**Status:** ✅ COMPLETED

**Tasks:**

- [ ] Create `internal/stow/manager.go`
- [ ] Implement stow/unstow operations
- [ ] Detect and handle conflicts
- [ ] Validate symlinks after stowing
- [ ] Add manual stow commands
- [ ] Write tests with mocked stow

**Deliverables:**
- Can stow/unstow configs
- Conflict detection working
- Manual stow commands (`gopherdot stow add/remove`)
- Tests passing

**What you'll learn:**
- Running external commands
- File system operations
- Symlink handling
- Error recovery

---

### Phase 6: External Dependencies (3-4 hours)

**Goal:** Clone external repos (Pure, TPM, NvChad).

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Create `internal/deps/external.go`
- [ ] Implement git clone operations
- [ ] Handle different clone methods (clone vs copy)
- [ ] Implement conditional cloning
- [ ] Show progress during cloning
- [ ] Write tests with mocked git operations

**Deliverables:**
- Can clone external dependencies
- Copy method working (for NvChad)
- Progress indicators
- Tests passing

**What you'll learn:**
- Git operations from Go
- Conditional logic
- File copying
- Progress UX

---

### Phase 7: Machine-Specific Config (4-5 hours)

**Goal:** Prompt for machine-specific values and generate config files.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Create `internal/machine/prompts.go`
- [ ] Create `internal/machine/templates.go`
- [ ] Implement GPG key detection
- [ ] Handle different prompt types
- [ ] Implement template rendering
- [ ] Write tests for prompts and templates

**Deliverables:**
- Can prompt for machine-specific config
- Template rendering working
- GPG key detection
- Generated `~/.gitconfig.local`
- Tests passing

**What you'll learn:**
- Go templates (`text/template`)
- Interactive prompts
- String manipulation
- File writing

---

### Phase 8: Main Install Command (4-5 hours)

**Goal:** Orchestrate the full installation flow.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Create `internal/setup/setup.go`
- [ ] Implement full install orchestration
- [ ] Add config selection UI
- [ ] Implement beautiful progress UX
- [ ] Handle errors gracefully
- [ ] Add flags (--auto, --minimal, --skip-deps)
- [ ] Write integration tests

**Deliverables:**
- Full `gopherdot install` command working
- Beautiful interactive UX
- Error handling
- Integration tests

**What you'll learn:**
- Orchestrating complex flows
- Error handling strategies
- UX design for CLI
- Integration testing

---

### Phase 9: Doctor Command (3-4 hours)

**Goal:** Health check and troubleshooting.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Create `internal/doctor/check.go`
- [ ] Create `internal/doctor/report.go`
- [ ] Implement all health checks
- [ ] Generate beautiful health report
- [ ] Suggest fixes for common issues
- [ ] Write tests for checks

**Deliverables:**
- `gopherdot doctor` command working
- Beautiful health report
- Helpful suggestions
- Tests passing

**What you'll learn:**
- System inspection
- Report formatting
- Helpful error messages
- User experience design

---

### Phase 10: Additional Commands (4-5 hours)

**Goal:** Update, list, reconfigure, uninstall commands.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Implement `gopherdot update`
- [ ] Implement `gopherdot list`
- [ ] Implement `gopherdot reconfigure`
- [ ] Implement `gopherdot uninstall`
- [ ] Implement state management
- [ ] Write tests for each command

**Deliverables:**
- All maintenance commands working
- State management robust
- Tests passing

**What you'll learn:**
- JSON marshaling
- Git operations
- State management
- File removal safely

---

### Phase 11: Init Command (4-5 hours)

**Goal:** Generate `.gopherdot.yaml` from existing dotfiles.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Create `internal/config/init.go`
- [ ] Implement directory scanning
- [ ] Implement config type detection
- [ ] Create interactive wizard
- [ ] Generate YAML with comments
- [ ] Write tests for init logic

**Deliverables:**
- `gopherdot init` command working
- Can detect common configs
- Generated YAML is valid
- Tests passing

**What you'll learn:**
- Directory traversal
- Pattern matching
- YAML generation
- Interactive wizards

---

### Phase 12: Distribution & Release (3-4 hours)

**Goal:** Make GopherDot easy to install.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Create `scripts/install.sh` (bootstrap)
- [ ] Create `scripts/build.sh` (cross-compile)
- [ ] Set up GitHub Actions (release.yml)
- [ ] Set up GitHub Actions (test.yml)
- [ ] Test release process
- [ ] Update Makefile with release target

**Deliverables:**
- Bootstrap script working
- Cross-compilation working
- GitHub Actions set up
- Can create releases easily

**What you'll learn:**
- Cross-compilation in Go
- GitHub Actions
- Shell scripting
- Distribution strategies

---

### Phase 13: Documentation (3-4 hours)

**Goal:** Comprehensive, helpful documentation.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Write main README.md
- [ ] Write docs/installation.md
- [ ] Write docs/getting-started.md
- [ ] Write docs/config-reference.md
- [ ] Write docs/commands.md
- [ ] Write docs/creating-dotfiles.md
- [ ] Create example dotfiles (minimal & advanced)
- [ ] Add help text to all commands

**Deliverables:**
- Comprehensive documentation
- Example dotfiles
- Good help text in CLI

**What you'll learn:**
- Technical writing
- User documentation
- Example creation

---

### Phase 14: Polish & v1.0.0 (2-3 hours)

**Goal:** Final polish for v1.0.0 release.

**Status:** ⏳ PENDING

**Tasks:**

- [ ] Code cleanup
- [ ] Add version checking
- [ ] Add logo/branding
- [ ] Final testing on multiple platforms
- [ ] Create CHANGELOG.md
- [ ] Create v1.0.0 release

**Deliverables:**
- v1.0.0 released
- Tested on Fedora, macOS, WSL
- Ready for public use

---

## Testing Strategy

### Unit Tests

- Test individual functions in isolation
- Mock external dependencies (file system, commands)
- Fast, run on every change
- Goal: 70%+ code coverage

**Example:**

```go
// internal/platform/detect_test.go
func TestDetectOS(t *testing.T) {
    os := DetectOS()
    if os != "linux" && os != "darwin" && os != "windows" {
        t.Errorf("unexpected OS: %s", os)
    }
}
```

### Integration Tests

- Test full commands with example dotfiles
- Mock system operations where necessary
- Slower, run before releases
- Verify end-to-end flows

**Example:**

```go
// test/integration/install_test.go
func TestInstallCommand(t *testing.T) {
    // Set up example dotfiles
    // Run gopherdot install --auto
    // Verify symlinks created
    // Verify state saved
}
```

### Manual Testing

- Test on real machines (Fedora, macOS, WSL)
- Test with actual dotfiles
- Check UX flows
- Verify error handling

**Test Plan:**

1. Fresh Fedora VM - Run full install
2. macOS machine - Run full install
3. WSL2 Ubuntu - Run full install
4. Test all commands on each platform
5. Test error scenarios (missing deps, conflicts, etc.)

### Test Structure

```
gopherdot/
├── internal/
│   ├── config/
│   │   ├── loader.go
│   │   └── loader_test.go       # Unit tests
│   ├── platform/
│   │   ├── detect.go
│   │   └── detect_test.go
│   └── ...
├── test/
│   ├── integration/
│   │   ├── install_test.go      # Integration tests
│   │   └── doctor_test.go
│   └── fixtures/
│       └── example-dotfiles/    # Test dotfiles
```

---

## Go Learning Resources

Since you're learning Go through this project, here are some helpful resources:

### Official Resources

- **A Tour of Go**: https://go.dev/tour/ (1-2 hours, essential)
- **Effective Go**: https://go.dev/doc/effective_go (reference)
- **Go by Example**: https://gobyexample.com/ (quick examples)

### Video Tutorials

- **Freecodecamp Go Course** (7 hours): https://www.youtube.com/watch?v=YS4e4q9oBaU
- **Tech With Tim Go Tutorial**: https://www.youtube.com/watch?v=446E-r0rXHI

### Books

- **The Go Programming Language** by Donovan & Kernighan (best book)
- **Let's Go** by Alex Edwards (web-focused but good fundamentals)

### Concepts You'll Learn by Phase

1. **Phase 0-1**: Basic syntax, modules, packages, imports
2. **Phase 2-3**: Interfaces, structs, methods, error handling
3. **Phase 4-5**: Slices, maps, file I/O, os/exec
4. **Phase 6-7**: Templates, string manipulation, git operations
5. **Phase 8-9**: Orchestration, complex flows, error strategies
6. **Phase 10-14**: Testing, distribution, polish

### Tips

- Run `go fmt` often (formats code automatically)
- Use `go vet` to catch common mistakes
- Read standard library code (it's excellent Go)
- Use `gofmt -s` for simplification
- Install `golangci-lint` for comprehensive linting
- Ask questions as you go!

---

## Timeline Estimate

**Total: 50-60 hours** (spread over 4-6 weekends)

| Phase | Task | Hours | Weekend |
|-------|------|-------|---------|
| 0 | Project Setup | 2-3 | 1 |
| 1 | Platform Detection | 3-4 | 1 |
| 2 | Package Managers | 4-5 | 1-2 |
| 3 | Config Loading | 3-4 | 2 |
| 4 | Dependency Install | 5-6 | 2-3 |
| 5 | Stow Management | 4-5 | 3 |
| 6 | External Deps | 3-4 | 3 |
| 7 | Machine Config | 4-5 | 4 |
| 8 | Install Command | 4-5 | 4 |
| 9 | Doctor Command | 3-4 | 4-5 |
| 10 | Other Commands | 4-5 | 5 |
| 11 | Init Command | 4-5 | 5-6 |
| 12 | Distribution | 3-4 | 6 |
| 13 | Documentation | 3-4 | 6 |
| 14 | Polish & v1.0 | 2-3 | 6 |

**Realistic pace:** 8-10 hours per weekend = 6 weekends

**Aggressive pace:** 12-15 hours per weekend = 4 weekends

**Flexible approach:** Work through phases at your own pace!

---

## Success Metrics

**v1.0.0 is ready when:**

✅ Works on Linux (Fedora/Ubuntu), macOS, WSL  
✅ Can install your dotfiles successfully  
✅ Handles dependency installation  
✅ Generates machine-specific config  
✅ Doctor command validates installation  
✅ Documentation is complete  
✅ Example dotfiles work  
✅ `gopherdot init` generates valid config  
✅ Tests pass (70%+ coverage)  
✅ Binaries available on GitHub Releases  
✅ Bootstrap script works  

---

## Next Steps

### Immediate (Phase 0)

1. [x] Create GitHub repository
2. [x] Write this PLAN.md
3. [ ] Initialize Go module
4. [ ] Set up project structure
5. [ ] Create basic CLI with version command
6. [ ] Test build process

### Short Term (Phases 1-3)

- Platform detection
- Package manager abstraction
- Config loading

### Medium Term (Phases 4-8)

- Dependency management
- Stow operations
- Full install command

### Long Term (Phases 9-14)

- Maintenance commands
- Distribution
- Documentation
- v1.0.0 release

---

## Questions & Decisions

### Answered

- ✅ **Name:** GopherDot
- ✅ **Separate repo:** Yes, for reusability
- ✅ **License:** MIT
- ✅ **Versioning:** Semantic versioning
- ✅ **Testing:** All approaches (unit, integration, manual)
- ✅ **Distribution:** Bootstrap script + GitHub Releases
- ✅ **Dependency installation:** Prompt with batch install
- ✅ **Sudo handling:** Ask upfront, cache for session
- ✅ **Config selection:** Defaults per YAML, optional checkboxes

### To Be Decided

- Go version (recommend 1.21+)
- Commit strategy (per phase or smaller increments)
- Code style preferences
- Public vs private until v1.0
- Contributing guidelines

---

## Resources

### Project Links

- **Repository:** https://github.com/nvandessel/gopherdot
- **Issues:** https://github.com/nvandessel/gopherdot/issues
- **Releases:** https://github.com/nvandessel/gopherdot/releases

### Related Projects

- **GNU Stow:** https://www.gnu.org/software/stow/
- **Cobra CLI:** https://github.com/spf13/cobra
- **Bubbletea (TUI framework):** https://github.com/charmbracelet/bubbletea
- **Huh (forms & prompts):** https://github.com/charmbracelet/huh
- **Bubbles (TUI components):** https://github.com/charmbracelet/bubbles
- **Lipgloss (styling):** https://github.com/charmbracelet/lipgloss

### Inspiration

- **chezmoi:** https://www.chezmoi.io/ (another dotfile manager in Go)
- **yadm:** https://yadm.io/ (yet another dotfile manager)
- **dotbot:** https://github.com/anishathalye/dotbot (Python-based)

---

## Changelog

- **2026-01-02**: Initial plan created
- **2026-01-02**: Phases 0-5 completed in first session
  - **Phase 0**: ✅ Project setup, migrated from Survey to Bubbletea/Huh
  - **Phase 1**: ✅ Platform detection (Linux/macOS/WSL, distro, package manager)
  - **Phase 2**: ✅ Package manager abstraction (DNF, APT, Brew, Pacman, YUM)
  - **Phase 3**: ✅ Config schema & loading with validation
  - **Phase 4**: ✅ Dependency checking and installation
  - **Phase 5**: ✅ Stow management (add, remove, refresh)
  - **Progress**: 36% complete (5/14 phases), 39 tests passing, ~4,500 lines of code

---

**Let's build something awesome! 🐹🚀**
