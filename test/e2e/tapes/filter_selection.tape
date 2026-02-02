# VHS tape for testing filter and selection functionality
# Tests filtering configs and multi-select operations
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

# Screenshot initial state
Screenshot test/e2e/screenshots/filter_initial.png

# Enter filter mode with '/'
Type "/"
Sleep 300ms
Screenshot test/e2e/screenshots/filter_mode.png

# Type filter text
Type "vim"
Sleep 300ms
Screenshot test/e2e/screenshots/filter_vim.png

# Exit filter mode with Escape
Escape
Sleep 300ms
Screenshot test/e2e/screenshots/filter_exit.png

# Clear filter and select all with 'A'
Type "A"
Sleep 300ms
Screenshot test/e2e/screenshots/filter_select_all.png

# Toggle selection with space
Space
Sleep 300ms
Screenshot test/e2e/screenshots/filter_toggle.png

# Navigate down and toggle
Down
Sleep 200ms
Space
Sleep 300ms
Screenshot test/e2e/screenshots/filter_multi_select.png

# Press '?' for help
Type "?"
Sleep 500ms
Screenshot test/e2e/screenshots/filter_help.png

# Close help
Type "?"
Sleep 300ms

# Quit
Type "q"
Sleep 500ms

# Capture final output
Output test/e2e/outputs/filter_selection.txt
