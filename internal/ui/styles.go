package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#7D56F4") // Purple
	SecondaryColor = lipgloss.Color("#04B575") // Green
	ErrorColor     = lipgloss.Color("#FF0000") // Red
	WarningColor   = lipgloss.Color("#FFCC00") // Yellow
	SubtleColor    = lipgloss.Color("#626262") // Gray

	// Text Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	TextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	SubtleStyle = lipgloss.NewStyle().
			Foreground(SubtleColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	// Box Styles
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2)

	// List Styles
	ItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(PrimaryColor).
				Bold(true)
)
