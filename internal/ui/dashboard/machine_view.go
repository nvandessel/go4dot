package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/machine"
	"github.com/nvandessel/go4dot/internal/ui"
)

// MachineViewCloseMsg is sent when the machine view should close
type MachineViewCloseMsg struct{}

// MachineConfigCompleteMsg is sent when a machine config is completed
type MachineConfigCompleteMsg struct {
	ID     string
	Values map[string]string
}

// MachineView displays machine configuration options and forms
type MachineView struct {
	cfg           *config.Config
	machineStatus []machine.MachineConfigStatus

	viewport    viewport.Model
	width       int
	height      int
	ready       bool
	selectedIdx int

	// Current form being displayed
	currentForm   *huh.Form
	currentConfig *config.MachinePrompt
	formValues    map[string]string
}

// NewMachineView creates a new machine configuration view
func NewMachineView(cfg *config.Config) *MachineView {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	status := machine.CheckMachineConfigStatus(cfg)

	return &MachineView{
		cfg:           cfg,
		machineStatus: status,
		viewport:      vp,
		formValues:    make(map[string]string),
	}
}

// Init initializes the machine view
func (m *MachineView) Init() tea.Cmd {
	return nil
}

// SetSize updates the view dimensions
func (m *MachineView) SetSize(width, height int) {
	m.width = width
	m.height = height
	contentWidth := width - 6
	contentHeight := height - 10
	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}
	m.viewport.Width = contentWidth
	m.viewport.Height = contentHeight
	m.ready = true
	m.updateContent()
}

// Update handles messages
func (m *MachineView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If we have an active form, handle it
	if m.currentForm != nil {
		return m.updateForm(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			return m, func() tea.Msg { return MachineViewCloseMsg{} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if m.selectedIdx < len(m.cfg.MachineConfig)-1 {
				m.selectedIdx++
				m.updateContent()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if m.selectedIdx > 0 {
				m.selectedIdx--
				m.updateContent()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			if m.selectedIdx < len(m.cfg.MachineConfig) {
				return m.startConfigForm()
			}
		}

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *MachineView) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle form abort
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("esc"))) {
			m.currentForm = nil
			m.currentConfig = nil
			return m, nil
		}
	}

	// Forward to form
	form, cmd := m.currentForm.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.currentForm = f
	}

	// Check for completion
	if m.currentForm.State == huh.StateCompleted {
		// Extract values and send completion message
		configID := m.currentConfig.ID
		m.currentForm = nil
		m.currentConfig = nil

		// Refresh status
		m.machineStatus = machine.CheckMachineConfigStatus(m.cfg)
		m.updateContent()

		return m, func() tea.Msg {
			return MachineConfigCompleteMsg{
				ID:     configID,
				Values: m.formValues,
			}
		}
	}

	// Check for abort
	if m.currentForm.State == huh.StateAborted {
		m.currentForm = nil
		m.currentConfig = nil
		return m, nil
	}

	return m, cmd
}

func (m *MachineView) startConfigForm() (tea.Model, tea.Cmd) {
	if m.selectedIdx >= len(m.cfg.MachineConfig) {
		return m, nil
	}

	mc := &m.cfg.MachineConfig[m.selectedIdx]
	m.currentConfig = mc
	m.formValues = make(map[string]string)

	// Build form fields from machine config prompts
	var fields []huh.Field
	for _, prompt := range mc.Prompts {
		val := prompt.Default
		m.formValues[prompt.ID] = val

		switch prompt.Type {
		case "confirm":
			var boolVal bool
			if val == "true" || val == "yes" {
				boolVal = true
			}
			fields = append(fields, huh.NewConfirm().
				Title(prompt.Prompt).
				Value(&boolVal))
		case "select":
			if len(prompt.Options) > 0 {
				var options []huh.Option[string]
				for _, opt := range prompt.Options {
					options = append(options, huh.NewOption(opt, opt))
				}
				fields = append(fields, huh.NewSelect[string]().
					Title(prompt.Prompt).
					Options(options...).
					Value(&val))
			} else {
				fields = append(fields, huh.NewInput().
					Title(prompt.Prompt).
					Value(&val))
			}
		default: // text
			valPtr := &val
			f := huh.NewInput().
				Title(prompt.Prompt).
				Value(valPtr)
			if prompt.Required {
				f.Validate(func(s string) error {
					if s == "" {
						return fmt.Errorf("required")
					}
					return nil
				})
			}
			fields = append(fields, f)
		}
	}

	m.currentForm = huh.NewForm(huh.NewGroup(fields...)).
		WithWidth(m.width - 20).
		WithShowHelp(false)

	return m, m.currentForm.Init()
}

// View renders the machine view
func (m *MachineView) View() string {
	if !m.ready {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Padding(1, 2).
		Width(m.width - 4).
		Height(m.height - 4)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	// If we have an active form, show it
	if m.currentForm != nil {
		formTitle := "Configure"
		if m.currentConfig != nil {
			formTitle = m.currentConfig.Description
		}
		content := lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("🖥️ "+formTitle),
			"",
			m.currentForm.View(),
			"",
			hintStyle.Render("ESC Cancel"),
		)
		dialog := borderStyle.Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("🖥️ Machine Configuration"),
		"",
		m.viewport.View(),
		"",
		hintStyle.Render("↑/↓ Navigate • Enter Configure • ESC Close"),
	)

	dialog := borderStyle.Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (m *MachineView) updateContent() {
	if len(m.cfg.MachineConfig) == 0 {
		m.viewport.SetContent("No machine configurations defined.")
		return
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("#333333"))

	// Build status map
	statusMap := make(map[string]string)
	for _, s := range m.machineStatus {
		statusMap[s.ID] = s.Status
	}

	for i, mc := range m.cfg.MachineConfig {
		// Status icon
		var icon string
		status := statusMap[mc.ID]
		switch status {
		case "configured":
			icon = okStyle.Render("✓")
		case "missing":
			icon = warnStyle.Render("○")
		case "error":
			icon = errStyle.Render("✗")
		default:
			icon = descStyle.Render("?")
		}

		// Build line
		line := fmt.Sprintf("%s %s", icon, nameStyle.Render(mc.Description))
		if i == m.selectedIdx {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)

		// Destination
		lines = append(lines, descStyle.Render("   → "+mc.Destination))

		// Prompts count
		promptCount := len(mc.Prompts)
		lines = append(lines, descStyle.Render(fmt.Sprintf("   %d field(s) to configure", promptCount)))

		lines = append(lines, "")
	}

	m.viewport.SetContent(strings.Join(lines, "\n"))
}
