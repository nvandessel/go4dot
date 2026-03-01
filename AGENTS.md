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

# E2E and Visual Testing (IMPORTANT for TUI work)
make e2e-visual         # Run VHS visual tests (screenshots in test/e2e/screenshots/)
make e2e-visual-update  # Update golden files for visual tests
make e2e-test           # Run fast E2E TUI tests (teatest)
make e2e-all            # Run all E2E tests
make validate           # Full validation (build, lint, test, e2e)
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

### VHS Visual Testing (MANDATORY for TUI changes)

**IMPORTANT:** When implementing TUI features, you MUST verify your work using VHS visual tests. This provides screenshot evidence that the feature works correctly.

VHS is a terminal recording tool that captures screenshots and GIFs of terminal sessions. Tests run in isolated Docker containers.

#### Running VHS Tests

```bash
# Run all VHS visual tests
go test -v -tags=e2e ./test/e2e/scenarios/...

# Run specific VHS test
go test -v -tags=e2e -run "TestVHS_ConflictResolution" ./test/e2e/scenarios/...

# Update golden files (run when expected output changes)
UPDATE_GOLDEN=1 go test -v -tags=e2e -run "TestVHS_" ./test/e2e/scenarios/...
```

#### Creating VHS Tapes

VHS tapes are script files in `test/e2e/tapes/`. **Critical settings:**

```tape
# IMPORTANT: Width/Height are in PIXELS, not characters!
Set Shell bash
Set FontSize 14
Set Width 1200      # Minimum 120 pixels required
Set Height 600      # Minimum 120 pixels required
Set Padding 10

# Environment variables (must come AFTER Set commands)
Env HOME "/tmp/test-env"
Env XDG_CONFIG_HOME "/tmp/test-env/.config"

# Commands and interactions
Type "g4d"
Enter
Sleep 2s

# Screenshots are saved to test/e2e/screenshots/
Screenshot "test/e2e/screenshots/feature_name.png"

# Output captures terminal text for golden file comparison
Output "test/e2e/outputs/feature_name.txt"
```

#### Adding a New VHS Test

1. Create tape file in `test/e2e/tapes/your_feature.tape`
2. Add test function in `test/e2e/scenarios/vhs_visual_test.go`:
```go
func TestVHS_YourFeature(t *testing.T) {
    runVHSTest(t, vhsTestCase{
        name:       "your feature",
        tapePath:   "test/e2e/tapes/your_feature.tape",
        outputPath: "test/e2e/outputs/your_feature.txt",
        goldenPath: "test/e2e/golden/your_feature.txt",
    })
}
```
3. Run with `UPDATE_GOLDEN=1` to create initial golden file
4. **View screenshots** to verify the feature works correctly

#### Verifying TUI Work with Screenshots

After running VHS tests, screenshots are saved to `test/e2e/screenshots/`. **You MUST view these screenshots** to verify:
- UI renders correctly
- Modals/dialogs appear as expected
- User interactions produce correct results
- No visual corruption or layout issues

```bash
# List generated screenshots
ls -la test/e2e/screenshots/

# Screenshots can be viewed with any image viewer
# They are PNG files at 1200x600 resolution
```

#### VHS Technical Notes

- **Debian base image**: VHS containers use `debian:bookworm-slim` (Ubuntu's chromium requires snap which doesn't work in Docker)
- **Pixel dimensions**: Width/Height MUST be at least 120x120 pixels. Use 1200x600 for readable output.
- **Screenshot paths**: Are automatically redirected to container paths and copied back
- **Container runtime**: Supports both Docker and Podman (auto-detected)

#### Test Verification Workflow

When implementing TUI features:

1. Write the feature code
2. Create or update VHS tape to test the feature
3. Run VHS test: `UPDATE_GOLDEN=1 go test -v -tags=e2e -run "TestVHS_YourFeature" ./test/e2e/scenarios/...`
4. **View the screenshots** to verify the feature works
5. If screenshots look wrong, fix the code and re-run
6. Only commit when screenshots confirm correct behavior

This visual verification is critical because:
- Unit tests can pass while the UI is broken
- TUI rendering issues are hard to detect programmatically
- Screenshots provide evidence the feature actually works

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
- ensure you're writing unit tests
- ensure you're implementing good Architecture
- focus on SOLID principles

### TUI Development (CRITICAL)
- **ALWAYS verify TUI changes with VHS visual tests** - Unit tests are not sufficient for TUI work
- Create/update VHS tapes for any new TUI feature or modal
- **VIEW THE SCREENSHOTS** after running tests to confirm the feature works
- Do not consider TUI work complete until screenshots confirm correct rendering
- See "VHS Visual Testing" section above for detailed instructions

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

<!-- BEGIN BEADS INTEGRATION -->
## Issue Tracking with bd (beads)

**IMPORTANT**: This project uses **bd (beads)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why bd?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: Auto-syncs to JSONL for version control
- Agent-optimized: JSON output, ready work detection, discovered-from links
- Prevents duplicate tracking systems and confusion

### Quick Start

**Check for ready work:**

```bash
bd ready --json
```

**Create new issues:**

```bash
bd create "Issue title" --description="Detailed context" -t bug|feature|task -p 0-4 --json
bd create "Issue title" --description="What this issue is about" -p 1 --deps discovered-from:bd-123 --json
```

**Claim and update:**

```bash
bd update bd-42 --status in_progress --json
bd update bd-42 --priority 1 --json
```

**Complete work:**

```bash
bd close bd-42 --reason "Completed" --json
```

### Issue Types

- `bug` - Something broken
- `feature` - New functionality
- `task` - Work item (tests, docs, refactoring)
- `epic` - Large feature with subtasks
- `chore` - Maintenance (dependencies, tooling)

### Priorities

- `0` - Critical (security, data loss, broken builds)
- `1` - High (major features, important bugs)
- `2` - Medium (default, nice-to-have)
- `3` - Low (polish, optimization)
- `4` - Backlog (future ideas)

### Workflow for AI Agents

1. **Check ready work**: `bd ready` shows unblocked issues
2. **Claim your task**: `bd update <id> --status in_progress`
3. **Work on it**: Implement, test, document
4. **Discover new work?** Create linked issue:
   - `bd create "Found bug" --description="Details about what was found" -p 1 --deps discovered-from:<parent-id>`
5. **Complete**: `bd close <id> --reason "Done"`

### Auto-Sync

bd automatically syncs with git:

- Exports to `.beads/issues.jsonl` after changes (5s debounce)
- Imports from JSONL when newer (e.g., after `git pull`)
- No manual export/import needed!

### Important Rules

- ✅ Use bd for ALL task tracking
- ✅ Always use `--json` flag for programmatic use
- ✅ Link discovered work with `discovered-from` dependencies
- ✅ Check `bd ready` before asking "what should I work on?"
- ❌ Do NOT create markdown TODO lists
- ❌ Do NOT use external issue trackers
- ❌ Do NOT duplicate tracking systems

For more details, see README.md and docs/QUICKSTART.md.

<!-- END BEADS INTEGRATION -->
