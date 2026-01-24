package dashboard

import "github.com/charmbracelet/bubbles/key"

// keyMap defines the key bindings
type keyMap struct {
	Sync    key.Binding
	Doctor  key.Binding
	Install key.Binding
	Machine key.Binding
	Update  key.Binding
	Menu    key.Binding
	Quit    key.Binding
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Expand  key.Binding
	Filter  key.Binding
	Help    key.Binding
	Select  key.Binding
	All     key.Binding
	Bulk    key.Binding
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
	Install: key.NewBinding(
		key.WithKeys("i"),
		key.WithHelp("i", "install"),
	),
	Machine: key.NewBinding(
		key.WithKeys("m"),
		key.WithHelp("m", "overrides"),
	),
	Update: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update"),
	),
	Menu: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "more"),
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
	Expand: key.NewBinding(
		key.WithKeys("e", "right"),
		key.WithHelp("e", "expand/collapse"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Select: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "select"),
	),
	All: key.NewBinding(
		key.WithKeys("A"),
		key.WithHelp("shift+a", "select all"),
	),
	Bulk: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("shift+s", "sync selected"),
	),
}
