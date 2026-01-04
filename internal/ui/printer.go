package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Success prints a success message (green tick)
func Success(format string, a ...interface{}) {
	icon := SuccessStyle.Render("✓")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", icon, msg)
}

// Error prints an error message (red cross)
func Error(format string, a ...interface{}) {
	icon := ErrorStyle.Render("✖")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", icon, msg)
}

// Warning prints a warning message (yellow triangle)
func Warning(format string, a ...interface{}) {
	icon := lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render("⚠")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", icon, msg)
}

// Info prints an informational message (blue i)
func Info(format string, a ...interface{}) {
	icon := lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render("ℹ")
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s %s\n", icon, msg)
}

// Section prints a section header
func Section(title string) {
	fmt.Println()
	fmt.Println(TitleStyle.Render(title))
}
