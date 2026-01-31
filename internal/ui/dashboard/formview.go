package dashboard

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/ui"
)

// FormCompleteMsg is sent when a form is completed successfully
type FormCompleteMsg struct {
	FormID string
}

// FormCancelMsg is sent when a form is cancelled
type FormCancelMsg struct {
	FormID string
}

// FormView wraps a huh.Form to embed it within the Bubble Tea dashboard.
// It handles form completion, cancellation, and styling.
type FormView struct {
	id       string
	title    string
	form     *huh.Form
	width    int
	height   int
	finished bool
	canceled bool
}

// NewFormView creates a new FormView wrapper around a huh.Form.
// The id is used to identify the form in completion/cancellation messages.
func NewFormView(id, title string, form *huh.Form) *FormView {
	return &FormView{
		id:    id,
		title: title,
		form:  form,
	}
}

// Init initializes the FormView
func (f *FormView) Init() tea.Cmd {
	return f.form.Init()
}

// SetSize updates the form dimensions
func (f *FormView) SetSize(width, height int) {
	f.width = width
	f.height = height
	// huh.Form doesn't have a direct SetSize method, but we can use WithWidth
	// We need to recreate or configure the form with the proper width
	f.form = f.form.WithWidth(width - 4) // Account for padding/borders
	f.form = f.form.WithHeight(height - 4)
}

// Update handles messages for the FormView
func (f *FormView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle escape to cancel
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(keyMsg, key.NewBinding(key.WithKeys("ctrl+c"))) {
			f.canceled = true
			return f, func() tea.Msg { return FormCancelMsg{FormID: f.id} }
		}
	}

	// Forward to the form
	form, cmd := f.form.Update(msg)
	if m, ok := form.(*huh.Form); ok {
		f.form = m
	}

	// Check if form is complete
	if f.form.State == huh.StateCompleted {
		f.finished = true
		return f, func() tea.Msg { return FormCompleteMsg{FormID: f.id} }
	}

	// Check if form was aborted (user pressed Escape within huh)
	if f.form.State == huh.StateAborted {
		f.canceled = true
		return f, func() tea.Msg { return FormCancelMsg{FormID: f.id} }
	}

	return f, cmd
}

// View renders the FormView with a styled border and title
func (f *FormView) View() string {
	if f.width < 10 || f.height < 5 {
		return ""
	}

	// Title style
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.PrimaryColor).
		Bold(true).
		Padding(0, 1)

	// Container style with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.PrimaryColor).
		Padding(1, 2).
		Width(f.width - 4).
		Height(f.height - 4)

	// Build view
	title := titleStyle.Render(f.title)
	content := f.form.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		containerStyle.Render(content),
	)
}

// IsFinished returns true if the form completed successfully
func (f *FormView) IsFinished() bool {
	return f.finished
}

// IsCanceled returns true if the form was canceled
func (f *FormView) IsCanceled() bool {
	return f.canceled
}

// ID returns the form identifier
func (f *FormView) ID() string {
	return f.id
}

// FormViewOverlay renders a FormView as a centered overlay on the dashboard
type FormViewOverlay struct {
	formView *FormView
	width    int
	height   int
}

// NewFormViewOverlay creates a new overlay wrapper for a FormView
func NewFormViewOverlay(fv *FormView) *FormViewOverlay {
	return &FormViewOverlay{
		formView: fv,
	}
}

// SetSize updates the overlay dimensions and propagates to the form
func (o *FormViewOverlay) SetSize(width, height int) {
	o.width = width
	o.height = height

	// Form takes up 60% of width and 70% of height, centered
	formWidth := width * 60 / 100
	if formWidth < 40 {
		formWidth = 40
	}
	if formWidth > width-10 {
		formWidth = width - 10
	}

	formHeight := height * 70 / 100
	if formHeight < 15 {
		formHeight = 15
	}
	if formHeight > height-6 {
		formHeight = height - 6
	}

	o.formView.SetSize(formWidth, formHeight)
}

// Update forwards messages to the underlying FormView
func (o *FormViewOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		o.SetSize(msg.Width, msg.Height)
	}
	return o.formView.Update(msg)
}

// View renders the form as a centered overlay with a dimmed background
func (o *FormViewOverlay) View() string {
	// Create dimmed background effect
	bgStyle := lipgloss.NewStyle().
		Foreground(ui.SubtleColor)

	// Render form content
	formContent := o.formView.View()

	// Center the form
	return lipgloss.Place(
		o.width,
		o.height,
		lipgloss.Center,
		lipgloss.Center,
		formContent,
		lipgloss.WithWhitespaceChars("░"),
		lipgloss.WithWhitespaceForeground(bgStyle.GetForeground()),
	)
}

// FormView returns the underlying FormView
func (o *FormViewOverlay) FormView() *FormView {
	return o.formView
}
