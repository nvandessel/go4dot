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
	"github.com/nvandessel/go4dot/internal/deps"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/ui"
)

// ExternalViewCloseMsg is sent when the external view should close
type ExternalViewCloseMsg struct{}

// externalStatusLoadedMsg is sent when external status is loaded
type externalStatusLoadedMsg struct {
	status []deps.ExternalStatus
	err    error
}

// ExternalView displays external dependencies management
type ExternalView struct {
	cfg          *config.Config
	dotfilesPath string
	platform     *platform.Platform

	status   []deps.ExternalStatus
	viewport viewport.Model
	spinner  spinner.Model
	width    int
	height   int
	ready    bool
	loading  bool
	selected int
}

// NewExternalView creates a new external dependencies view
func NewExternalView(cfg *config.Config, dotfilesPath string, plat *platform.Platform) *ExternalView {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(ui.PrimaryColor)

	return &ExternalView{
		cfg:          cfg,
		dotfilesPath: dotfilesPath,
		platform:     plat,
		viewport:     vp,
		spinner:      s,
		loading:      true,
	}
}

// Init starts loading external status
func (e *ExternalView) Init() tea.Cmd {
	return tea.Batch(
		e.spinner.Tick,
		e.loadStatus,
	)
}

func (e *ExternalView) loadStatus() tea.Msg {
	status := deps.CheckExternalStatus(e.cfg, e.platform, e.dotfilesPath)
	return externalStatusLoadedMsg{status: status}
}

// SetSize updates the view dimensions
func (e *ExternalView) SetSize(width, height int) {
	e.width = width
	e.height = height
	contentWidth := width - 6
	contentHeight := height - 10 // Account for title, hints, borders
	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}
	e.viewport.Width = contentWidth
	e.viewport.Height = contentHeight
	e.ready = true
	e.updateContent()
}

// Update handles messages
func (e *ExternalView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			return e, func() tea.Msg { return ExternalViewCloseMsg{} }
		case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
			if e.selected < len(e.status)-1 {
				e.selected++
				e.updateContent()
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
			if e.selected > 0 {
				e.selected--
				e.updateContent()
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		e.spinner, cmd = e.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case externalStatusLoadedMsg:
		e.loading = false
		if msg.err != nil {
			// Handle error
			e.status = nil
		} else {
			e.status = msg.status
		}
		e.updateContent()

	case tea.MouseMsg:
		var cmd tea.Cmd
		e.viewport, cmd = e.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return e, tea.Batch(cmds...)
}

// View renders the external dependencies view
func (e *ExternalView) View() string {
	if !e.ready {
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
		Width(e.width - 4).
		Height(e.height - 4)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	var content string
	if e.loading {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("🔗 External Dependencies"),
			"",
			e.spinner.View()+" Loading status...",
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("🔗 External Dependencies"),
			"",
			e.viewport.View(),
			"",
			hintStyle.Render("↑/↓ Navigate • ESC Close"),
		)
	}

	dialog := borderStyle.Render(content)

	return lipgloss.Place(
		e.width,
		e.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (e *ExternalView) updateContent() {
	if len(e.status) == 0 {
		e.viewport.SetContent("No external dependencies configured.")
		return
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ui.SubtleColor)
	selectedStyle := lipgloss.NewStyle().Background(lipgloss.Color("#333333"))

	for i, s := range e.status {
		// Status icon
		var icon string
		switch s.Status {
		case "installed":
			icon = okStyle.Render("✓")
		case "missing":
			icon = warnStyle.Render("○")
		case "skipped":
			icon = descStyle.Render("⊘")
		default:
			icon = errStyle.Render("?")
		}

		// Build line - use Dep.Name from the embedded ExternalDep
		name := s.Dep.Name
		if name == "" {
			name = s.Dep.ID
		}
		line := fmt.Sprintf("%s %s", icon, nameStyle.Render(name))
		if i == e.selected {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)

		// URL and destination
		lines = append(lines, descStyle.Render("   URL: "+s.Dep.URL))
		lines = append(lines, descStyle.Render("   Dest: "+s.Dep.Destination))

		// Status message
		var statusMsg string
		switch s.Status {
		case "installed":
			statusMsg = okStyle.Render("Installed")
		case "missing":
			statusMsg = warnStyle.Render("Not cloned")
		case "skipped":
			statusMsg = descStyle.Render("Skipped (platform mismatch)")
		}
		lines = append(lines, "   Status: "+statusMsg)
		lines = append(lines, "")
	}

	// Summary
	var installed, missing, skipped int
	for _, s := range e.status {
		switch s.Status {
		case "installed":
			installed++
		case "missing":
			missing++
		case "skipped":
			skipped++
		}
	}

	summaryStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(ui.SubtleColor).
		PaddingTop(1)

	summary := fmt.Sprintf("Total: %d | Installed: %d | Missing: %d | Skipped: %d",
		len(e.status), installed, missing, skipped)
	lines = append(lines, summaryStyle.Render(summary))

	e.viewport.SetContent(strings.Join(lines, "\n"))
}
