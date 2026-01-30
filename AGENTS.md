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

### TUI Testing

For TUI components (using Bubble Tea), use the `teatest` framework for headless testing. This ensures that the UI renders correctly and responds to input without needing a physical terminal.

#### Basic teatest Pattern

```go
func TestDashboard_Interaction(t *testing.T) {
    m := NewModel()
    tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

    // Wait for initial render
    teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
        return strings.Contains(string(out), "Expected Content")
    })

    // Send keystrokes
    tm.Send(tea.KeyMsg{Type: tea.KeyDown})

    // Verify changes
    teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
        return strings.Contains(string(out), "New Selection")
    })

    tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

#### Extended teatest Helpers (E2E)

For E2E tests (with `-tags=e2e`), use the extended helpers in `test/e2e/helpers/teatest_extended.go`. These provide a fluent API for testing TUI interactions:

**NewTUITestModel** - Create a test model with helpers:
```go
tm := helpers.NewTUITestModel(t, model, teatest.WithInitialTermSize(100, 40))
```

**WaitForText / WaitForNotText** - Wait for text to appear or disappear:
```go
tm.WaitForText("vim", 2*time.Second)               // Wait for text
tm.WaitForNotText("Loading", 1*time.Second)        // Wait for text to disappear
```

**SendKeys** - Send keyboard input with multiple formats:
```go
tm.SendKeys('?')                                   // Single rune
tm.SendKeys(tea.KeyEsc)                            // Special key
tm.SendKeys("hello")                               // String (types each character)
tm.SendKeysWithDelay(5*time.Millisecond, 'a', 'b') // Custom delay between keys
```

**KeySequence Builder** - Fluent API for complex interactions:
```go
seq := helpers.NewKeySequence().
    Type("vim").                    // Type text
    Down().Up().                    // Arrow keys
    Space().Enter().                // Common keys
    Tab().Esc().                    // Navigation
    Backspace().Delete().           // Editing
    Home().End().                   // Line navigation
    PageUp().PageDown()             // Page navigation

seq.SendTo(tm)                                     // Send with default delay
seq.SendToWithDelay(tm, 5*time.Millisecond)       // Send with custom delay
```

**Complete Example**:
```go
//go:build e2e

func TestDashboard_Navigation(t *testing.T) {
    state := dashboard.State{
        Platform: &platform.Platform{OS: "linux"},
        Configs: []config.ConfigItem{
            {Name: "vim"},
            {Name: "zsh"},
        },
        HasConfig: true,
    }

    model := dashboard.New(state)
    tm := helpers.NewTUITestModel(t, &model, teatest.WithInitialTermSize(100, 40))

    // Wait for initial render
    tm.WaitForText("vim", 2*time.Second)

    // Navigate and select
    helpers.NewKeySequence().
        Down().              // Move to zsh
        Space().             // Select
        Esc().               // Quit
        SendTo(tm)

    tm.WaitFinished(2 * time.Second)
}
```

See `test/e2e/scenarios/dashboard_tui_test.go` for more examples.

## Development Notes

- GNU stow must be installed on the system (not bundled)
- Config files are discovered from: `.`, `~/dotfiles`, `~/.dotfiles`
- Package names may differ across distros - use `MapPackageName()` in `platform/packages.go`
- Error wrapping: use `fmt.Errorf("context: %w", err)`
- State is stored in `~/.config/go4dot/state.json`
- TUI uses Charm libraries: Bubbletea, Huh, Bubbles, Lipgloss

## Sandbox Testing

Creates a safe to use sandbox environment for quick testing.
Launches a Docker or Podman container, so the symlinks can easily and safely be tested and validated.

```bash
make sandbox   # Run Docker/Podman container for isolated testing
```

## AI guidance
- **IMPORTANT:** Read and follow [GO_GUIDELINES.md](./GO_GUIDELINES.md) for all Go code.
- ensure you're making feature branches
- use gh cli when you can
- finish all tasks with make build and make lint prior to commits
- ensure you're writting unit tests
- ensure you're implmenting good Architecture
- focus on SOLID principles

## Issue Tracking with Beads

This repository uses **Beads** for AI-native issue tracking. Issues live in `.beads/` and sync with git.

### Agent Warning: Interactive Commands

DO NOT use bd edit - it opens an interactive editor ($EDITOR) which AI agents cannot use.

Use bd update with flags instead (see Basic Commands below).

### Basic bd Commands

**Listing Issues:**
```bash
bd list                     # List all open issues
bd list --status closed     # List closed issues
bd list --status all        # List all issues
bd list --assigned me       # List issues assigned to you
bd list --tag bug           # Filter by tag
```

**Creating Issues:**
```bash
bd create --title "Issue title" --description "Details"
bd create --title "Bug fix" --description "Fix the thing" --tag bug --tag priority-high
bd create --title "Feature" --design "Design notes" --acceptance "AC criteria"
```

**Viewing Issue Details:**
```bash
bd show <id>                # Show full issue details
```

**Updating Issues (Non-Interactive):**
```bash
bd update <id> --title "new title"
bd update <id> --description "new description"
bd update <id> --design "design notes"
bd update <id> --notes "additional notes"
bd update <id> --acceptance "acceptance criteria"
bd update <id> --status in-progress
bd update <id> --status closed
bd update <id> --assigned username
bd update <id> --tag new-tag
```

**Closing Issues:**
```bash
bd close <id>               # Mark issue as closed
bd update <id> --status closed  # Alternative syntax
```

**Deleting Issues:**
```bash
bd delete <id>              # Permanently delete an issue
```

**Status Transitions:**
- `open` → `in-progress` → `closed`
- Use `--status` flag with bd update to change status

### GitHub Integration

- **Linking Issues:** When creating a Bead based on a GitHub issue, mention the GitHub issue number/URL in the Bead description.
- **Closing Issues:** When opening a PR for a Bead that has a corresponding GitHub issue, use the `Closes #issue-number` notation in the PR description. This ensures the GitHub issue is automatically closed when the PR merges, maintaining sync and traceability.

### Syncing

Beads uses a **dedicated `beads-sync` branch** for issue tracking, separate from code changes on `main`. This keeps bead history clean and focused.

#### Daily Workflow: Committing Beads

When you create, update, or close beads during a session:

```bash
# Work happens on main branch (or feature branches)
# Beads are committed to beads-sync branch

bd sync          # Export database to JSONL (switches to beads-sync worktree)
git add .beads/  # Stage beads files (in beads-sync worktree)
git commit -m "feat: add/update beads for X"  # Commit to beads-sync
git push         # Push beads-sync to remote
```

**IMPORTANT:** The `bd sync` command automatically handles the beads-sync worktree. You don't need to manually checkout branches.

#### Weekly/Regular: Sync Beads to Main

The `beads-sync` branch will diverge from `main` over time. **Periodically (at least weekly or when significant bead work accumulates), merge beads-sync into main via PR**:

```bash
# 1. Ensure beads-sync is pushed
bd sync && git push  # From any branch

# 2. Create PR branch from main
git checkout main
git pull
git checkout -b sync-beads-YYYYMMDD

# 3. Merge beads-sync (creates merge commit)
git merge beads-sync --no-edit

# 4. Push and create PR
git push -u origin sync-beads-YYYYMMDD
gh pr create --title "chore: sync beads from beads-sync branch" \
  --body "Periodic sync of bead changes from beads-sync to keep main up to date with issue tracking."

# 5. After PR merges, return to main
git checkout main
git pull
```

**Why this workflow?**
- **Separation of concerns**: Code changes (main/feature branches) vs. issue tracking (beads-sync)
- **Clean history**: Bead commits don't clutter feature branch history
- **Flexibility**: Multiple agents can work on beads without conflicting with code work
- **Sync point**: Regular merges keep main aware of current project issues

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
