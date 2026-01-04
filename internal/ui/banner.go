package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var banner = `
                  __ __    __      __ 
   ____   ____   / // /___/ /___  / /_
  / __ \ / __ \ / // // __  // __ \/ __/
 / /_/ // /_/ // // // /_/ // /_/ / /_  
 \__, / \____//_//_/ \__,_/ \____/\__/  
/____/                                  
`

// PrintBanner prints the ASCII art banner
func PrintBanner(version string) {
	fmt.Println(lipgloss.NewStyle().
		Foreground(PrimaryColor).
		Render(banner))

	fmt.Println(lipgloss.NewStyle().
		Foreground(SubtleColor).
		Render("           v" + version))
	fmt.Println()
}
