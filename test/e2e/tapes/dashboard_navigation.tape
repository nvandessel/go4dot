# VHS tape for testing g4d CLI basic output
# This is a foundational test to validate the VHS infrastructure

# Set up terminal environment
Set Shell bash
Set FontSize 12
Set Width 1200
Set Height 800

# Show g4d help to validate binary works
Type "./bin/g4d --help"
Enter
Sleep 1s

# Output to text file for golden file comparison
Output test/e2e/golden/cli_help.txt

# Also capture a screenshot for visual validation
Screenshot test/e2e/screenshots/cli_help.png
