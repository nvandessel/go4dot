# VHS tape for testing dashboard startup with valid config
# This tests that the dashboard renders properly on initial load
#
# Prerequisites: Binary built at ./bin/g4d, test fixtures in test/e2e/fixtures/
# Note: This tape is designed to run in Docker container via NewDockerTestContainer
# In container: binary is at /usr/local/bin/g4d, fixtures at /home/testuser/fixtures

Set Shell bash
Set FontSize 14
Set Width 120
Set Height 40
Set Padding 10

# Isolate from user's real environment
Set Env HOME /tmp/g4d-e2e-test
Set Env XDG_CONFIG_HOME /tmp/g4d-e2e-test/.config

# Run g4d in the test fixtures directory (has valid .go4dot.yaml)
# Note: In container, ./bin/g4d becomes 'g4d' and fixtures are at /home/testuser/fixtures
Type "cd ~/fixtures && g4d"
Enter
Sleep 2s

# Take screenshot of initial dashboard state
Screenshot test/e2e/screenshots/dashboard_startup.png

# Wait a bit more to ensure full render
Sleep 500ms

# Quit the dashboard
Type "q"
Sleep 500ms

# Capture final output
Output test/e2e/outputs/dashboard_startup.txt
