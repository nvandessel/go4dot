# Configuration Reference (.go4dot.yaml)

The `.go4dot.yaml` file is the heart of go4dot. It defines your dependencies, configurations, and setup logic.

## Structure Overview

```yaml
schema_version: "1.0"

metadata:
  # Project information
  name: "My Dotfiles"
  author: "Jane Doe"
  ...

dependencies:
  # System packages to install
  critical: [...]
  core: [...]

configs:
  # Dotfile groups to stow
  core: [...]
  optional: [...]

external:
  # External repos to clone (plugins, themes)
  ...

machine_config:
  # Prompts and templates for machine-specific files
  ...

machines:
  # Per-machine profiles for multi-machine setups
  ...

archived:
  # Old configs kept for documentation
  ...

post_install: |
  # Message shown after successful install
```

## detailed Reference

### Metadata

Basic information about your dotfiles.

```yaml
metadata:
  name: "Nic's Dotfiles"
  author: "Nic Van Dessel"
  repository: "https://github.com/nvandessel/dotfiles"
  description: "My personal development environment"
  version: "1.0.0"
```

### Dependencies

System packages that need to be installed via the OS package manager (dnf, apt, brew).

- **critical**: Must be installed for the setup to proceed (e.g., git, stow).
- **core**: Recommended packages for a standard setup.
- **optional**: Nice-to-have tools.

**Format:**
Can be a simple string (package name) or an object map for platform differences.

```yaml
dependencies:
  critical:
    - git
    - stow
  
  core:
    # Simple string (assumes same name on all package managers)
    - zsh
    - tmux
    
    # Object map for different names
    - name: neovim
      binary: nvim        # Command to check availability
      package:
        dnf: neovim
        apt: neovim
        brew: neovim
        pacman: neovim
```

### Configs

Groups of dotfiles to be managed by GNU Stow.

- **core**: Installed by default.
- **optional**: User selects which ones to install during setup.

```yaml
configs:
  core:
    - name: git
      path: git               # Directory name in your repo
      description: Git config
      platforms: [linux, macos]
      requires_machine_config: true  # Wait for machine config before stowing?

  optional:
    - name: i3
      path: i3
      description: i3 Window Manager
      platforms: [linux]      # Only show on Linux
      depends_on: [xorg]      # informational dependency

    - name: kde
      path: kde
      description: KDE Plasma settings
      condition:              # Fine-grained conditions (more flexible than platforms)
        hostname: fedora-workstation
```

**Condition vs Platforms:** The `platforms` field is a simple OS filter. The `condition` field supports all condition keys (os, distro, hostname, arch, wsl, package_manager) and can be combined. Both are checked if present.

### Dependencies (Conditional)

Dependencies can have conditions to only install on specific platforms or machines:

```yaml
dependencies:
  core:
    - name: plasma-desktop
      condition:
        distro: fedora
    - name: hyprland
      condition:
        hostname: cachyos-laptop
```

### External

External repositories to clone (e.g., plugin managers, themes, zsh plugins).

```yaml
external:
  - name: Pure Prompt
    id: pure
    url: https://github.com/sindresorhus/pure.git
    destination: ~/.zsh/pure
    method: clone             # "clone" (default) or "copy"
    merge_strategy: overwrite # "overwrite" (default) or "keep_existing"
    condition:                # Optional conditions
      os: linux
      distro: fedora
      hostname: my-laptop     # Machine-specific
      wsl: true
      architecture: amd64
```

**Fields:**
- `name`: Display name for the dependency.
- `id`: Unique identifier used in commands.
- `url`: Git repository URL.
- `destination`: Where to clone/copy (supports `~` expansion).
- `method`: `clone` (default, keeps `.git`) or `copy` (removes `.git` for owned files).
- `merge_strategy`: `overwrite` (default) replaces existing, `keep_existing` skips if present.
- `condition`: Optional platform conditions (all must match if specified).

### Machine Config

Prompts for values that differ between machines (e.g., Work vs Personal) and generates config files from templates.

```yaml
machine_config:
  - id: git
    description: Git user configuration
    destination: ~/.gitconfig.local
    prompts:
      - id: user_name
        prompt: Full name for git commits
        type: text            # text, confirm, or select
        required: true
        default: ""           # Optional default value
      - id: user_email
        prompt: Email for git commits
        type: text
        required: true
    template: |
      [user]
          name = {{ .user_name }}
          email = {{ .user_email }}
```

**Prompt Types:**
- `text`: Free-form text input (default).
- `confirm`: Yes/no boolean prompt.
- `select`: Selection from predefined options (falls back to text input).

### Machines

Define per-machine profiles for multi-machine dotfiles setups. Each profile matches by hostname and can override which configs to install and provide default values for machine_config prompts.

```yaml
machines:
  - name: Fedora Workstation        # Human-readable name
    hostname: fedora-workstation    # Matches os.Hostname()
    exclude_configs: [hyprland]     # Don't install these configs
    defaults:                       # Default values for machine_config prompts
      user_email: work@example.com

  - name: CachyOS Laptop
    hostname: cachyos-laptop
    exclude_configs: [kde]
    defaults:
      user_email: personal@example.com

  - name: MacBook
    hostname: macbook-pro
    include_configs: [git, tmux, nvim]  # Only install these configs
    defaults:
      user_email: work@example.com
```

**Fields:**
- `name`: Human-readable machine name (shown during install).
- `hostname`: Machine hostname to match. Supports comma-separated values for multiple hostnames.
- `include_configs`: If set, only these configs are installed. If empty, all configs are included.
- `exclude_configs`: These configs are never installed on this machine.
- `defaults`: Key-value map of default values for machine_config prompts. Overrides auto-detected defaults but still allows user to change interactively.

**Condition keys** (used in `condition` maps on configs, dependencies, and external deps):
- `os` / `platform`: linux, darwin, windows
- `distro`: fedora, ubuntu, cachyos, arch, etc.
- `hostname`: Machine hostname (supports comma-separated list)
- `arch` / `architecture`: amd64, arm64, etc.
- `package_manager`: dnf, apt, brew, pacman, etc.
- `wsl`: true, false

### Post Install

Optional message displayed after successful installation.

```yaml
post_install: |
  Installation complete!

  Don't forget to:
  - Source your shell config: source ~/.zshrc
  - Install your preferred fonts
```

### Archived

Configs that are no longer actively installed but kept for documentation. These won't appear in the install wizard.

```yaml
archived:
  - name: old-vim
    path: vim
    description: Legacy vim config (replaced by nvim)
```

## Example File

See `examples/minimal/.go4dot.yaml` or `examples/advanced/.go4dot.yaml` in the repository for complete examples.
