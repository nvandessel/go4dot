# VHS tape for testing confirm dialog overlay
# Tests that the confirm dialog renders correctly when triggered via menu
#
# Prerequisites: Binary built at ./bin/g4d (will be modified to g4d in container)
# Note: This tape is designed to run in Docker container via NewDockerTestContainer

Set Shell bash
Set FontSize 14
Set Width 1200
Set Height 600
Set Padding 10

# Isolate from user's real environment (Env must come after Set commands)
Env HOME "/tmp/g4d-confirm-test"
Env XDG_CONFIG_HOME "/tmp/g4d-confirm-test/.config"

# Run g4d in the test fixtures directory
Type "cd /home/testuser/fixtures && g4d"
Enter
Sleep 2s

# Screenshot the dashboard before opening menu
Screenshot "test/e2e/screenshots/confirm_before_menu.png"

# Open the menu with tab
Type "	"
Sleep 1s

# Screenshot the menu overlay
Screenshot "test/e2e/screenshots/confirm_menu.png"

# Navigate down to "Uninstall go4dot" (3rd item)
Type "j"
Sleep 300ms
Type "j"
Sleep 300ms

# Select uninstall to trigger confirm dialog
Enter
Sleep 1s

# Screenshot the confirm dialog
Screenshot "test/e2e/screenshots/confirm_dialog.png"

# Navigate to select "Yes" button (left key)
Type "h"
Sleep 500ms

# Screenshot with "Yes" selected
Screenshot "test/e2e/screenshots/confirm_dialog_yes.png"

# Press escape to cancel (don't actually uninstall)
Escape
Sleep 1s

# Screenshot after cancelling
Screenshot "test/e2e/screenshots/confirm_cancelled.png"

# Quit
Type "q"
Sleep 500ms

# Capture final output
Output "test/e2e/outputs/confirm_dialog.txt"
