package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// State holds all the shared data for the dashboard.
// It is passed to components to ensure they have the data they need to render.
type State struct {
	Platform       *platform.Platform
	DriftSummary   *stow.DriftSummary
	LinkStatus     map[string]*stow.ConfigLinkStatus
	MachineStatus  []MachineStatus
	Configs        []config.ConfigItem
	DotfilesPath   string
	UpdateMsg      string
	HasBaseline    bool
	FilterText     string
	SelectedConfig string
}

// Model is the main container for the dashboard.
// It holds all the sub-components and manages the overall layout and state.
type Model struct {
	state    State
	width    int
	height   int
	quitting bool
	result   *Result

	// Components
	header  Header
	summary Summary
	sidebar Sidebar
	details Details
	footer  Footer
}

// New creates a new dashboard model.
func New(s State) Model {
	return Model{
		state:   s,
		header:  NewHeader(s),
		summary: NewSummary(s),
		sidebar: NewSidebar(s),
		details: NewDetails(s),
		footer:  NewFooter(),
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
			m.quitting = true
			m.result = &Result{Action: ActionQuit}
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.sidebar.width = msg.Width / 3
		m.sidebar.height = msg.Height - 10 // placeholder
		m.details.width = msg.Width - m.sidebar.width
		m.details.height = m.sidebar.height
		m.footer.width = msg.Width
	}

	// TODO: Update components
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	sidebarView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.SubtleColor).
		Width(m.sidebar.width - 2).
		Height(m.sidebar.height).
		Render(m.sidebar.View())

	detailsView := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Width(m.details.width-2).
		Height(m.details.height).
		Padding(0, 1).
		Render(m.details.View())

	mainContent := lipgloss.JoinHorizontal(
		lipgloss.Top,
		sidebarView,
		detailsView,
	)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.header.View(),
		m.summary.View(),
		mainContent,
		m.footer.View(),
	)
}

// Result is returned when the dashboard exits
type Result struct {
	Action         Action
	ConfigName     string
	ConfigNames    []string
	FilterText     string
	SelectedConfig string
}

// Action represents a user action from the dashboard
type Action int

const (
	ActionNone Action = iota
	ActionSync
	ActionSyncConfig
	ActionDoctor
	ActionInstall
	ActionMachineConfig
	ActionExternal
	ActionUninstall
	ActionUpdate
	ActionList
	ActionInit
	ActionQuit
	ActionBulkSync
)

// MachineStatus represents the status of a machine config for the dashboard
type MachineStatus struct {
	ID          string
	Description string
	Status      string // "configured", "missing", "error"
}

// Run starts the dashboard and returns the selected action
func Run(s State) (*Result, error) {
	m := New(s)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(Model).result, nil
}
