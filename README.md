# go4dot

A powerful, cross-platform CLI tool for managing dotfiles with style.

> [!WARNING]
> This is in active dev, see 🏗️ Development Status for more info.

[![Go Version](https://img.shields.io/github/go-mod/go-version/nvandessel/go4dot)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ Features

- 🔍 **Platform Detection** - Automatically detect OS, distro, and package manager
- 📦 **Dependency Management** - Check for and install required tools
- 🎨 **Interactive Setup** - Beautiful TUI with prompts and progress indicators
- 🔧 **Machine-Specific Config** - Prompt for values that differ per machine
- 🔗 **Stow Management** - Safely symlink configs with conflict detection
- 🌐 **External Dependencies** - Clone plugin managers, themes, etc. from GitHub
- 🏥 **Health Checking** - Doctor command to validate installation
- 🌍 **Universal** - Works with ANY dotfiles repo with a `.go4dot.yaml` config

## 🚀 Quick Start

### Installation

```bash
# Clone your dotfiles repository
git clone https://github.com/yourusername/dotfiles.git ~/dotfiles
cd ~/dotfiles

# Run the bootstrap script (installs go4dot + runs setup)
./install.sh
```

Or install go4dot manually:

```bash
# Using Go
go install github.com/nvandessel/go4dot/cmd/g4d@latest

# Or download from releases
curl -fsSL https://raw.githubusercontent.com/nvandessel/go4dot/main/scripts/install.sh | bash
```

### Usage

```bash
# Install dotfiles interactively
g4d install

# Initialize a new .go4dot.yaml for your existing dotfiles
g4d init

# Check your dotfiles health
g4d doctor

# Update dotfiles and restow
g4d update

# List installed configs
g4d list
```

## 📚 Documentation

- [Installation Guide](docs/installation.md) - Coming soon
- [Getting Started](docs/getting-started.md) - Coming soon
- [Configuration Reference](docs/config-reference.md) - Coming soon
- [Command Reference](docs/commands.md) - Coming soon
- [Creating Your Own Dotfiles](docs/creating-dotfiles.md) - Coming soon

## 🏗️ Development Status

go4dot is currently in active development. See [PLAN.md](PLAN.md) for the complete implementation roadmap.

**Current Status:** Phase 0 - Project Setup ✅

### Building from Source

```bash
# Clone the repository
git clone https://github.com/nvandessel/go4dot.git
cd go4dot

# Build
make build

# Run
./bin/g4d version

# Run tests
make test

# Install locally
make install
```

### Available Make Targets

```bash
make build         # Build for current platform
make build-all     # Build for all platforms
make test          # Run tests
make test-coverage # Run tests with coverage report
make install       # Install to GOPATH/bin
make clean         # Remove build artifacts
make fmt           # Format code
make vet           # Run go vet
make lint          # Run golangci-lint
make help          # Show all targets
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [chezmoi](https://www.chezmoi.io/), [yadm](https://yadm.io/), and [dotbot](https://github.com/anishathalye/dotbot)
- Built with [Cobra](https://github.com/spf13/cobra), [Bubbletea](https://github.com/charmbracelet/bubbletea), [Huh](https://github.com/charmbracelet/huh), and [Lipgloss](https://github.com/charmbracelet/lipgloss)
- Powered by [GNU Stow](https://www.gnu.org/software/stow/)

## 📮 Contact

- **Author:** Nic Van Dessel
- **Repository:** [github.com/nvandessel/go4dot](https://github.com/nvandessel/go4dot)
- **Issues:** [github.com/nvandessel/go4dot/issues](https://github.com/nvandessel/go4dot/issues)

---

**Made with ❤️ and Go**
