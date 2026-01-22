package dashboard

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/ui"
)

type menuItem struct {
	title, desc string
	action      Action
}

func (i menuItem) Title() string       { return i.title }
func (i menuItem) Description() string { return i.desc }
func (i menuItem) FilterValue() string { return i.title }

type Menu struct {
	list   list.Model
	width  int
	height int
}

func NewMenu() Menu {
	items := []list.Item{
		menuItem{title: "List Configs", desc: "View all configurations in a simple list", action: ActionList},
		menuItem{title: "External Dependencies", desc: "Manage external git repositories", action: ActionExternal},
		menuItem{title: "Uninstall go4dot", desc: "Remove all symlinks and state", action: ActionUninstall},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "More Commands"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return Menu{list: l}
}

func (m Menu) Init() tea.Cmd {
	return nil
}

func (m *Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Menu) View() string {
	return ui.BoxStyle.Render(m.list.View())
}
