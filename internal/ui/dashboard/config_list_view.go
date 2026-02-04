package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/ui"
)

// ConfigListViewCloseMsg is sent when the config list view should close
type ConfigListViewCloseMsg struct{}

// ConfigListView displays a full list of configurations
type ConfigListView struct {
	configs  []config.ConfigItem
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// NewConfigListView creates a new config list view
func NewConfigListView(configs []config.ConfigItem) *ConfigListView {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()
	return &ConfigListView{
		configs:  configs,
		viewport: vp,
	}
}

// Init initializes the config list view
func (c *ConfigListView) Init() tea.Cmd {
	return nil
}

// SetSize updates the view dimensions
func (c *ConfigListView) SetSize(width, height int) {
	c.width = width
	c.height = height
	// Account for title and borders
	contentWidth := width - 6
	contentHeight := height - 6
	if contentWidth < 10 {
		contentWidth = 10
	}
	if contentHeight < 5 {
		contentHeight = 5
	}
	c.viewport.Width = contentWidth
	c.viewport.Height = contentHeight
	c.ready = true
	c.updateContent()
}

// Update handles messages
func (c *ConfigListView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "q"))):
			return c, func() tea.Msg { return ConfigListViewCloseMsg{} }
		}
	case tea.MouseMsg:
		c.viewport, cmd = c.viewport.Update(msg)
		return c, cmd
	}

	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

// View renders the config list
func (c *ConfigListView) View() string {
	if !c.ready {
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
		Width(c.width - 4).
		Height(c.height - 4)

	hintStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render("📦 Configuration List"),
		"",
		c.viewport.View(),
		"",
		hintStyle.Render("Press ESC or q to close"),
	)

	dialog := borderStyle.Render(content)

	return lipgloss.Place(
		c.width,
		c.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (c *ConfigListView) updateContent() {
	if len(c.configs) == 0 {
		c.viewport.SetContent("No configurations found.")
		return
	}

	var lines []string

	headerStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true)

	nameStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	platformStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor).
		Italic(true)

	// Group by type (core vs optional)
	// For now, just list all configs
	lines = append(lines, headerStyle.Render("Core Configurations"))
	lines = append(lines, strings.Repeat("─", c.viewport.Width-2))

	for i, cfg := range c.configs {
		// Config name
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, nameStyle.Render(cfg.Name)))

		// Description
		if cfg.Description != "" {
			lines = append(lines, "   "+descStyle.Render(cfg.Description))
		}

		// Path
		lines = append(lines, "   "+descStyle.Render("Path: "+cfg.Path))

		// Platforms
		if len(cfg.Platforms) > 0 {
			platforms := strings.Join(cfg.Platforms, ", ")
			lines = append(lines, "   "+platformStyle.Render("Platforms: "+platforms))
		}

		// Dependencies
		if len(cfg.DependsOn) > 0 {
			deps := strings.Join(cfg.DependsOn, ", ")
			lines = append(lines, "   "+descStyle.Render("Depends on: "+deps))
		}

		lines = append(lines, "")
	}

	c.viewport.SetContent(strings.Join(lines, "\n"))
}
