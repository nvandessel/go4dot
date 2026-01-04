# Manual Test Plan

Since this is a dotfiles manager, it's best to test it in a safe environment first.

## 1. Setup Sandbox

Create a directory to simulate your home folder.

```bash
mkdir -p /tmp/g4d-sandbox/home
mkdir -p /tmp/g4d-sandbox/dotfiles
export HOME=/tmp/g4d-sandbox/home
```

## 2. Initialize

Copy your dotfiles (or a subset) to the sandbox.

```bash
cp -r ~/dotfiles/zsh /tmp/g4d-sandbox/dotfiles/
# ... copy others
```

Run init:

```bash
cd /tmp/g4d-sandbox/dotfiles
g4d init
```

Verify `.go4dot.yaml` is created.

## 3. Install (Dry Run logic)

Currently, go4dot modifies files directly. To test safely:

1. Backup your current configs if running on real machine.
2. Use `g4d install --minimal` first.

## 4. UI Testing

Run `g4d` without arguments to check the interactive dashboard.

```bash
g4d
```

Verify that:
- The banner is displayed.
- The menu lists Install, Update, Doctor, List, Init, and Quit.
- Navigation works with arrow keys.
- You can select an item with Enter.

Also check specific commands for output formatting:

```bash
g4d doctor
g4d list --all
g4d detect
```

Verify that they use the new styled output (colors, icons, sections).
