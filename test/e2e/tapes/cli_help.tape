# VHS tape for testing g4d CLI help output
# This is a foundational test to validate the VHS infrastructure

# Set up terminal environment
Set Shell bash
Set FontSize 12
Set Width 1200
Set Height 800

# Show g4d help to validate binary works
Type "./bin/g4d --help"
Enter
Sleep 2s

# Output to temporary file for comparison with golden
Output test/e2e/outputs/cli_help.txt
