# VHS tape for testing dashboard panel navigation
# Tests that Tab/number keys switch between panels correctly
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

# Screenshot initial state (Configs panel focused)
Screenshot test/e2e/screenshots/dashboard_nav_initial.png

# Press Tab to cycle to next panel
Tab
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_tab1.png

# Press Tab again
Tab
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_tab2.png

# Press Tab to continue cycling
Tab
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_tab3.png

# Press 1 to jump directly to Configs panel
Type "1"
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_panel1.png

# Press 2 to jump to Details panel
Type "2"
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_panel2.png

# Press 3 to jump to Health panel
Type "3"
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_panel3.png

# Press 'd' shortcut to focus Health panel (doctor)
Type "d"
Sleep 500ms
Screenshot test/e2e/screenshots/dashboard_nav_doctor.png

# Quit the dashboard
Type "q"
Sleep 500ms

# Capture final output
Output test/e2e/outputs/dashboard_navigation.txt
