package dashboard

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nvandessel/go4dot/internal/ui"
)

const (
	// menuMaxWidth is the maximum width of the compact menu panel.
	menuMaxWidth = 46
	// menuCompactHeight is the height allocated to the list widget inside the
	// compact menu panel. The default delegate uses 2 lines per item (title +
	// description) plus 1 line spacing between items, plus the title header
	// area. We give a small amount of extra room so the list renders cleanly.
	menuCompactHeight = 14
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
	l.SetShowHelp(false)

	return Menu{list: l}
}

func (m Menu) Init() tea.Cmd {
	return nil
}

func (m *Menu) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Menu) View() string {
	return ui.BoxStyle.Render(m.list.View())
}

// CompactWidth returns the constrained width for the compact menu panel.
// It caps at menuMaxWidth but respects smaller terminal widths.
func CompactWidth(termWidth int) int {
	w := menuMaxWidth
	if termWidth > 0 && termWidth < w+10 {
		w = termWidth - 10
		if w < 20 {
			w = 20
		}
	}
	return w
}

// CompactHeight returns the constrained height for the compact menu panel.
// It caps at menuCompactHeight but respects smaller terminal heights.
func CompactHeight(termHeight int) int {
	h := menuCompactHeight
	if termHeight > 0 && termHeight < h+10 {
		h = termHeight - 10
		if h < 6 {
			h = 6
		}
	}
	return h
}

// SetSize sets the menu dimensions, clamping to compact panel bounds.
func (m *Menu) SetSize(width, height int) {
	w := CompactWidth(width)
	h := CompactHeight(height)
	m.width = w
	m.height = h
	m.list.SetSize(w, h)
}
