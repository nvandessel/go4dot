# go4dot 🐹

A powerful, cross-platform CLI tool for managing dotfiles with style.

[![Go Version](https://img.shields.io/github/go-mod/go-version/nvandessel/go4dot)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

[![codecov](https://codecov.io/github/nvandessel/go4dot/graph/badge.svg?token=6M7NX2424Q)](https://codecov.io/github/nvandessel/go4dot)

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

**One-line install (Linux/macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/nvandessel/go4dot/main/scripts/install.sh | bash
```

See the [Installation Guide](docs/installation.md) for other methods.

### Usage

1. **Clone your dotfiles:**
   ```bash
   git clone https://github.com/yourusername/dotfiles.git ~/dotfiles
   cd ~/dotfiles
   ```

2. **Install:**
   ```bash
   g4d install
   ```

### Creating New Dotfiles?

```bash
cd ~/my-dotfiles
g4d init
```

## 📚 Documentation

- [Installation Guide](docs/installation.md)
- [Getting Started](docs/getting-started.md)
- [Configuration Reference](docs/config-reference.md)
- [Command Reference](docs/commands.md)
- [Creating Your Own Dotfiles](docs/creating-dotfiles.md)

## 🏗️ Building from Source

```bash
git clone https://github.com/nvandessel/go4dot.git
cd go4dot
make build
./bin/g4d version
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Made with ❤️ and Go**
