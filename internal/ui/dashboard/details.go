package dashboard

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Details is the model for the details component.
type Details struct {
	state       State
	width       int
	height      int
	selectedIdx int // This will be passed from the sidebar
}

// NewDetails creates a new details component.
func NewDetails(s State) Details {
	return Details{
		state:       s,
		selectedIdx: 0,
	}
}

// View renders the details.
func (d Details) View() string {
	if len(d.state.Configs) == 0 {
		return lipgloss.Place(d.width-2, d.height, lipgloss.Center, lipgloss.Center,
			ui.SubtleStyle.Render("No configuration selected"),
		)
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

	// TODO: This is duplicated from sidebar.go, should be refactored
	driftMap := make(map[string]*stow.DriftResult)
	if d.state.DriftSummary != nil {
		for i := range d.state.DriftSummary.Results {
			r := &d.state.DriftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}
	// statusInfo := d.getConfigStatusInfo(cfg, linkStatus, driftMap[cfg.Name])

	title := titleStyle.Render(strings.ToUpper(cfg.Name))
	// statusBadge := lipgloss.NewStyle().
	// 	Foreground(ui.TextColor).
	// 	Background(ui.SubtleColor).
	// 	Padding(0, 1).
	// 	MarginLeft(1).
	// 	Render(statusInfo.statusText)

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

		currentHeight := lipgloss.Height(strings.Join(lines, "\n"))
		if d.height > currentHeight+2 {
			lines = append(lines, strings.Repeat("\n", d.height-currentHeight-2))
			lines = append(lines, statsStyle.Render(statsLine))
		} else {
			lines = append(lines, statsStyle.Render(statsLine))
		}
	}

	return strings.Join(lines, "\n")
}
