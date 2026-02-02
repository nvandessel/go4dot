# VHS tape for testing the onboarding flow
# Tests the no-config view and starting the onboarding wizard
#
# Prerequisites: Binary built at ./bin/g4d (will be modified to g4d in container)
# Note: This tape is designed to run in Docker container via NewDockerTestContainer

Set Shell bash
Set FontSize 14
Set Width 120
Set Height 40
Set Padding 10

# Create an isolated test environment with NO config
Set Env HOME /tmp/g4d-onboarding-test
Set Env XDG_CONFIG_HOME /tmp/g4d-onboarding-test/.config

# Create a clean directory for testing (no .go4dot.yaml)
Type "mkdir -p /tmp/g4d-onboarding-test/dotfiles && cd /tmp/g4d-onboarding-test/dotfiles"
Enter
Sleep 500ms

# Run g4d - should show no-config view
# Note: ./bin/g4d is replaced with 'g4d' when running in container
Type "./bin/g4d"
Enter
Sleep 2s

# Screenshot the no-config view
Screenshot test/e2e/screenshots/onboarding_noconfig.png

# Press 'i' to start onboarding wizard
Type "i"
Sleep 1s

# Screenshot the onboarding wizard (step 1)
Screenshot test/e2e/screenshots/onboarding_wizard.png

# Cancel with Esc to return to no-config view
Escape
Sleep 500ms
Screenshot test/e2e/screenshots/onboarding_cancelled.png

# Quit
Escape
Sleep 500ms

# Capture final output
Output test/e2e/outputs/onboarding_flow.txt
