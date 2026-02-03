# VHS tape for testing the health panel functionality
# Tests entering health panel, navigation, and exiting
#
# Prerequisites: Binary built at ./bin/g4d, test fixtures in test/e2e/fixtures/
# Note: This tape is designed to run in Docker container via NewDockerTestContainer
# In container: binary is at /usr/local/bin/g4d, fixtures at /home/testuser/fixtures

Set Shell bash
Set FontSize 14
Set Width 1200
Set Height 600
Set Padding 10

# Isolate from user's real environment
Env HOME "/tmp/g4d-e2e-test"
Env XDG_CONFIG_HOME "/tmp/g4d-e2e-test/.config"

# Run g4d in the test fixtures directory
# Note: In container, fixtures are at /home/testuser/fixtures
Type "cd ~/fixtures && g4d"
Enter
Sleep 2s

# Screenshot initial dashboard state
Screenshot "test/e2e/screenshots/health_initial.png"

# Press 'd' to focus health panel (doctor shortcut)
Type "d"
Sleep 500ms
Screenshot "test/e2e/screenshots/health_focused.png"

# Navigate down in health panel
Down
Sleep 300ms
Screenshot "test/e2e/screenshots/health_nav_down.png"

# Navigate up
Up
Sleep 300ms
Screenshot "test/e2e/screenshots/health_nav_up.png"

# Press Enter to get details on a health item
Enter
Sleep 500ms
Screenshot "test/e2e/screenshots/health_details.png"

# Return to configs panel with number key
Type "1"
Sleep 500ms
Screenshot "test/e2e/screenshots/health_return_configs.png"

# Quit
Type "q"
Sleep 500ms

# Capture final output
Output "test/e2e/outputs/health_panel.txt"
