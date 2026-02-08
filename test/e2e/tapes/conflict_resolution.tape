# VHS tape for testing conflict resolution modal
# Tests that when conflicts exist during install, the modal appears
#
# Prerequisites: Binary built at ./bin/g4d (will be modified to g4d in container)
# Note: This tape is designed to run in Docker container via NewDockerTestContainer

Set Shell bash
Set FontSize 14
Set Width 1200
Set Height 600
Set Padding 10

# Create an isolated test environment (Env must come after Set commands)
Env HOME "/tmp/g4d-conflict-test"
Env XDG_CONFIG_HOME "/tmp/g4d-conflict-test/.config"

# Set up test environment with CONFLICTING files
Type "mkdir -p /tmp/g4d-conflict-test && cd /tmp/g4d-conflict-test"
Enter
Sleep 300ms

# Create an existing .zshrc in home (this will conflict)
Type "echo 'existing zshrc' > /tmp/g4d-conflict-test/.zshrc"
Enter
Sleep 300ms

# Create dotfiles directory with a zsh config
Type "mkdir -p /tmp/g4d-conflict-test/dotfiles/zsh"
Enter
Sleep 300ms

Type "echo 'dotfiles zshrc' > /tmp/g4d-conflict-test/dotfiles/zsh/.zshrc"
Enter
Sleep 300ms

# Create a minimal .go4dot.yaml config
Type "cat > /tmp/g4d-conflict-test/dotfiles/.go4dot.yaml << 'EOF'"
Enter
Type "schema_version: '1.0'"
Enter
Type "metadata:"
Enter
Type "  name: test-dotfiles"
Enter
Type "configs:"
Enter
Type "  core:"
Enter
Type "    - name: zsh"
Enter
Type "      path: zsh"
Enter
Type "EOF"
Enter
Sleep 300ms

# Navigate to dotfiles directory and run g4d
Type "cd /tmp/g4d-conflict-test/dotfiles && g4d"
Enter
Sleep 2s

# Screenshot the dashboard before triggering install
Screenshot "test/e2e/screenshots/conflict_before_install.png"

# Show help overlay to validate overlay rendering
Type "?"
Sleep 1s
Screenshot "test/e2e/screenshots/conflict_help_overlay.png"
Type "?"
Sleep 500ms

# Press 'i' to trigger install (should show conflict modal)
Type "i"
Sleep 2s

# Screenshot the conflict resolution modal
Screenshot "test/e2e/screenshots/conflict_modal.png"

# Press 'b' to select backup and resolve
Type "b"
Sleep 2s

# Screenshot after resolution (should show install running)
Screenshot "test/e2e/screenshots/conflict_after_resolve.png"

# Wait for operation to complete
Sleep 3s

# Final screenshot
Screenshot "test/e2e/screenshots/conflict_complete.png"

# Quit
Type "q"
Sleep 500ms

# Capture final output
Output "test/e2e/outputs/conflict_resolution.txt"
