# 🐹 go4dot - Complete Implementation Plan

**Project:** A Go-based CLI tool for managing dotfiles across multiple machines with interactive setup, platform detection, and dependency management.

**Repository:** `github.com/nvandessel/go4dot`

**Status:** 🚧 Under Active Development - **78% Complete (11/14 phases)**

---

## 📊 Current Status (2026-01-04)

### ✅ Completed Phases (11/14)
- **Phase 0**: Project Setup - Full Go project structure, dependencies, Makefile
- **Phase 1**: Platform Detection - OS/distro/package manager detection
- **Phase 2**: Package Managers - DNF, APT, Brew, Pacman, YUM implementations
- **Phase 3**: Config Loading - YAML parsing, validation, discovery
- **Phase 4**: Dependencies - Checking and installation of system packages
- **Phase 5**: Stow Management - Symlink creation/removal with GNU stow
- **Phase 6**: External Dependencies - Clone external repos with git, conditions, copy method
- **Phase 7**: Machine Config - Interactive prompts, Go templates, GPG/SSH detection
- **Phase 8**: Install Command - Full orchestration with --auto, --minimal, --skip-* flags
- **Phase 9**: Doctor Command - Health checks, symlink validation, fix suggestions
- **Phase 10**: Additional Commands - State management, list, update, reconfigure, uninstall
- **Phase 11**: Init Command - Generate .go4dot.yaml from existing dotfiles

### 🎯 What Works Now
```bash
# Commands available:
g4d install [path]                  # Full installation (main command!)
g4d init [path]                     # Initialize config from existing dotfiles
g4d detect                          # Show platform info
g4d config validate [path]          # Validate .go4dot.yaml
g4d config show [path]              # Display config
g4d deps check [path]               # Check dependency status
g4d deps install [path]             # Install missing deps
g4d stow add <config> [path]        # Stow a config
g4d stow remove <config> [path]     # Unstow a config
g4d stow refresh [path]             # Refresh all configs
g4d external status [path]          # Show external deps status
g4d external clone [id] [path]      # Clone external deps
g4d external update [id] [path]     # Update external deps
g4d external remove <id> [path]     # Remove an external dep
g4d machine info                    # Show system info (git, GPG, SSH)
g4d machine status [path]           # Show machine config status
g4d machine configure [id] [path]   # Configure machine settings
g4d machine show <id> [path]        # Preview a machine config
g4d machine remove <id> [path]      # Remove a machine config
g4d doctor [-v] [path]              # Health check with fix suggestions
g4d list [-a] [path]                # List installed and available configs
g4d update [--external] [path]      # Pull latest and restow
g4d reconfigure [id] [path]         # Re-run machine config prompts
g4d uninstall [-f] [path]           # Remove symlinks and state
```

### 📈 Project Stats
- **Lines of Code**: ~9,800+
- **Tests**: 121+ passing (25-80% coverage per module)
- **Commands**: 28 working commands
- **Platforms**: Linux (Fedora, Ubuntu, Arch), macOS, WSL

### ⏳ Next Up - Phase 14: Polish & v1.0.0
Final code cleanup, version checking, and v1.0.0 release.

**Tasks:**
- [x] Create `internal/ui` package (styles, menu, spinner)
- [x] Implement interactive dashboard
- [x] Refactor commands to use new UI
- [x] Add version checking
- [ ] Run linter and fix issues
- [ ] Create v1.0.0 release

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

A **standalone CLI tool** (go4dot) that manages dotfiles repositories with the following features:

- ✅ **Platform detection** - Automatically detect OS, distro, and package manager
- ✅ **Dependency management** - Check for and install required tools
- ✅ **Interactive setup** - Beautiful TUI with prompts and progress indicators
- ✅ **Machine-specific config** - Prompt for values that differ per machine (git name, email, GPG keys)
- ✅ **Stow management** - Safely symlink configs with conflict detection
- ✅ **External dependencies** - Clone plugin managers, themes, etc. from GitHub
- ✅ **Health checking** - Doctor command to validate installation
- ✅ **Universal** - Works with ANY dotfiles repo that has a `.go4dot.yaml` config file

### Key Design Decisions

- **Name:** go4dot (playful, memorable, Go-themed)
- **Two separate repos:** CLI tool + your dotfiles (keeps concerns separated)
- **Zero-cost hosting:** GitHub Pages + GitHub Releases
- **Versioning:** Semantic versioning (v1.0.0, v1.1.0, etc.)
- **Testing:** Unit tests, integration tests, and example dotfiles
- **Distribution:** Bootstrap script (`curl | bash`), GitHub Releases, and `go install`
- **Init command:** Generate `.go4dot.yaml` by scanning existing dotfiles

### User Experience Flow

**For you (or anyone using go4dot):**

```bash
# Clone your dotfiles
git clone https://github.com/nvandessel/dotfiles.git ~/dotfiles
cd ~/dotfiles

# Run the bootstrap script (installs go4dot + runs setup)
./install.sh

# Or manually:
curl -fsSL https://raw.githubusercontent.com/nvandessel/gopherdot/main/install.sh | bash
g4d install
```

**For someone creating their own dotfiles with go4dot:**

```bash
cd ~/my-dotfiles

# Initialize go4dot config (scans your dotfiles, generates .go4dot.yaml)
g4d init

# Edit .go4dot.yaml to customize
vim .go4dot.yaml

# Run setup
g4d install
```

---

## Project Architecture

### Two Separate Repositories

```
┌─────────────────────────────────────────────┐
│  github.com/nvandessel/go4dot           │  ← The CLI Tool
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
│  • .go4dot.yaml (manifest)              │
│  • install.sh (bootstraps go4dot)       │
└─────────────────────────────────────────────┘
```

### Why Separate Repos?

**Advantages:**

- ✅ **Reusable** - Anyone can use go4dot with their own dotfiles
- ✅ **Versioned independently** - Release CLI tool separately from your dotfiles
- ✅ **Cleaner** - Your dotfiles stay pure config files
- ✅ **Community project** - Others can contribute to the tool
- ✅ **Standard practice** - Common pattern (Homebrew manages formulae, not itself)

### How They Integrate

1. **go4dot discovers dotfiles:**
   - Checks current directory for `.go4dot.yaml`
   - Checks `~/dotfiles`
   - Checks `~/.dotfiles`
   - Prompts if not found

2. **go4dot reads `.go4dot.yaml`** from your dotfiles repo

3. **go4dot manages everything:**
   - Platform detection
   - Dependency installation
   - Machine-specific prompts
   - Stowing configs
   - Cloning external deps
   - Health checking

---

## Core Components

### 1. Configuration File: `.go4dot.yaml`

**Location:** Root of dotfiles repository

**Purpose:** Declarative manifest that tells go4dot:
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
g4d
├── install         # Interactive setup (main command)
├── init            # Generate .go4dot.yaml from existing dotfiles
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

See the full `.go4dot.yaml` specification with detailed examples in [CONFIG_SPEC.md](./docs/CONFIG_SPEC.md) (to be created).

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

### `g4d install`

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

### `g4d init`

Generate `.go4dot.yaml` from existing dotfiles.

**Flow:**
1. Scan directory for configs
2. Detect what each directory is (git, nvim, tmux, etc.)
3. Interactive prompts for unknowns
4. Prompt for metadata (name, author, description)
5. Prompt for platform support
6. Generate `.go4dot.yaml` with helpful comments
7. Show next steps

### `g4d doctor`

Health check and troubleshooting.

**Checks:**
- System dependencies present and correct version
- Stowed symlinks valid (not broken)
- `~/.gitconfig.local` exists and has required fields
- External dependencies exist (NvChad, TPM, Pure)
- Font availability (optional)
- Platform-specific checks

### `g4d update`

Pull latest dotfiles and update.

**Flow:**
1. Git pull in dotfiles directory
2. Show what changed
3. Check if `.go4dot.yaml` changed
4. Offer to install new configs
5. Restow configs
6. Update external deps
7. Show migration notes if any

### `g4d list`

Show installed configs.

**Output:**
- Installed configs
- Available but not installed
- Platform-specific (not available on this platform)
- Archived configs

### `g4d reconfigure`

Re-run machine-specific prompts.

**Options:**
- Reconfigure everything
- Reconfigure specific parts (git, paths, etc.)

### `g4d stow`

Manual stow operations.

**Subcommands:**
- `add <config>` - Stow a specific config
- `remove <config>` - Unstow a specific config
- `refresh` - Restow all active configs
- `list` - Show available configs

### `g4d uninstall`

Remove dotfiles.

**Flow:**
1. Confirm action
2. Unstow all configs
3. Optionally remove external deps
4. Optionally remove machine config
5. Remove state file

### `g4d version`

Show version information.

**Output:**
- Version number
- Build time
- Go version
- Platform

---

## Go Project Structure

```
go4dot/
├── cmd/
│   └── g4d/
│       └── main.go                 # Entry point
│
├── internal/                       # Private application code
│   ├── config/
│   │   ├── loader.go              # Load .go4dot.yaml
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
│   ├── installation.md            # How to install go4dot
│   ├── getting-started.md         # Quick start guide
│   ├── config-reference.md        # .go4dot.yaml specification
│   ├── commands.md                # Command reference
│   └── creating-dotfiles.md       # Guide for creating dotfiles
│
├── examples/
│   ├── minimal/                   # Minimal example dotfiles
│   │   ├── git/.gitconfig
│   │   ├── zsh/.zshrc
│   │   └── .go4dot.yaml
│   │
│   └── advanced/                  # Full-featured example
│       ├── git/
│       ├── nvim/
│       ├── tmux/
│       ├── .go4dot.yaml
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
- [x] Test build: `make build && ./g4d version`
- [x] Create basic README.md
- [x] Initialize git and make first commit

**Deliverables:**
- Buildable Go project
- Basic CLI with version command
- Project structure in place
- Can run `make build` and `./g4d version`

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
- [ ] Add `g4d detect` command for testing
- [ ] Write unit tests for detection logic

**Deliverables:**
- Platform detection working on Linux/macOS/WSL
- `g4d detect` command shows platform info
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

**Goal:** Parse `.go4dot.yaml` files.

**Status:** ✅ COMPLETED

**Tasks:**

- [ ] Create config structs in `internal/config/schema.go`
- [ ] Implement YAML loading in `internal/config/loader.go`
- [ ] Implement validation in `internal/config/validator.go`
- [ ] Add `g4d config validate` command
- [ ] Add `g4d config show` command
- [ ] Write tests for loading and validation

**Deliverables:**
- Can load `.go4dot.yaml` files
- Validation working
- `g4d config` commands
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
- [ ] Add `g4d doctor --deps-only` command
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
- Manual stow commands (`g4d stow add/remove`)
- Tests passing

**What you'll learn:**
- Running external commands
- File system operations
- Symlink handling
- Error recovery

---

### Phase 6: External Dependencies (3-4 hours)

**Goal:** Clone external repos (Pure, TPM, NvChad).

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create `internal/deps/external.go`
- [x] Implement git clone operations
- [x] Handle different clone methods (clone vs copy)
- [x] Implement conditional cloning
- [x] Show progress during cloning
- [x] Write tests with mocked git operations
- [x] Add CLI commands (`g4d external status/clone/update/remove`)

**Deliverables:**
- Can clone external dependencies
- Copy method working (for NvChad)
- Progress indicators
- Tests passing
- Platform-conditional cloning
- Dry-run support

**What you'll learn:**
- Git operations from Go
- Conditional logic
- File copying
- Progress UX

---

### Phase 7: Machine-Specific Config (4-5 hours)

**Goal:** Prompt for machine-specific values and generate config files.

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create `internal/machine/prompts.go`
- [x] Create `internal/machine/templates.go`
- [x] Create `internal/machine/git.go` for GPG/SSH/git detection
- [x] Implement GPG key detection
- [x] Implement SSH key detection
- [x] Handle different prompt types (text, password, confirm)
- [x] Implement Go template rendering
- [x] Write tests for prompts and templates
- [x] Add CLI commands (`g4d machine status/configure/show/remove/info`)

**Deliverables:**
- Can prompt for machine-specific config
- Template rendering working
- GPG key detection
- SSH key detection
- Git config detection (user.name, user.email, signingkey)
- Generated config files from templates
- Tests passing

**What you'll learn:**
- Go templates (`text/template`)
- Interactive prompts
- String manipulation
- File writing

---

### Phase 8: Main Install Command (4-5 hours)

**Goal:** Orchestrate the full installation flow.

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create `internal/setup/setup.go`
- [x] Implement full install orchestration
- [x] Add progress indicators with sections
- [x] Handle errors gracefully (continue on non-fatal errors)
- [x] Add flags (--auto, --minimal, --skip-deps, --skip-external, --skip-machine, --skip-stow)
- [x] Add install command to CLI
- [x] Write tests for setup package

**Deliverables:**
- Full `g4d install` command working
- Progress output with sections
- Error aggregation and summary
- Flexible skip options
- Tests passing

**What you'll learn:**
- Orchestrating complex flows
- Error handling strategies
- UX design for CLI
- Integration testing

---

### Phase 9: Doctor Command (3-4 hours)

**Goal:** Health check and troubleshooting.

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create `internal/doctor/check.go`
- [x] Create `internal/doctor/report.go`
- [x] Implement all health checks
- [x] Generate beautiful health report
- [x] Suggest fixes for common issues
- [x] Write tests for checks

**Deliverables:**
- `g4d doctor` command working
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

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Implement `g4d update`
- [x] Implement `g4d list`
- [x] Implement `g4d reconfigure`
- [x] Implement `g4d uninstall`
- [x] Implement state management
- [x] Write tests for each command

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

**Goal:** Generate `.go4dot.yaml` from existing dotfiles.

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create `internal/config/init.go`
- [x] Implement directory scanning
- [x] Implement config type detection
- [x] Create interactive wizard
- [x] Generate YAML with comments
- [x] Write tests for init logic

**Deliverables:**
- `g4d init` command working
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

**Goal:** Make go4dot easy to install.

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Create `scripts/install.sh` (bootstrap)
- [x] Create `scripts/build.sh` (cross-compile)
- [x] Set up GitHub Actions (release.yml)
- [x] Set up GitHub Actions (test.yml)
- [x] Test release process
- [x] Update Makefile with release target

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

**Status:** ✅ COMPLETED

**Tasks:**

- [x] Write main README.md
- [x] Write docs/installation.md
- [x] Write docs/getting-started.md
- [x] Write docs/config-reference.md
- [x] Write docs/commands.md
- [x] Write docs/creating-dotfiles.md
- [x] Create example dotfiles (minimal & advanced)
- [x] Add help text to all commands

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
    // Run g4d install --auto
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
✅ `g4d init` generates valid config  
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

- ✅ **Name:** go4dot
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

- **Repository:** https://github.com/nvandessel/go4dot
- **Issues:** https://github.com/nvandessel/go4dot/issues
- **Releases:** https://github.com/nvandessel/go4dot/releases

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
- **2026-01-03**: Phase 6 completed
  - **Phase 6**: ✅ External dependencies (clone, update, remove, status)
    - Git clone operations with shallow clone (--depth 1)
    - Clone vs copy methods (copy removes .git for owned files)
    - Platform-conditional cloning (os, distro, wsl, architecture)
    - CLI commands: `external status/clone/update/remove`
    - Dry-run support for all operations
  - **Progress**: 43% complete (6/14 phases), 55 tests passing, ~5,200 lines of code
- **2026-01-03**: Phase 7 completed
  - **Phase 7**: ✅ Machine-specific configuration
    - Interactive prompts for machine-specific values
    - Go template rendering for config file generation
    - GPG key detection (list keys, find by email)
    - SSH key detection (ssh-agent keys)
    - Git config detection (user.name, user.email, signingkey)
    - System info gathering (username, hostname, etc.)
    - CLI commands: `machine info/status/configure/show/remove`
  - **Progress**: 50% complete (7/14 phases), 87 tests passing, ~6,500 lines of code
- **2026-01-03**: Phase 8 completed
  - **Phase 8**: ✅ Main install command
    - Full installation orchestration
    - Progress output with sections (deps, configs, external, machine)
    - Error aggregation (continues on non-fatal errors)
    - Flags: --auto, --minimal, --skip-deps, --skip-external, --skip-machine, --skip-stow
    - Post-install message support
    - CLI command: `g4d install [path]`
  - **Progress**: 57% complete (8/14 phases), 94 tests passing, ~7,200 lines of code
- **2026-01-04**: Phase 11 completed
  - **Phase 11**: ✅ Init command
    - Implemented directory scanning to find config folders
    - Created interactive wizard for metadata and config selection
    - Generated well-formatted .go4dot.yaml with comments
    - Added unit tests for scanning and generation logic
    - CLI command: `g4d init [path]`
  - **Progress**: 78% complete (11/14 phases), 121 tests passing, ~9,800 lines of code
- **2026-01-04**: Phase 12 completed
  - **Phase 12**: ✅ Distribution & Release
    - Added `scripts/build.sh` for multi-platform build and packaging
    - Added `scripts/install.sh` for one-line installation (curl | bash)
    - Set up GitHub Actions for CI testing (on PR) and releases (on tag)
    - Updated Makefile with `release` target
  - **Progress**: 85% complete (12/14 phases), 121 tests passing, ~9,900 lines of code
- **2026-01-04**: Phase 13 completed
  - **Phase 13**: ✅ Documentation
    - Created docs directory with full guides (install, getting started, config)
    - Updated main README.md
    - Created minimal and advanced example dotfiles
  - **Progress**: 92% complete (13/14 phases), 121 tests passing, ~10,000+ lines of code

---

**Let's build something awesome! 🐹🚀**
