package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type errMsg error

type spinnerModel struct {
	spinner  spinner.Model
	quitting bool
	err      error
	message  string
	action   func() error
}

func initialSpinnerModel(msg string, action func() error) spinnerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return spinnerModel{
		spinner: s,
		message: msg,
		action:  action,
	}
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			if err := m.action(); err != nil {
				return errMsg(err)
			}
			return nil
		},
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "esc" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case errMsg:
		m.err = msg
		return m, tea.Quit
	case nil: // Action completed successfully
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.err != nil {
		return ErrorStyle.Render("✖") + " " + m.message + ": " + m.err.Error() + "\n"
	}
	if m.quitting {
		return ""
	}
	str := fmt.Sprintf("%s %s...", m.spinner.View(), m.message)
	return str
}

// RunSpinner runs a task with a spinner
func RunSpinner(msg string, action func() error) error {
	p := tea.NewProgram(initialSpinnerModel(msg, action))
	m, err := p.Run()
	if err != nil {
		return err
	}
	if model, ok := m.(spinnerModel); ok && model.err != nil {
		return model.err
	}

	// Print success (replace spinner line)
	// We rely on the caller or the model view logic for this,
	// but usually we want to keep the success message
	Success(msg + " Done")
	return nil
}
