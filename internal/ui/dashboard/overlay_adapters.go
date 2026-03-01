package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// overlayHelpContent returns the help content for overlay compositing (without box frame).
func overlayHelpContent(h Help) string {
	var b strings.Builder
	boxWidth := 60
	if h.width > 0 && h.width < boxWidth+4 {
		boxWidth = h.width - 4
	}
	if boxWidth < 0 {
		boxWidth = 0
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(boxWidth).
		Align(lipgloss.Center).
		MarginBottom(1)

	headerStyle := lipgloss.NewStyle().
		Foreground(ui.SecondaryColor).
		Bold(true).
		MarginTop(1).
		MarginLeft(2)

	keyStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(14).
		Align(lipgloss.Right)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		MarginLeft(2)

	subtleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Width(boxWidth).
		Align(lipgloss.Center).
		MarginTop(1)

	b.WriteString(titleStyle.Render("go4dot Dashboard - Keyboard Shortcuts"))
	b.WriteString("\n")

	b.WriteString(headerStyle.Render("Navigation"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("↑/k"), descStyle.Render("Move selection up"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("↓/j"), descStyle.Render("Move selection down"))

	b.WriteString(headerStyle.Render("Actions"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("enter"), descStyle.Render("Sync selected config"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("s"), descStyle.Render("Sync all configs"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("shift+s"), descStyle.Render("Sync selected configs"))

	b.WriteString(headerStyle.Render("Selection & Filter"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("space"), descStyle.Render("Toggle selection"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("shift+a"), descStyle.Render("Select/deselect all visible"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("/"), descStyle.Render("Enter filter mode"))

	b.WriteString(headerStyle.Render("Other"))
	b.WriteString("\n")
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("d"), descStyle.Render("Run doctor check"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("m"), descStyle.Render("Configure overrides"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("tab"), descStyle.Render("More commands menu"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("?"), descStyle.Render("Toggle help screen"))
	fmt.Fprintf(&b, "%s%s\n", keyStyle.Render("q / esc"), descStyle.Render("Quit dashboard"))

	b.WriteString(subtleStyle.Render("Press ?, q, or esc to close"))

	// Force all lines to uniform width so the overlay background fills evenly
	return lipgloss.NewStyle().Width(boxWidth).Render(b.String())
}

// overlayMenuContent returns the menu content for overlay compositing (without box frame).
// The content is constrained to a compact size so the menu feels like a small
// dropdown/popup rather than a full-screen takeover.
func overlayMenuContent(m *Menu) string {
	w := CompactWidth(m.width)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		m.list.View(),
		"",
		hintStyle.Render("ESC to close"),
	)

	return lipgloss.NewStyle().Width(w).Render(content)
}

// overlayConfirmContent returns the confirm dialog content for overlay compositing (without border/placement).
func overlayConfirmContent(c *Confirm) string {
	dialogWidth := 50
	if c.width > 0 && c.width < dialogWidth+20 {
		dialogWidth = c.width - 20
		if dialogWidth < 30 {
			dialogWidth = 30
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	dStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	selectedBtnStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.PrimaryColor).
		Padding(0, 3).
		Bold(true)

	normalBtnStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Padding(0, 3)

	var yesBtn, noBtn string
	if c.selected == 0 {
		yesBtn = selectedBtnStyle.Render(c.affirmative)
		noBtn = normalBtnStyle.Render(c.negative)
	} else {
		yesBtn = normalBtnStyle.Render(c.affirmative)
		noBtn = selectedBtnStyle.Render(c.negative)
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "  ", noBtn)
	buttonsRow := lipgloss.NewStyle().Width(dialogWidth - 4).Align(lipgloss.Center).Render(buttons)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render(c.title),
		"",
		dStyle.Render(c.description),
		"",
		buttonsRow,
	)
}

// overlayOnboardingContent returns the onboarding content for overlay compositing (without placement).
func overlayOnboardingContent(o *Onboarding) string {
	if o.quitting {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(1, 0)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	formView := ""
	if o.form != nil {
		formView = o.form.View()
	}

	var content string

	switch o.step {
	case stepScanning:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Initializing go4dot"),
			"",
			o.spinner.View()+" Scanning for dotfiles...",
		)
	case stepWriting:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Creating Configuration"),
			"",
			o.spinner.View()+" Writing .go4dot.yaml...",
		)
	case stepComplete:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Configuration Created"),
			"",
			ui.SuccessStyle.Render("Your .go4dot.yaml has been created!"),
			"",
			subtitleStyle.Render("Run 'g4d install' to set up your dotfiles."),
		)
	case stepMetadata:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Project Information"),
			subtitleStyle.Render(fmt.Sprintf("Found %d potential configs", len(o.scannedConfigs))),
			"",
			formView,
		)
	case stepConfigs:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Select Configurations"),
			subtitleStyle.Render("Choose which configs to manage"),
			"",
			formView,
		)
	case stepExternal:
		title := "External Dependencies"
		if len(o.externalDeps) > 0 {
			title = fmt.Sprintf("External Dependencies (%d added)", len(o.externalDeps))
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			subtitleStyle.Render("Git repos for plugins, themes, etc."),
			"",
			formView,
		)
	case stepExternalDetails:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Add External Dependency"),
			subtitleStyle.Render("Enter git repository details"),
			"",
			formView,
		)
	case stepDependencies:
		title := "System Dependencies"
		if len(o.systemDeps) > 0 {
			title = fmt.Sprintf("System Dependencies (%d added)", len(o.systemDeps))
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			subtitleStyle.Render("Required packages (neovim, tmux, etc.)"),
			"",
			formView,
		)
	case stepDependenciesDetails:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Add System Dependency"),
			subtitleStyle.Render("Enter package details"),
			"",
			formView,
		)
	case stepMachine:
		title := "Machine Configuration"
		if len(o.machineConfigs) > 0 {
			title = fmt.Sprintf("Machine Configuration (%d added)", len(o.machineConfigs))
		}
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(title),
			subtitleStyle.Render("Machine-specific settings (git signing, etc.)"),
			"",
			formView,
		)
	case stepMachineDetails:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Configure Machine Setting"),
			subtitleStyle.Render("Enter configuration details"),
			"",
			formView,
		)
	case stepConfirm:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Review Configuration"),
			"",
			o.renderSummary(),
			"",
			formView,
		)
	}

	return content
}

// overlayConfigListContent returns the config list content for overlay compositing (without border/placement).
func overlayConfigListContent(c *ConfigListView) string {
	if !c.ready {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(0, 1)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Configuration List"),
		"",
		c.viewport.View(),
		"",
		hintStyle.Render("Press ESC or q to close"),
	)
}

// overlayExternalContent returns the external view content for overlay compositing (without border/placement).
func overlayExternalContent(e *ExternalView) string {
	if !e.ready {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(0, 1)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	if e.loading {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("External Dependencies"),
			"",
			e.spinner.View()+" Loading status...",
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("External Dependencies"),
		"",
		e.viewport.View(),
		"",
		hintStyle.Render("Navigate  ESC Close"),
	)
}

// overlayMachineContent returns the machine view content for overlay compositing (without border/placement).
func overlayMachineContent(m *MachineView) string {
	if !m.ready {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(0, 1)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	if m.currentForm != nil {
		formTitle := "Configure"
		if m.currentConfig != nil {
			formTitle = m.currentConfig.Description
		}
		return lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render(formTitle),
			"",
			m.currentForm.View(),
			"",
			hintStyle.Render("ESC Cancel"),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("Machine Configuration"),
		"",
		m.viewport.View(),
		"",
		hintStyle.Render("Navigate  Enter Configure  ESC Close"),
	)
}

// overlayConflictContent returns the conflict view content for overlay compositing (without border/placement).
// This extracts the inner content from ConflictView without the border frame and lipgloss.Place wrapping.
func overlayConflictContent(v *ConflictView) string {
	dialogWidth := 60
	if v.width > 0 && v.width < dialogWidth+20 {
		dialogWidth = v.width - 20
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.WarningColor).
		Bold(true).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	configNameStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	fileStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		PaddingLeft(2)

	selectedBtnStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Background(ui.PrimaryColor).
		Padding(0, 2).
		Bold(true)

	normalBtnStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Padding(0, 2)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	// Build title and subtitle
	title := titleStyle.Render("File Conflicts Detected")
	subtitle := subtitleStyle.Render(fmt.Sprintf("Found %d conflicting file(s):", len(v.conflicts)))

	// Build file list grouped by config
	var fileLines []string
	home, _ := os.UserHomeDir()
	maxFilesToShow := 8
	totalShown := 0
	displayedConfigs := make(map[string]bool)

	for _, configName := range v.configNames {
		files := v.byConfig[configName]
		fileLines = append(fileLines, configNameStyle.Render(configName+":"))
		displayedConfigs[configName] = true

		showCount := len(files)
		remaining := maxFilesToShow - totalShown
		if showCount > remaining {
			showCount = remaining
		}

		for i := 0; i < showCount; i++ {
			displayPath := files[i].TargetPath
			if home != "" {
				if relPath, err := filepath.Rel(home, files[i].TargetPath); err == nil && !strings.HasPrefix(relPath, "..") {
					displayPath = "~/" + relPath
				}
			}
			fileLines = append(fileLines, fileStyle.Render(displayPath))
			totalShown++
		}

		if showCount < len(files) {
			fileLines = append(fileLines, fileStyle.Render(fmt.Sprintf("... and %d more", len(files)-showCount)))
		}

		if totalShown >= maxFilesToShow {
			break
		}
	}

	if totalShown >= maxFilesToShow {
		remainingConfigs := len(v.configNames) - len(displayedConfigs)
		if remainingConfigs > 0 {
			fileLines = append(fileLines, fileStyle.Render(fmt.Sprintf("... and %d more config(s)", remainingConfigs)))
		}
	}

	fileList := strings.Join(fileLines, "\n")

	// Build buttons
	var backupBtn, deleteBtn, cancelBtn string
	if v.selectedIdx == 0 {
		backupBtn = selectedBtnStyle.Render("Backup (.g4d-backup)")
	} else {
		backupBtn = normalBtnStyle.Render("Backup (.g4d-backup)")
	}
	if v.selectedIdx == 1 {
		deleteBtn = selectedBtnStyle.Render("Delete")
	} else {
		deleteBtn = normalBtnStyle.Render("Delete")
	}
	if v.selectedIdx == 2 {
		cancelBtn = selectedBtnStyle.Render("Cancel")
	} else {
		cancelBtn = normalBtnStyle.Render("Cancel")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, backupBtn, "  ", deleteBtn, "  ", cancelBtn)
	buttonsRow := lipgloss.NewStyle().Width(dialogWidth - 4).Align(lipgloss.Center).Render(buttons)

	hints := hintStyle.Render("b Backup  d Delete  c Cancel  Enter Select")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		subtitle,
		"",
		fileList,
		"",
		buttonsRow,
		"",
		hints,
	)
}
