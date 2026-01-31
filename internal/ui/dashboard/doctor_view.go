package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/doctor"
	"github.com/nvandessel/go4dot/internal/ui"
)

// DoctorViewCloseMsg is sent when the doctor view should close
type DoctorViewCloseMsg struct{}

// doctorResultMsg is sent when doctor checks complete
type doctorResultMsg struct {
	result *doctor.CheckResult
	err    error
}

// DoctorView displays health check results
type DoctorView struct {
	cfg          *config.Config
	dotfilesPath string

	result   *doctor.CheckResult
	viewport viewport.Model
	spinner  spinner.Model
	width    int
	height   int
	ready    bool
	loading  bool
}

// NewDoctorView creates a new doctor view
func NewDoctorView(cfg *config.Config, dotfilesPath string) *DoctorView {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	return &DoctorView{
		cfg:          cfg,
		dotfilesPath: dotfilesPath,
		viewport:     vp,
		spinner:      s,
		loading:      true,
	}
}

// Init starts running doctor checks
func (d *DoctorView) Init() tea.Cmd {
	return tea.Batch(
		d.spinner.Tick,
		d.runChecks,
	)
}

func (d *DoctorView) runChecks() tea.Msg {
	opts := doctor.CheckOptions{
		DotfilesPath: d.dotfilesPath,
	}
	result, err := doctor.RunChecks(d.cfg, opts)
	return doctorResultMsg{result: result, err: err}
}

// SetSize updates the view dimensions
func (d *DoctorView) SetSize(width, height int) {
	d.width = width
	d.height = height
	contentWidth := width - 6
	contentHeight := height - 8
	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}
	d.viewport.Width = contentWidth
	d.viewport.Height = contentHeight
	d.ready = true
	d.updateContent()
}

// Update handles messages
func (d *DoctorView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			return d, func() tea.Msg { return DoctorViewCloseMsg{} }
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case doctorResultMsg:
		d.loading = false
		if msg.err != nil {
			d.result = nil
		} else {
			d.result = msg.result
		}
		d.updateContent()
	}

	// Forward to viewport for scrolling (handles mouse and keyboard events)
	var cmd tea.Cmd
	d.viewport, cmd = d.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return d, tea.Batch(cmds...)
}

// View renders the doctor view
func (d *DoctorView) View() string {
	if !d.ready {
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
		Width(d.width - 4).
		Height(d.height - 4)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	var content string
	if d.loading {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("🏥 Health Check"),
			"",
			d.spinner.View()+" Running diagnostics...",
		)
	} else {
		// Summary line
		summary := d.renderSummary()

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("🏥 Health Check"),
			summary,
			"",
			d.viewport.View(),
			"",
			hintStyle.Render("↑/↓ Scroll • ESC Close"),
		)
	}

	dialog := borderStyle.Render(content)

	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (d *DoctorView) renderSummary() string {
	if d.result == nil {
		return ui.ErrorStyle.Render("Failed to run health checks")
	}

	ok, warnings, errors, skipped := d.result.CountByStatus()

	var parts []string
	if errors > 0 {
		parts = append(parts, ui.ErrorStyle.Render(fmt.Sprintf("%d errors", errors)))
	}
	if warnings > 0 {
		parts = append(parts, ui.WarningStyle.Render(fmt.Sprintf("%d warnings", warnings)))
	}
	if ok > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(ui.SecondaryColor).Render(fmt.Sprintf("%d passed", ok)))
	}
	if skipped > 0 {
		parts = append(parts, ui.SubtleStyle.Render(fmt.Sprintf("%d skipped", skipped)))
	}

	return strings.Join(parts, " • ")
}

func (d *DoctorView) updateContent() {
	if d.result == nil {
		d.viewport.SetContent("No results available.")
		return
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	skipStyle := ui.SubtleStyle
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	fixStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Italic(true)

	for _, check := range d.result.Checks {
		// Status icon
		var icon string
		switch check.Status {
		case doctor.StatusOK:
			icon = okStyle.Render("✓")
		case doctor.StatusWarning:
			icon = warnStyle.Render("⚠")
		case doctor.StatusError:
			icon = errStyle.Render("✗")
		case doctor.StatusSkipped:
			icon = skipStyle.Render("○")
		}

		// Build check line
		lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(check.Name)))
		lines = append(lines, "   "+descStyle.Render(check.Description))

		// Message
		if check.Message != "" {
			msgStyle := descStyle
			switch check.Status {
			case doctor.StatusError:
				msgStyle = errStyle
			case doctor.StatusWarning:
				msgStyle = warnStyle
			}
			lines = append(lines, "   "+msgStyle.Render(check.Message))
		}

		// Fix suggestion
		if check.Fix != "" {
			lines = append(lines, "   "+fixStyle.Render("Fix: "+check.Fix))
		}

		lines = append(lines, "")
	}

	// External status
	if len(d.result.ExternalStatus) > 0 {
		lines = append(lines, nameStyle.Render("External Dependencies"))
		lines = append(lines, strings.Repeat("─", 40))
		for _, ext := range d.result.ExternalStatus {
			var icon string
			switch ext.Status {
			case "installed":
				icon = okStyle.Render("✓")
			case "missing":
				icon = warnStyle.Render("○")
			case "error":
				icon = errStyle.Render("✗")
			case "skipped":
				icon = skipStyle.Render("⊘")
			default:
				icon = skipStyle.Render("⊘")
			}
			name := ext.Dep.Name
			if name == "" {
				name = ext.Dep.ID
			}
			lines = append(lines, fmt.Sprintf("  %s %s", icon, name))
		}
		lines = append(lines, "")
	}

	// Machine config status
	if len(d.result.MachineStatus) > 0 {
		lines = append(lines, nameStyle.Render("Machine Configuration"))
		lines = append(lines, strings.Repeat("─", 40))
		for _, mc := range d.result.MachineStatus {
			var icon string
			switch mc.Status {
			case "configured":
				icon = okStyle.Render("✓")
			case "missing":
				icon = warnStyle.Render("○")
			case "error":
				icon = errStyle.Render("✗")
			}
			lines = append(lines, fmt.Sprintf("  %s %s", icon, mc.Description))
		}
		lines = append(lines, "")
	}

	d.viewport.SetContent(strings.Join(lines, "\n"))
}
