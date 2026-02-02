package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// ConflictView displays a modal for resolving file conflicts
type ConflictView struct {
	conflicts   []stow.ConflictFile
	byConfig    map[string][]stow.ConflictFile
	configNames []string // sorted config names for consistent display
	width       int
	height      int
	selectedIdx int // 0=Backup, 1=Delete, 2=Cancel
}

// NewConflictView creates a new conflict resolution view
func NewConflictView(conflicts []stow.ConflictFile) *ConflictView {
	byConfig := GroupConflictsByConfig(conflicts)

	// Get sorted config names for consistent display order
	var configNames []string
	for name := range byConfig {
		configNames = append(configNames, name)
	}
	sort.Strings(configNames)

	return &ConflictView{
		conflicts:   conflicts,
		byConfig:    byConfig,
		configNames: configNames,
		selectedIdx: 0, // Default to Backup (safest option)
	}
}

// Init initializes the conflict view
func (v *ConflictView) Init() tea.Cmd {
	return nil
}

// SetSize updates the view dimensions
func (v *ConflictView) SetSize(width, height int) {
	v.width = width
	v.height = height
}

// Update handles messages for the conflict view
func (v *ConflictView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("left", "h"))):
			if v.selectedIdx > 0 {
				v.selectedIdx--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("right", "l"))):
			if v.selectedIdx < 2 {
				v.selectedIdx++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
			v.selectedIdx = (v.selectedIdx + 1) % 3
		case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
			v.selectedIdx = (v.selectedIdx + 2) % 3
		case key.Matches(msg, key.NewBinding(key.WithKeys("b"))):
			// Shortcut for backup
			return v, v.resolve(ConflictChoiceBackup)
		case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
			// Shortcut for delete
			return v, v.resolve(ConflictChoiceDelete)
		case key.Matches(msg, key.NewBinding(key.WithKeys("c", "esc"))):
			// Shortcut for cancel
			return v, v.resolve(ConflictChoiceCancel)
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			choice := ConflictResolutionChoice(v.selectedIdx)
			return v, v.resolve(choice)
		}
	}
	return v, nil
}

func (v *ConflictView) resolve(choice ConflictResolutionChoice) tea.Cmd {
	return func() tea.Msg {
		if choice == ConflictChoiceCancel {
			return ConflictResolvedMsg{
				Choice:   choice,
				Resolved: false,
			}
		}

		err := ResolveConflictsAction(v.conflicts, choice)
		if err != nil {
			return ConflictResolvedMsg{
				Choice:   choice,
				Resolved: false,
				Error:    err,
			}
		}

		return ConflictResolvedMsg{
			Choice:   choice,
			Resolved: true,
		}
	}
}

// View renders the conflict view
func (v *ConflictView) View() string {
	dialogWidth := 60
	if v.width > 0 && v.width < dialogWidth+20 {
		dialogWidth = v.width - 20
		if dialogWidth < 40 {
			dialogWidth = 40
		}
	}

	// Styles
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

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.WarningColor).
		Padding(1, 2).
		Width(dialogWidth)

	// Build title
	title := titleStyle.Render("File Conflicts Detected")

	// Build subtitle
	subtitle := subtitleStyle.Render(fmt.Sprintf("Found %d conflicting file(s):", len(v.conflicts)))

	// Build file list grouped by config
	var fileLines []string
	home := os.Getenv("HOME")
	maxFilesToShow := 8
	totalShown := 0

	for _, configName := range v.configNames {
		files := v.byConfig[configName]
		fileLines = append(fileLines, configNameStyle.Render(configName+":"))

		showCount := len(files)
		remaining := maxFilesToShow - totalShown
		if showCount > remaining {
			showCount = remaining
		}

		for i := 0; i < showCount; i++ {
			relPath, _ := filepath.Rel(home, files[i].TargetPath)
			fileLines = append(fileLines, fileStyle.Render("~/"+relPath))
			totalShown++
		}

		if showCount < len(files) {
			fileLines = append(fileLines, fileStyle.Render(fmt.Sprintf("... and %d more", len(files)-showCount)))
		}

		if totalShown >= maxFilesToShow {
			break
		}
	}

	// Show if there are more configs not displayed
	if totalShown >= maxFilesToShow {
		remainingConfigs := 0
		for _, name := range v.configNames {
			shown := false
			for _, line := range fileLines {
				if strings.Contains(line, name+":") {
					shown = true
					break
				}
			}
			if !shown {
				remainingConfigs++
			}
		}
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

	// Build hints
	hints := hintStyle.Render("b Backup  d Delete  c Cancel  Enter Select")

	// Build dialog content
	content := lipgloss.JoinVertical(
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

	dialog := borderStyle.Render(content)

	// Center in available space
	return lipgloss.Place(
		v.width,
		v.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#222222")),
	)
}
