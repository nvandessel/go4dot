package status

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// RenderOptions controls output formatting.
type RenderOptions struct {
	JSON bool
}

// Render formats an Overview for display.
// When JSON mode is enabled, it returns machine-readable JSON.
// Otherwise it returns a colorful human-readable summary.
func Render(o *Overview, opts RenderOptions) (string, error) {
	if opts.JSON {
		return renderJSON(o)
	}
	return renderText(o), nil
}

func renderJSON(o *Overview) (string, error) {
	data, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling status to JSON: %w", err)
	}
	return string(data), nil
}

func renderText(o *Overview) string {
	var sb strings.Builder

	// Header
	header := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Render("go4dot status")
	sb.WriteString(header)
	sb.WriteString("\n\n")

	// Platform section
	sectionHeader(&sb, "Platform")
	writeField(&sb, "OS", formatOS(o.Platform))
	writeField(&sb, "Arch", o.Platform.Architecture)
	writeField(&sb, "Package Manager", o.Platform.PackageManager)
	if o.Platform.IsWSL {
		writeField(&sb, "WSL", "yes")
	}
	sb.WriteString("\n")

	if !o.Initialized {
		warning := ui.WarningStyle.Render("No .go4dot.yaml found")
		sb.WriteString(warning)
		sb.WriteString("\n")
		hint := ui.SubtleStyle.Render("  Run 'g4d init' to create one, or 'g4d install <path>' to set up dotfiles.")
		sb.WriteString(hint)
		sb.WriteString("\n")
		return sb.String()
	}

	// Dotfiles section
	sectionHeader(&sb, "Dotfiles")
	writeField(&sb, "Path", o.DotfilesPath)
	writeField(&sb, "Configs", fmt.Sprintf("%d total", o.ConfigCount))
	if o.LastSync != nil {
		writeField(&sb, "Last sync", formatTimeAgo(*o.LastSync))
	}
	sb.WriteString("\n")

	// Config status section
	sectionHeader(&sb, "Configs")

	synced, drifted, notInstalled := countStatuses(o.Configs)
	summaryLine := fmt.Sprintf("%s synced, %s drifted, %s not installed",
		ui.SuccessStyle.Render(fmt.Sprintf("%d", synced)),
		renderDriftedCount(drifted),
		ui.SubtleStyle.Render(fmt.Sprintf("%d", notInstalled)),
	)
	sb.WriteString("  ")
	sb.WriteString(summaryLine)
	sb.WriteString("\n")

	for _, cs := range o.Configs {
		sb.WriteString(renderConfigLine(cs))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Dependencies section
	sectionHeader(&sb, "Dependencies")
	ds := o.Dependencies
	if ds.Total == 0 {
		sb.WriteString("  ")
		sb.WriteString(ui.SubtleStyle.Render("none defined"))
		sb.WriteString("\n")
	} else {
		depLine := fmt.Sprintf("  %s installed",
			ui.SuccessStyle.Render(fmt.Sprintf("%d/%d", ds.Installed, ds.Total)),
		)
		sb.WriteString(depLine)
		if ds.Missing > 0 {
			fmt.Fprintf(&sb, ", %s",
				ui.ErrorStyle.Render(fmt.Sprintf("%d missing", ds.Missing)),
			)
		}
		if ds.VersionMissing > 0 {
			fmt.Fprintf(&sb, ", %s",
				ui.WarningStyle.Render(fmt.Sprintf("%d version mismatch", ds.VersionMissing)),
			)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func sectionHeader(sb *strings.Builder, title string) {
	style := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)
	sb.WriteString(style.Render(title))
	sb.WriteString("\n")
}

func writeField(sb *strings.Builder, label, value string) {
	labelStyle := ui.SubtleStyle
	fmt.Fprintf(sb, "  %s %s\n",
		labelStyle.Render(label+":"),
		value,
	)
}

func formatOS(p PlatformInfo) string {
	if p.Distro != "" {
		s := p.Distro
		if p.DistroVersion != "" {
			s += " " + p.DistroVersion
		}
		return s
	}
	return p.OS
}

func renderConfigLine(cs ConfigStatus) string {
	var icon, label string

	switch cs.Status {
	case SyncStatusSynced:
		icon = ui.SuccessStyle.Render("*")
		label = cs.Name
	case SyncStatusDrifted:
		icon = ui.WarningStyle.Render("~")
		label = cs.Name
		details := driftDetails(cs)
		if details != "" {
			label += " " + ui.SubtleStyle.Render(details)
		}
	case SyncStatusNotInstalled:
		icon = ui.SubtleStyle.Render("-")
		label = ui.SubtleStyle.Render(cs.Name)
	}

	coreTag := ""
	if cs.IsCore {
		coreTag = lipgloss.NewStyle().
			Foreground(ui.PrimaryColor).
			Render(" [core]")
	}

	return fmt.Sprintf("  %s %s%s", icon, label, coreTag)
}

func driftDetails(cs ConfigStatus) string {
	var parts []string
	if cs.NewFiles > 0 {
		parts = append(parts, fmt.Sprintf("+%d new", cs.NewFiles))
	}
	if cs.MissingFiles > 0 {
		parts = append(parts, fmt.Sprintf("-%d missing", cs.MissingFiles))
	}
	if cs.Conflicts > 0 {
		parts = append(parts, fmt.Sprintf("!%d conflicts", cs.Conflicts))
	}
	if len(parts) == 0 {
		return ""
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

func countStatuses(configs []ConfigStatus) (synced, drifted, notInstalled int) {
	for _, c := range configs {
		switch c.Status {
		case SyncStatusSynced:
			synced++
		case SyncStatusDrifted:
			drifted++
		case SyncStatusNotInstalled:
			notInstalled++
		}
	}
	return
}

func renderDriftedCount(count int) string {
	if count == 0 {
		return ui.SuccessStyle.Render("0")
	}
	return ui.WarningStyle.Render(fmt.Sprintf("%d", count))
}

func formatTimeAgo(t time.Time) string {
	d := time.Since(t)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
