This file provides guidance for working with code in this repository.

## Build Commands

```bash
make build          # Build binary to ./bin/g4d
make test           # Run tests with race detection and coverage
make lint           # Run golangci-lint (install: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin)
make fmt            # Format code with go fmt and gofmt -s
make vet            # Run go vet static analysis
make test-coverage  # Generate coverage.html report
make package        # Build and package release artifacts
make release        # Tag and push a new version (interactive)
make sandbox        # Build and deploy a local container for testing
```

### Running a Single Test

```bash
go test -v -run TestFunctionName ./internal/package
# Example: go test -v -run TestDetectLinuxDistro ./internal/platform
```

## Architecture

go4dot is a CLI tool for managing dotfiles using GNU Stow, with platform detection and dependency management.

### Package Structure

```
cmd/g4d/                  # CLI entry point with Cobra commands
  main.go                 # Root command setup
  install.go, init.go...  # Subcommand implementations
internal/
  platform/               # OS/distro detection + package manager abstraction
    detect.go             # Platform struct with Detect() method
    packages.go           # PackageManager interface + GetPackageManager factory
    packages_{dnf,apt,brew,pacman,yum}.go  # Strategy implementations
  config/                 # YAML config loading from .go4dot.yaml
    schema.go             # Config, Dependencies, ConfigItem structs
    loader.go             # LoadFromFile, Discover functions
    validator.go          # Validation logic
    init.go               # Directory scanning for g4d init
  deps/                   # Dependency checking/installation
    check.go              # CheckDependencies using platform detection
    install.go            # InstallDependencies using package managers
    external.go           # Git clone external deps (plugins, themes)
  stow/                   # GNU stow wrapper for symlink management
    manager.go            # Stow, Unstow, RestowConfigs functions
    drift.go              # Detect symlink drift/changes
  machine/                # Machine-specific config generation
    prompts.go            # Interactive prompts for machine values
    templates.go          # Go template rendering
    git.go                # GPG/SSH key detection
  doctor/                 # Health checking
    check.go              # Run health checks
    report.go             # Generate health report
  state/                  # Installation state tracking
    state.go              # Load/Save ~/.config/gopherdot/state.json
  setup/                  # Install orchestration
    setup.go              # Full install flow coordination
  ui/                     # TUI components (Charm libraries)
    styles.go             # Lipgloss styles and colors
    spinner.go, menu.go   # Interactive components
    banner.go             # ASCII art banner
  version/                # Version management
    check.go              # Version checking logic
```

### Key Design Patterns

**PackageManager interface** (strategy pattern) - each package manager (dnf, apt, brew, pacman) implements:
```go
type PackageManager interface {
    Name() string
    IsAvailable() bool
    Install(packages ...string) error
    IsInstalled(pkg string) bool
    Update() error
    NeedsSudo() bool
}
```

**Config loading flow**: Discover config file -> Parse YAML -> Validate -> Return typed Config struct

### CLI Command Groups

- `g4d install [path]` - Full interactive installation (main command)
- `g4d init [path]` - Generate .go4dot.yaml from existing dotfiles
- `g4d detect` - Show platform info
- `g4d config {validate,show}` - Config operations
- `g4d deps {check,install}` - Dependency management
- `g4d stow {add,remove,refresh}` - Symlink management
- `g4d external {status,clone,update,remove}` - External dependency management
- `g4d machine {info,status,configure,show,remove}` - Machine-specific config
- `g4d doctor` - Health check with fix suggestions
- `g4d list` - Show installed/available configs
- `g4d update` - Pull latest and restow
- `g4d reconfigure` - Re-run machine config prompts
- `g4d uninstall` - Remove symlinks and state

## Testing Patterns

Tests use table-driven patterns. Example:
```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{...}
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {...})
    }
}
```

## Development Notes

- GNU stow must be installed on the system (not bundled)
- Config files are discovered from: `.`, `~/dotfiles`, `~/.dotfiles`
- Package names may differ across distros - use `MapPackageName()` in `platform/packages.go`
- Error wrapping: use `fmt.Errorf("context: %w", err)`
- State is stored in `~/.config/go4dot/state.json`
- TUI uses Charm libraries: Bubbletea, Huh, Bubbles, Lipgloss

## Sandbox Testing

Creates a safe to use sandbox environment for quick testing.
Launches a Docker or Podman cotainer, so the symlinks can easily and safely be tested and validated.

```bash
make sandbox   # Run Docker/Podman container for isolated testing
```

## AI guidance
- ensure you're making feature branches
- use gh cli when you can
- finish all tasks with make build and make lint prior to commits
- ensure you're writting unit tests
- ensure you're implmenting good Architecture
- focus on SOLID principles

## Issue Tracking with Beads

This repository uses **Beads** for AI-native issue tracking. Issues live in `.beads/` and sync with git.

### Using bv (Beads Viewer) - Required for AI Agents

`bv` is a graph-aware triage engine optimized for AI agents. **Always use `--robot-*` flags** - bare `bv` launches an interactive TUI that blocks your session.

**Start with triage:**
```bash
bv --robot-triage              # THE MEGA-COMMAND: ranked recommendations, quick wins, blockers
bv --robot-next                # Minimal: single top pick + claim command
```

**Planning & analysis:**
```bash
bv --robot-plan                # Parallel execution tracks with unblocks lists
bv --robot-insights            # Full graph metrics: PageRank, cycles, critical path
bv --robot-alerts              # Stale issues, blocking cascades, priority mismatches
```

**Scoping:**
```bash
bv --robot-plan --label backend           # Scope to label's subgraph
bv --recipe actionable --robot-plan       # Pre-filter: ready to work (no blockers)
```

### bd Commands (for mutations)

```bash
bd create "Issue title"                   # Create a new issue
bd show <id>                              # Show issue details
bd update <id> --status in_progress       # Update status
bd close <id>                             # Close an issue
bd sync                                   # Export changes for git commit
```

### Workflow

1. **Triage first:** `bv --robot-triage` to understand priorities and what to work on
2. **Claim work:** Use the command from `bv --robot-next` output
3. **Update status:** `bd update <id> --status in_progress`
4. **Close when done:** `bd close <id>`
5. **Sync before commit:** `bd sync`

### Syncing

Beads uses a `beads-sync` branch for synchronization. When ending a session:
```bash
bd sync          # Export database to JSONL
git add .beads/  # Stage beads files
git commit       # Commit with your changes
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
