package dashboard

import "github.com/charmbracelet/bubbles/key"

// keyMap defines the key bindings
type keyMap struct {
	// Actions
	Sync    key.Binding
	Doctor  key.Binding
	Install key.Binding
	Machine key.Binding
	Update  key.Binding
	Menu    key.Binding
	Quit    key.Binding
	Enter   key.Binding
	Expand  key.Binding
	Filter  key.Binding
	Help    key.Binding
	Select  key.Binding
	All     key.Binding
	Bulk    key.Binding

	// List navigation (within panel)
	Up   key.Binding
	Down key.Binding

	// Panel navigation
	PanelNext  key.Binding
	PanelPrev  key.Binding
	PanelLeft  key.Binding
	PanelRight key.Binding
	PanelUp    key.Binding
	PanelDown  key.Binding

	// Direct panel jump (0-6)
	Panel0 key.Binding // Output/Console
	Panel1 key.Binding // Summary
	Panel2 key.Binding // Health
	Panel3 key.Binding // Overrides
	Panel4 key.Binding // External
	Panel5 key.Binding // Configs
	Panel6 key.Binding // Details
}

var keys = keyMap{
	// Actions
	Sync: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sync all"),
	),
	Doctor: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "health"),
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
		key.WithKeys("`"),
		key.WithHelp("`", "menu"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "esc", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "action"),
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
		key.WithHelp("A", "select all"),
	),
	Bulk: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "sync selected"),
	),

	// List navigation (within panel)
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),

	// Panel navigation
	PanelNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next panel"),
	),
	PanelPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev panel"),
	),
	PanelLeft: key.NewBinding(
		key.WithKeys("ctrl+h"),
		key.WithHelp("ctrl+h", "panel left"),
	),
	PanelRight: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "panel right"),
	),
	PanelUp: key.NewBinding(
		key.WithKeys("ctrl+k"),
		key.WithHelp("ctrl+k", "panel up"),
	),
	PanelDown: key.NewBinding(
		key.WithKeys("ctrl+j"),
		key.WithHelp("ctrl+j", "panel down"),
	),

	// Direct panel jump (0=output, 1-6 for others)
	Panel0: key.NewBinding(
		key.WithKeys("0"),
		key.WithHelp("0", "output"),
	),
	Panel1: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "summary"),
	),
	Panel2: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "health"),
	),
	Panel3: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "overrides"),
	),
	Panel4: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "external"),
	),
	Panel5: key.NewBinding(
		key.WithKeys("5"),
		key.WithHelp("5", "configs"),
	),
	Panel6: key.NewBinding(
		key.WithKeys("6"),
		key.WithHelp("6", "details"),
	),
}
