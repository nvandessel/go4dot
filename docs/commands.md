# Command Reference

## Global Flags

These flags can be used with any command:
- `--non-interactive`: Run without interactive prompts.
- `-y, --yes`: Alias for `--non-interactive`.

Environment variables:
- `GO4DOT_NON_INTERACTIVE=1`: Enable non-interactive mode.
- `CI=true`: Automatically enables non-interactive mode.

## `g4d install`
The main entry point. Orchestrates the full setup process.
- **Usage**: `g4d install [path]`
- **Flags**:
  - `--auto`: Run in non-interactive mode using defaults.
  - `--minimal`: Install only core configs/deps, skip optional ones.
  - `--skip-deps`: Skip system dependency check/install.
  - `--skip-external`: Skip cloning external dependencies.
  - `--skip-machine`: Skip machine configuration prompts.
  - `--skip-stow`: Skip stowing dotfiles.

## `g4d init`
Bootstrap a new configuration from existing dotfiles.
- **Usage**: `g4d init [path]`
- **Description**: Scans the directory for config folders and interacts with you to generate a `.go4dot.yaml`.

## `g4d doctor`
Check the health of your installation.
- **Usage**: `g4d doctor [path]`
- **Flags**:
  - `-v, --verbose`: Show detailed output including fix suggestions.
- **Checks**:
  - System dependencies
  - Broken symlinks
  - Missing external dependencies
  - Machine config validity

## `g4d update`
Update dotfiles and external dependencies.
- **Usage**: `g4d update [path]`
- **Flags**:
  - `--external`: Also update external dependencies (plugins, themes).
  - `--skip-restow`: Skip restowing configs after pull.
- **Actions**:
  - `git pull --rebase` in dotfiles repo
  - Show what changed
  - Restow configs to apply changes
  - Update external git repos (if `--external` is set)

## `g4d list`
List all available and installed configurations.
- **Usage**: `g4d list`
- **Flags**:
  - `-a, --all`: Show all details including archived/hidden.

## `g4d reconfigure`
Re-run machine-specific configuration prompts.
- **Usage**: `g4d reconfigure [id]`
- **Description**: Useful if you need to change a value (like git email) without reinstalling everything.

## `g4d uninstall`
Remove symlinks and clean up.
- **Usage**: `g4d uninstall`
- **Flags**:
  - `-f, --force`: Skip confirmation.
- **Description**: Unstows all configs. Does **not** delete your actual dotfiles files, only the symlinks.

## `g4d detect`
Show platform information.
- **Usage**: `g4d detect`
- **Output**: OS, Distro, Package Manager, etc.

## `g4d stow`
Manual stow operations.
- `g4d stow add <config>`: Stow a specific config group.
- `g4d stow remove <config>`: Unstow a specific config group.
- `g4d stow refresh`: Restow all active configs.

## `g4d external`
Manage external dependencies manually.
- `g4d external status`: Show status of external repos.
- `g4d external clone [id]`: Clone specific repo.
- `g4d external update [id]`: Update specific repo.
- `g4d external remove <id>`: Remove specific repo.

## `g4d machine`
Manage machine configuration manually.
- `g4d machine info`: Show system information (git config, GPG/SSH keys).
- `g4d machine status [path]`: Show status of machine configs.
- `g4d machine configure [id] [path]`: Run prompts for specific config.
  - `--defaults`: Use default values without prompting.
  - `--overwrite`: Overwrite existing configuration files.
- `g4d machine show <id> [path]`: Preview generated config.
- `g4d machine remove <id> [path]`: Remove a generated config file.

## `g4d version`
Display version information.
- **Usage**: `g4d version`
- **Output**: Version, build time, and Go version.
