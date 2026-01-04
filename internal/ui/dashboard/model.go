package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Action represents a user action from the dashboard
type Action int

const (
	ActionNone Action = iota
	ActionSync
	ActionSyncConfig
	ActionDoctor
	ActionInstall
	ActionInit
	ActionQuit
)

// Result is returned when the dashboard exits
type Result struct {
	Action     Action
	ConfigName string // For ActionSyncConfig
}

// Model is the Bubbletea model for the dashboard
type Model struct {
	width        int
	height       int
	platform     *platform.Platform
	driftSummary *stow.DriftSummary
	configs      []config.ConfigItem
	dotfilesPath string
	updateMsg    string
	selectedIdx  int
	result       *Result
	quitting     bool
	hasBaseline  bool // True if we have stored symlink counts (synced before)
}

// keyMap defines the key bindings
type keyMap struct {
	Sync   key.Binding
	Doctor key.Binding
	Quit   key.Binding
	Up     key.Binding
	Down   key.Binding
	Enter  key.Binding
	Help   key.Binding
}

var keys = keyMap{
	Sync: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sync all"),
	),
	Doctor: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "doctor"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("up/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("down/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "sync config"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
}

// New creates a new dashboard model
func New(p *platform.Platform, driftSummary *stow.DriftSummary, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool) Model {
	return Model{
		platform:     p,
		driftSummary: driftSummary,
		configs:      configs,
		dotfilesPath: dotfilesPath,
		updateMsg:    updateMsg,
		selectedIdx:  0,
		hasBaseline:  hasBaseline,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.result = &Result{Action: ActionQuit}
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Sync):
			m.result = &Result{Action: ActionSync}
			return m, tea.Quit

		case key.Matches(msg, keys.Doctor):
			m.result = &Result{Action: ActionDoctor}
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case key.Matches(msg, keys.Down):
			if m.selectedIdx < len(m.configs)-1 {
				m.selectedIdx++
			}

		case key.Matches(msg, keys.Enter):
			if len(m.configs) > 0 && m.selectedIdx < len(m.configs) {
				m.result = &Result{
					Action:     ActionSyncConfig,
					ConfigName: m.configs[m.selectedIdx].Name,
				}
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n\n")

	// Status summary
	status := m.renderStatus()
	b.WriteString(status)
	b.WriteString("\n\n")

	// Config list
	configList := m.renderConfigList()
	b.WriteString(configList)
	b.WriteString("\n\n")

	// Action bar
	actions := m.renderActions()
	b.WriteString(actions)

	return b.String()
}

func (m Model) renderHeader() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	title := titleStyle.Render("go4dot Dashboard")

	platformInfo := ""
	if m.platform != nil {
		platformInfo = fmt.Sprintf(" %s (%s)", m.platform.OS, m.platform.PackageManager)
	}

	subtitle := subtitleStyle.Render(platformInfo)

	updateInfo := ""
	if m.updateMsg != "" {
		updateInfo = subtitleStyle.Render(" " + m.updateMsg)
	}

	return title + subtitle + updateInfo
}

func (m Model) renderStatus() string {
	// If we haven't synced before (no baseline), show that clearly
	if !m.hasBaseline {
		return lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render("  Not synced yet - press [s] to create symlinks")
	}

	// We have baseline - check for drift
	if m.driftSummary != nil && m.driftSummary.HasDrift() {
		return lipgloss.NewStyle().
			Foreground(ui.WarningColor).
			Bold(true).
			Render(fmt.Sprintf("  %d config(s) need syncing", m.driftSummary.DriftedConfigs))
	}

	// Baseline exists and no drift
	return lipgloss.NewStyle().
		Foreground(ui.SecondaryColor).
		Bold(true).
		Render("  All synced")
}

func (m Model) renderConfigList() string {
	var lines []string

	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	selectedStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	subtleStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)

	// Build a map of drift results for quick lookup
	driftMap := make(map[string]*stow.DriftResult)
	if m.driftSummary != nil {
		for i := range m.driftSummary.Results {
			r := &m.driftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}

	for i, cfg := range m.configs {
		var line string
		prefix := "  "
		if i == m.selectedIdx {
			prefix = "> "
		}

		nameStyle := normalStyle
		if i == m.selectedIdx {
			nameStyle = selectedStyle
		}

		// Check drift status
		drift, hasDrift := driftMap[cfg.Name]
		statusIcon := okStyle.Render("")
		statusText := ""

		if hasDrift && drift.HasDrift {
			statusIcon = warnStyle.Render("")
			newCount := len(drift.NewFiles)
			if newCount > 0 {
				statusText = fmt.Sprintf(" %d new file(s)", newCount)
			}
		} else if hasDrift {
			statusText = fmt.Sprintf(" %d files", drift.CurrentCount)
		}

		// Pad name to align status
		paddedName := fmt.Sprintf("%-20s", cfg.Name)
		dots := subtleStyle.Render(strings.Repeat(".", 20-len(cfg.Name)))

		line = fmt.Sprintf("%s%s %s %s%s",
			prefix,
			nameStyle.Render(paddedName[:len(cfg.Name)]),
			dots,
			statusIcon,
			subtleStyle.Render(statusText),
		)

		lines = append(lines, line)

		// Show new files if this is the selected item and has drift
		if i == m.selectedIdx && hasDrift && drift.HasDrift && len(drift.NewFiles) > 0 {
			for j, f := range drift.NewFiles {
				if j >= 3 {
					lines = append(lines, subtleStyle.Render(fmt.Sprintf("      ... and %d more", len(drift.NewFiles)-3)))
					break
				}
				lines = append(lines, subtleStyle.Render(fmt.Sprintf("       %s", f)))
			}
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderActions() string {
	style := lipgloss.NewStyle().Foreground(ui.SubtleColor)
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)

	actions := []string{
		keyStyle.Render("[s]") + style.Render(" Sync All"),
		keyStyle.Render("[d]") + style.Render(" Doctor"),
		keyStyle.Render("[enter]") + style.Render(" Sync Selected"),
		keyStyle.Render("[q]") + style.Render(" Quit"),
	}

	return strings.Join(actions, "   ")
}

// GetResult returns the action result after the model exits
func (m Model) GetResult() *Result {
	return m.result
}

// Run starts the dashboard and returns the selected action
func Run(p *platform.Platform, driftSummary *stow.DriftSummary, configs []config.ConfigItem, dotfilesPath string, updateMsg string, hasBaseline bool) (*Result, error) {
	m := New(p, driftSummary, configs, dotfilesPath, updateMsg, hasBaseline)

	finalModel, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(Model).GetResult(), nil
}

// SetupModel is the Bubbletea model for the setup screen (no config)
type SetupModel struct {
	width      int
	height     int
	platform   *platform.Platform
	updateMsg  string
	result     *Result
	quitting   bool
	selectedIdx int
}

// NewSetup creates a new setup model
func NewSetup(p *platform.Platform, updateMsg string) SetupModel {
	return SetupModel{
		platform:  p,
		updateMsg: updateMsg,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return nil
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.result = &Result{Action: ActionQuit}
			m.quitting = true
			return m, tea.Quit

		case "i", "enter":
			m.result = &Result{Action: ActionInit}
			return m, tea.Quit

		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}

		case "down", "j":
			if m.selectedIdx < 1 {
				m.selectedIdx++
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m SetupModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Header - same style as dashboard
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	title := titleStyle.Render("go4dot")
	platformInfo := ""
	if m.platform != nil {
		platformInfo = fmt.Sprintf(" %s (%s)", m.platform.OS, m.platform.PackageManager)
	}
	subtitle := subtitleStyle.Render(platformInfo)
	updateInfo := ""
	if m.updateMsg != "" {
		updateInfo = subtitleStyle.Render(" " + m.updateMsg)
	}

	b.WriteString(title + subtitle + updateInfo)
	b.WriteString("\n\n")

	// Status
	statusStyle := lipgloss.NewStyle().
		Foreground(ui.WarningColor).
		Bold(true)
	b.WriteString(statusStyle.Render("  No configuration found"))
	b.WriteString("\n\n")

	// Message
	messageStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	b.WriteString(messageStyle.Render("  No .go4dot.yaml found in current directory."))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("  Initialize go4dot to start managing your dotfiles."))
	b.WriteString("\n\n")

	// Options
	normalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	selectedStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)

	options := []struct {
		label string
		desc  string
	}{
		{"Initialize go4dot", "Set up a new .go4dot.yaml config"},
		{"Quit", "Exit go4dot"},
	}

	for i, opt := range options {
		prefix := "  "
		style := normalStyle
		if i == m.selectedIdx {
			prefix = "> "
			style = selectedStyle
		}
		b.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(opt.label)))
		if i == m.selectedIdx {
			b.WriteString(subtitleStyle.Render(fmt.Sprintf("    %s", opt.desc)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")

	// Action bar - same style as dashboard
	keyStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Bold(true)
	actions := []string{
		keyStyle.Render("[i]") + subtitleStyle.Render(" Initialize"),
		keyStyle.Render("[q]") + subtitleStyle.Render(" Quit"),
	}
	b.WriteString(strings.Join(actions, "   "))

	return b.String()
}

// GetResult returns the action result after the model exits
func (m SetupModel) GetResult() *Result {
	return m.result
}

// RunSetup starts the setup screen and returns the selected action
func RunSetup(p *platform.Platform, updateMsg string) (*Result, error) {
	m := NewSetup(p, updateMsg)

	finalModel, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(SetupModel).GetResult(), nil
}
