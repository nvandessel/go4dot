package ui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors — Catppuccin Mocha palette
	PrimaryColor   = lipgloss.Color("#b4befe") // Lavender (Catppuccin Mocha)
	SecondaryColor = lipgloss.Color("#a6e3a1") // Green (Catppuccin Mocha)
	ErrorColor     = lipgloss.Color("#f38ba8") // Red (Catppuccin Mocha)
	WarningColor   = lipgloss.Color("#f9e2af") // Yellow (Catppuccin Mocha)
	SubtleColor    = lipgloss.Color("#9399b2") // Overlay2 (Catppuccin Mocha)
	TextColor      = lipgloss.Color("#cdd6f4") // Text (Catppuccin Mocha)

	// Text Styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			MarginBottom(1)

	TextStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	SubtleStyle = lipgloss.NewStyle().
			Foreground(SubtleColor)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(WarningColor).
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
				Foreground(TextColor).
				Background(PrimaryColor).
				Bold(true)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true).
			Underline(true)
)
