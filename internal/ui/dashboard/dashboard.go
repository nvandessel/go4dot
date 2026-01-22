package dashboard

import (
	"strings"

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
	state           State
	width           int
	height          int
	quitting        bool
	result          *Result
	filterMode      bool
	filterText      string
	selectedConfigs map[string]bool

	// Components
	header  Header
	summary Summary
	sidebar Sidebar
	details Details
	footer  Footer
}

// New creates a new dashboard model.
func New(s State) Model {
	m := Model{
		state:           s,
		selectedConfigs: make(map[string]bool),
	}
	m.header = NewHeader(s)
	m.summary = NewSummary(s)
	m.sidebar = NewSidebar(s, m.selectedConfigs)
	m.details = NewDetails(s)
	m.footer = NewFooter()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterMode {
			switch {
			case key.Matches(msg, keys.Quit): // esc
				m.filterMode = false
			case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
				m.filterMode = false
			case key.Matches(msg, key.NewBinding(key.WithKeys("backspace"))):
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
				}
			default:
				m.filterText += msg.String()
			}
			m.updateFilter()
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			m.result = &Result{Action: ActionQuit}
			return m, tea.Quit
		case key.Matches(msg, keys.Filter):
			m.filterMode = true
			return m, nil
		case key.Matches(msg, keys.Sync):
			m.result = &Result{Action: ActionSync}
			return m, tea.Quit
		case key.Matches(msg, keys.Doctor):
			m.result = &Result{Action: ActionDoctor}
			return m, tea.Quit
		case key.Matches(msg, keys.Install):
			m.result = &Result{Action: ActionInstall}
			return m, tea.Quit
		case key.Matches(msg, keys.Machine):
			m.result = &Result{Action: ActionMachineConfig}
			return m, tea.Quit
		case key.Matches(msg, keys.Update):
			m.result = &Result{Action: ActionUpdate}
			return m, tea.Quit
		case key.Matches(msg, keys.Menu):
			m.result = &Result{Action: ActionList}
			return m, tea.Quit
		case key.Matches(msg, keys.Enter):
			m.result = &Result{Action: ActionSyncConfig, ConfigName: m.state.Configs[m.sidebar.selectedIdx].Name}
			return m, tea.Quit
		case key.Matches(msg, keys.Select):
			cfgName := m.state.Configs[m.sidebar.selectedIdx].Name
			if m.selectedConfigs[cfgName] {
				delete(m.selectedConfigs, cfgName)
			} else {
				m.selectedConfigs[cfgName] = true
			}
		case key.Matches(msg, keys.All):
			allSelected := true
			for _, idx := range m.sidebar.filteredIdxs {
				if !m.selectedConfigs[m.state.Configs[idx].Name] {
					allSelected = false
					break
				}
			}
			if allSelected {
				for _, idx := range m.sidebar.filteredIdxs {
					delete(m.selectedConfigs, m.state.Configs[idx].Name)
				}
			} else {
				for _, idx := range m.sidebar.filteredIdxs {
					m.selectedConfigs[m.state.Configs[idx].Name] = true
				}
			}
		case key.Matches(msg, keys.Bulk):
			if len(m.selectedConfigs) > 0 {
				names := make([]string, 0, len(m.selectedConfigs))
				for name := range m.selectedConfigs {
					names = append(names, name)
				}
				m.result = &Result{Action: ActionBulkSync, ConfigNames: names}
				return m, tea.Quit
			}
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

	cmd = m.sidebar.Update(msg)
	cmds = append(cmds, cmd)

	m.details.selectedIdx = m.sidebar.selectedIdx

	return m, tea.Batch(cmds...)
}

func (m *Model) updateFilter() {
	filtered := []int{}
	if m.filterText == "" {
		for i := range m.state.Configs {
			filtered = append(filtered, i)
		}
	} else {
		for i, cfg := range m.state.Configs {
			if strings.Contains(strings.ToLower(cfg.Name), strings.ToLower(m.filterText)) {
				filtered = append(filtered, i)
			}
		}
	}
	m.sidebar.filteredIdxs = filtered
	if len(filtered) > 0 {
		m.sidebar.selectedIdx = filtered[0]
	}
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
	p := tea.NewProgram(&m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	return finalModel.(*Model).result, nil
}
