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
