package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/platform"
)

// MenuAction represents the action to take
type MenuAction int

const (
	ActionInstall MenuAction = iota
	ActionUpdate
	ActionDoctor
	ActionList
	ActionInit
	ActionQuit
)

type item struct {
	title, desc string
	action      MenuAction
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	list     list.Model
	choice   *MenuAction
	platform *platform.Platform
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			*m.choice = ActionQuit
			return m, tea.Quit
		}
		if msg.String() == "enter" {
			i, ok := m.list.SelectedItem().(item)
			if ok {
				*m.choice = i.action
				return m, tea.Quit
			}
		}
	case tea.WindowSizeMsg:
		h, v := BoxStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return BoxStyle.Render(m.list.View())
}

// RunInteractiveMenu starts the interactive dashboard
func RunInteractiveMenu() (MenuAction, error) {
	// Detect platform for display info
	p, _ := platform.Detect()

	// Load config just to see if it exists (optional check)
	_, _, err := config.LoadFromDiscovery()
	hasConfig := err == nil

	items := []list.Item{
		item{title: "🚀 Install", desc: "Run the installation wizard", action: ActionInstall},
		item{title: "🔄 Update", desc: "Update dotfiles and external dependencies", action: ActionUpdate},
		item{title: "🩺 Doctor", desc: "Check system health and installation status", action: ActionDoctor},
		item{title: "📝 List", desc: "View installed and available configurations", action: ActionList},
	}

	if !hasConfig {
		// If no config, prioritize Init
		items = append([]list.Item{
			item{title: "✨ Init", desc: "Initialize a new .go4dot.yaml from existing dotfiles", action: ActionInit},
		}, items...)
	} else {
		items = append(items, item{title: "✨ Init", desc: "Re-initialize/update .go4dot.yaml", action: ActionInit})
	}

	items = append(items, item{title: "🚪 Quit", desc: "Exit go4dot", action: ActionQuit})

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.Title = "go4dot Dashboard"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	// Add header info
	if p != nil {
		l.Title += fmt.Sprintf(" • %s (%s)", p.OS, p.PackageManager)
	}

	choice := ActionQuit
	m := model{list: l, choice: &choice, platform: p}

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return ActionQuit, err
	}

	return choice, nil
}
