package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nvandessel/go4dot/internal/ui"
)

// Details is the model for the details component.
type Details struct {
	state       State
	width       int
	height      int
	selectedIdx int // This will be passed from the sidebar
	viewport    viewport.Model
	ready       bool
}

// NewDetails creates a new details component.
func NewDetails(s State) Details {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()
	return Details{
		state:       s,
		selectedIdx: 0,
		viewport:    vp,
	}
}

// SetSize updates the details pane dimensions
func (d *Details) SetSize(width, height int) {
	d.width = width
	d.height = height
	// Account for border (2) and padding
	contentWidth := width - 4
	contentHeight := height - 2
	if contentWidth < 1 {
		contentWidth = 1
	}
	if contentHeight < 1 {
		contentHeight = 1
	}
	d.viewport.Width = contentWidth
	d.viewport.Height = contentHeight
	d.ready = true
	d.updateContent()
}

// Update handles messages for the details component
func (d *Details) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Forward mouse events to viewport for scrolling
		d.viewport, cmd = d.viewport.Update(msg)
		return cmd
	}

	d.viewport, cmd = d.viewport.Update(msg)
	return cmd
}

// updateContent rebuilds the viewport content
func (d *Details) updateContent() {
	content := d.renderContent()
	d.viewport.SetContent(content)
}

// View renders the details.
func (d Details) View() string {
	if !d.ready {
		return ""
	}
	return d.viewport.View()
}

// renderContent generates the content string for the details pane
func (d Details) renderContent() string {
	if len(d.state.Configs) == 0 {
		return lipgloss.Place(d.width-2, d.height, lipgloss.Center, lipgloss.Center,
			ui.SubtleStyle.Render("No configuration selected"),
		)
	}

	if d.selectedIdx >= len(d.state.Configs) {
		return ""
	}

	cfg := d.state.Configs[d.selectedIdx]
	linkStatus := d.state.LinkStatus[cfg.Name]

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle
	subtleStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Bold(true).
		Background(ui.PrimaryColor).
		Padding(0, 1)

	title := titleStyle.Render(strings.ToUpper(cfg.Name))

	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, title))
	lines = append(lines, "")

	if cfg.Description != "" {
		descStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Italic(true).Width(d.width - 4)
		lines = append(lines, descStyle.Render(cfg.Description))
		lines = append(lines, "")
	}

	if linkStatus != nil {
		lines = append(lines, headerStyle.Render("FILESYSTEM MAPPINGS"))

		for _, f := range linkStatus.Files {
			icon := okStyle.Render("✓")
			if !f.IsLinked {
				if strings.Contains(strings.ToLower(f.Issue), "conflict") ||
					strings.Contains(strings.ToLower(f.Issue), "exists") ||
					strings.Contains(strings.ToLower(f.Issue), "elsewhere") {
					icon = warnStyle.Render("⚠")
				} else {
					icon = errStyle.Render("✗")
				}
			}

			source := f.RelPath
			target := filepath.Join("~", f.RelPath)

			mapping := fmt.Sprintf("%s %s %s %s", icon, source, subtleStyle.Render("→"), target)
			lines = append(lines, mapping)

			if !f.IsLinked && f.Issue != "" {
				lines = append(lines, subtleStyle.Render("    └─ "+f.Issue))
			}
		}
		lines = append(lines, "")
	}

	if len(cfg.DependsOn) > 0 {
		lines = append(lines, headerStyle.Render("MODULE DEPENDENCIES"))
		for _, depName := range cfg.DependsOn {
			status := subtleStyle.Render("(unknown)")
			if d.state.LinkStatus != nil {
				if depStatus, ok := d.state.LinkStatus[depName]; ok {
					if depStatus.IsFullyLinked() {
						status = okStyle.Render("(✓ linked)")
					} else {
						status = warnStyle.Render("(✗ missing)")
					}
				}
			}
			lines = append(lines, fmt.Sprintf("• %s %s", depName, status))
		}
		lines = append(lines, "")
	}

	if len(cfg.ExternalDeps) > 0 {
		lines = append(lines, headerStyle.Render("EXTERNAL REPOSITORIES"))
		for _, extDep := range cfg.ExternalDeps {
			lines = append(lines, fmt.Sprintf("• %s", extDep.URL))
			lines = append(lines, subtleStyle.Render("  └─ "+extDep.Destination))
		}
		lines = append(lines, "")
	}

	if linkStatus != nil {
		statsLine := fmt.Sprintf("Total: %d files", linkStatus.TotalCount)
		statsStyle := lipgloss.NewStyle().
			Foreground(ui.SubtleColor).
			Align(lipgloss.Right).
			Width(d.width - 4)

		lines = append(lines, statsStyle.Render(statsLine))
	}

	return strings.Join(lines, "\n")
}
