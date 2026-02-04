package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// updateMenu handles messages when in the menu view
func (m *Model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			m.popView()
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			item, ok := m.menu.list.SelectedItem().(menuItem)
			if ok {
				return m.handleMenuAction(item.action)
			}
		}
	}

	var cmd tea.Cmd
	newMenu, cmd := m.menu.Update(msg)
	m.menu = newMenu.(*Menu)
	return m, cmd
}

// handleMenuAction handles the selected menu action
func (m *Model) handleMenuAction(action Action) (tea.Model, tea.Cmd) {
	switch action {
	case ActionList:
		m.configList = NewConfigListView(m.state.Configs)
		m.configList.SetSize(m.width, m.height)
		m.pushView(viewConfigList)
		return m, nil

	case ActionExternal:
		if m.state.Config == nil {
			return m, nil
		}
		m.externalView = NewExternalView(m.state.Config, m.state.DotfilesPath, m.state.Platform)
		m.externalView.SetSize(m.width, m.height)
		m.pushView(viewExternal)
		return m, m.externalView.Init()

	case ActionUninstall:
		m.confirm = NewConfirm(
			"uninstall",
			"Uninstall go4dot?",
			"This will remove all symlinks and state. This action cannot be undone.",
		).WithLabels("Yes, uninstall", "Cancel")
		m.confirm.SetSize(m.width, m.height)
		m.pushView(viewConfirm)
		return m, nil

	default:
		m.setResult(action)
		return m, tea.Quit
	}
}
