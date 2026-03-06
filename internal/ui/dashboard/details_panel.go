package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/doctor"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// DetailsContext indicates what panel the details should show info for
type DetailsContext int

const (
	DetailsContextConfigs DetailsContext = iota
	DetailsContextHealth
	DetailsContextOverrides
	DetailsContextExternal
)

// DetailsPanel displays expanded info for the focused panel's selected item
// This is a scrollable panel when focused
type DetailsPanel struct {
	BasePanel
	state    State
	viewport viewport.Model
	ready    bool

	// Context determines what to display
	context DetailsContext

	// Data from various panels
	configsPanel   *ConfigsPanel
	healthPanel    *HealthPanel
	overridesPanel *OverridesPanel
	externalPanel  *ExternalPanel
}

// NewDetailsPanel creates a new details panel
func NewDetailsPanel(state State) *DetailsPanel {
	vp := viewport.New(0, 0)
	vp.Style = lipgloss.NewStyle()

	return &DetailsPanel{
		BasePanel: NewBasePanel(PanelDetails, "6 Details"),
		state:     state,
		viewport:  vp,
		context:   DetailsContextConfigs,
	}
}

// Init implements Panel interface
func (p *DetailsPanel) Init() tea.Cmd {
	return nil
}

// SetSize implements Panel interface
func (p *DetailsPanel) SetSize(width, height int) {
	p.BasePanel.SetSize(width, height)

	contentWidth := p.ContentWidth()
	contentHeight := p.ContentHeight()

	p.viewport.Width = contentWidth
	p.viewport.Height = contentHeight
	p.ready = true
	p.updateContent()
}

// Update implements Panel interface
func (p *DetailsPanel) Update(msg tea.Msg) tea.Cmd {
	if !p.ready {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		if p.focused {
			p.viewport, cmd = p.viewport.Update(msg)
			return cmd
		}
	case tea.KeyMsg:
		if p.focused {
			p.viewport, cmd = p.viewport.Update(msg)
			return cmd
		}
	}

	return nil
}

// View implements Panel interface
func (p *DetailsPanel) View() string {
	if !p.ready {
		return ""
	}
	return p.viewport.View()
}

// GetSelectedItem implements Panel interface - details doesn't have selection
func (p *DetailsPanel) GetSelectedItem() *SelectedItem {
	return nil
}

// SetContext sets what panel's content to display details for
func (p *DetailsPanel) SetContext(ctx DetailsContext) {
	p.context = ctx
	p.updateContent()
}

// SetPanels sets references to other panels for context-aware display
func (p *DetailsPanel) SetPanels(configs *ConfigsPanel, health *HealthPanel, overrides *OverridesPanel, external *ExternalPanel) {
	p.configsPanel = configs
	p.healthPanel = health
	p.overridesPanel = overrides
	p.externalPanel = external
}

// RefreshContent updates the content based on current context
func (p *DetailsPanel) RefreshContent() {
	p.updateContent()
}

func (p *DetailsPanel) updateContent() {
	var content string

	switch p.context {
	case DetailsContextHealth:
		content = p.renderHealthDetails()
	case DetailsContextOverrides:
		content = p.renderOverridesDetails()
	case DetailsContextExternal:
		content = p.renderExternalDetails()
	default:
		content = p.renderConfigDetails()
	}

	p.viewport.SetContent(content)
}

func (p *DetailsPanel) renderConfigDetails() string {
	if p.configsPanel == nil {
		return ui.SubtleStyle.Render("No config selected")
	}

	cfg := p.configsPanel.GetSelectedConfig()
	if cfg == nil {
		return ui.SubtleStyle.Render("No config selected")
	}

	linkStatus := p.state.LinkStatus[cfg.Name]

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle
	subtleStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle
	titleStyle := lipgloss.NewStyle().
		Foreground(ui.TextColor).
		Bold(true).
		Background(ui.PrimaryColor).
		Padding(0, 1)

	title := titleStyle.Render(strings.ToUpper(cfg.Name))
	lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Center, title))
	lines = append(lines, "")

	if cfg.Description != "" {
		descStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Italic(true).Width(p.ContentWidth())
		lines = append(lines, descStyle.Render(cfg.Description))
		lines = append(lines, "")
	}

	// Show source and destination paths
	if linkStatus != nil || cfg.Path != "" {
		lines = append(lines, headerStyle.Render("PATHS"))
		pathStyle := lipgloss.NewStyle().Foreground(ui.TextColor)
		if cfg.Path != "" {
			lines = append(lines, fmt.Sprintf("%s %s",
				subtleStyle.Render("Source:"),
				pathStyle.Render(filepath.Join(p.state.DotfilesPath, cfg.Path))))
		}
		home := os.Getenv("HOME")
		if home != "" {
			lines = append(lines, fmt.Sprintf("%s %s",
				subtleStyle.Render("Dest:  "),
				pathStyle.Render(home)))
		}
		lines = append(lines, "")
	}

	// Get drift result for enhanced display
	var driftResult *stow.DriftResult
	if p.state.DriftSummary != nil {
		driftResult = p.state.DriftSummary.ResultByName(cfg.Name)
	}

	// Show drift status summary
	if driftResult != nil && (driftResult.HasDrift || len(driftResult.OrphanFiles) > 0) {
		lines = append(lines, headerStyle.Render("DRIFT STATUS"))
		var driftParts []string
		if len(driftResult.NewFiles) > 0 {
			driftParts = append(driftParts, okStyle.Render(fmt.Sprintf("+%d new", len(driftResult.NewFiles))))
		}
		if len(driftResult.MissingFiles) > 0 {
			driftParts = append(driftParts, errStyle.Render(fmt.Sprintf("-%d missing", len(driftResult.MissingFiles))))
		}
		if len(driftResult.ConflictFiles) > 0 {
			conflictText := fmt.Sprintf("!%d conflicts", len(driftResult.ConflictFiles))
			if len(driftResult.ContentDriftFiles) > 0 {
				conflictText += fmt.Sprintf(" (%d content differs)", len(driftResult.ContentDriftFiles))
			}
			driftParts = append(driftParts, warnStyle.Render(conflictText))
		}
		if len(driftResult.OrphanFiles) > 0 {
			driftParts = append(driftParts, subtleStyle.Render(fmt.Sprintf("?%d untracked", len(driftResult.OrphanFiles))))
		}
		lines = append(lines, "  "+strings.Join(driftParts, ", "))
		lines = append(lines, "")
	}

	if linkStatus != nil {
		linked := linkStatus.LinkedCount
		total := linkStatus.TotalCount
		var statsTag string
		if linked == total {
			statsTag = okStyle.Render(fmt.Sprintf(" [%d/%d ✓]", linked, total))
		} else {
			statsTag = warnStyle.Render(fmt.Sprintf(" [%d/%d]", linked, total))
		}
		lines = append(lines, headerStyle.Render("FILESYSTEM MAPPINGS")+statsTag)

		// Build and render file tree with proper connectors
		tree := buildFileTree(linkStatus.Files)

		// Augment tree with drift information
		if driftResult != nil {
			addOrphansToTree(tree, driftResult.OrphanFiles)
			markContentDriftInTree(tree, driftResult.ContentDriftFiles)
		}

		treeLines := renderFileTree(tree, "", true, okStyle, warnStyle, errStyle, subtleStyle)
		lines = append(lines, treeLines...)
		lines = append(lines, "")
	}

	if len(cfg.DependsOn) > 0 {
		lines = append(lines, headerStyle.Render("MODULE DEPENDENCIES"))
		for _, depName := range cfg.DependsOn {
			status := subtleStyle.Render("(unknown)")
			if p.state.LinkStatus != nil {
				if depStatus, ok := p.state.LinkStatus[depName]; ok {
					if depStatus.IsFullyLinked() {
						status = okStyle.Render("(✓ linked)")
					} else {
						status = warnStyle.Render("(✗ missing)")
					}
				}
			}
			lines = append(lines, fmt.Sprintf("  %s %s", depName, status))
		}
		lines = append(lines, "")
	}

	if len(cfg.ExternalDeps) > 0 {
		lines = append(lines, headerStyle.Render("EXTERNAL REPOSITORIES"))
		for i, extDep := range cfg.ExternalDeps {
			connector := "├─"
			if i == len(cfg.ExternalDeps)-1 {
				connector = "└─"
			}
			lines = append(lines, fmt.Sprintf("  %s %s", subtleStyle.Render(connector), extDep.URL))
			indent := "│ "
			if i == len(cfg.ExternalDeps)-1 {
				indent = "  "
			}
			lines = append(lines, subtleStyle.Render(fmt.Sprintf("  %s └─ %s", indent, extDep.Destination)))
		}
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// fileTreeNode represents a node in the file tree (either a directory or file)
type fileTreeNode struct {
	name            string
	isDir           bool
	isLinked        bool
	issue           string
	isOrphan        bool // File in dest not tracked by source
	hasContentDrift bool // Conflict file with different content from source
	children        map[string]*fileTreeNode
}

// buildFileTree creates a tree structure from flat file paths
func buildFileTree(files []stow.FileStatus) *fileTreeNode {
	root := &fileTreeNode{
		name:     "/",
		isDir:    true,
		children: make(map[string]*fileTreeNode),
	}

	for _, f := range files {
		parts := strings.Split(f.RelPath, string(filepath.Separator))
		current := root

		for i, part := range parts {
			if part == "" {
				continue
			}

			isLast := i == len(parts)-1

			if current.children == nil {
				current.children = make(map[string]*fileTreeNode)
			}

			child, exists := current.children[part]
			if !exists {
				child = &fileTreeNode{
					name:     part,
					isDir:    !isLast,
					children: make(map[string]*fileTreeNode),
				}
				current.children[part] = child
			}

			if isLast {
				child.isLinked = f.IsLinked
				child.issue = f.Issue
				child.isDir = false
			}

			current = child
		}
	}

	return root
}

// renderFileTree renders the tree structure with proper tree connectors (├─, └─, │)
func renderFileTree(node *fileTreeNode, prefix string, isRoot bool, okStyle, warnStyle, errStyle, subtleStyle lipgloss.Style) []string {
	var lines []string

	// Sort children: directories first, then files, both alphabetically
	var dirs, files []string
	for name, child := range node.children {
		if child.isDir {
			dirs = append(dirs, name)
		} else {
			files = append(files, name)
		}
	}
	sort.Strings(dirs)
	sort.Strings(files)

	// Combine into ordered list for proper connector rendering
	allNames := append(dirs, files...)
	totalChildren := len(allNames)

	for i, name := range allNames {
		child := node.children[name]
		isLast := i == totalChildren-1

		// Choose connector
		connector := "├─"
		if isLast {
			connector = "└─"
		}

		// Choose continuation prefix for children
		childPrefix := prefix + "│  "
		if isLast {
			childPrefix = prefix + "   "
		}

		// Root-level items get a leading indent for visual padding
		linePrefix := prefix
		if isRoot {
			linePrefix = "  "
			childPrefix = "  " + childPrefix[len(prefix):]
		}

		if child.isDir {
			// Directory node
			dirLabel := subtleStyle.Render(connector) + " " + subtleStyle.Render(name+"/")
			lines = append(lines, linePrefix+dirLabel)
			childLines := renderFileTree(child, childPrefix, false, okStyle, warnStyle, errStyle, subtleStyle)
			lines = append(lines, childLines...)
		} else {
			// File node - choose status icon
			var icon string
			if child.isOrphan {
				icon = subtleStyle.Render("?")
			} else if child.isLinked {
				icon = okStyle.Render("✓")
			} else if child.hasContentDrift {
				icon = errStyle.Render("≠")
			} else if strings.Contains(strings.ToLower(child.issue), "conflict") ||
				strings.Contains(strings.ToLower(child.issue), "exists") ||
				strings.Contains(strings.ToLower(child.issue), "elsewhere") {
				icon = warnStyle.Render("⚠")
			} else {
				icon = errStyle.Render("✗")
			}

			lines = append(lines, linePrefix+subtleStyle.Render(connector)+" "+icon+" "+name)

			// Show issue description
			if child.isOrphan {
				lines = append(lines, subtleStyle.Render(childPrefix+"→ untracked (not in source)"))
			} else if !child.isLinked && child.issue != "" {
				issueText := child.issue
				if child.hasContentDrift {
					issueText += " [content differs]"
				}
				lines = append(lines, subtleStyle.Render(childPrefix+"→ "+issueText))
			}
		}
	}

	return lines
}

func (p *DetailsPanel) renderHealthDetails() string {
	if p.healthPanel == nil || p.healthPanel.IsLoading() {
		return ui.SubtleStyle.Render("Loading health checks...")
	}

	check := p.healthPanel.GetSelectedCheck()
	if check == nil {
		return ui.SubtleStyle.Render("No check selected")
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	skipStyle := ui.SubtleStyle
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	fixStyle := lipgloss.NewStyle().Foreground(ui.PrimaryColor).Italic(true)
	headerStyle := ui.HeaderStyle

	// Status icon
	var icon, statusText string
	switch check.Status {
	case doctor.StatusOK:
		icon = okStyle.Render("✓")
		statusText = okStyle.Render("Passed")
	case doctor.StatusWarning:
		icon = warnStyle.Render("⚠")
		statusText = warnStyle.Render("Warning")
	case doctor.StatusError:
		icon = errStyle.Render("✗")
		statusText = errStyle.Render("Error")
	case doctor.StatusSkipped:
		icon = skipStyle.Render("○")
		statusText = skipStyle.Render("Skipped")
	}

	lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(check.Name)))
	lines = append(lines, statusText)
	lines = append(lines, "")
	lines = append(lines, headerStyle.Render("DESCRIPTION"))
	lines = append(lines, descStyle.Render(check.Description))
	lines = append(lines, "")

	if check.Message != "" {
		lines = append(lines, headerStyle.Render("MESSAGE"))
		msgStyle := descStyle
		switch check.Status {
		case doctor.StatusError:
			msgStyle = errStyle
		case doctor.StatusWarning:
			msgStyle = warnStyle
		}
		lines = append(lines, msgStyle.Render(check.Message))
		lines = append(lines, "")
	}

	if check.Fix != "" {
		lines = append(lines, headerStyle.Render("FIX SUGGESTION"))
		lines = append(lines, fixStyle.Render(check.Fix))
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func (p *DetailsPanel) renderOverridesDetails() string {
	if p.overridesPanel == nil {
		return ui.SubtleStyle.Render("No machine config selected")
	}

	mc := p.overridesPanel.GetSelectedConfig()
	if mc == nil {
		return ui.SubtleStyle.Render("No machine config selected")
	}

	status := p.overridesPanel.GetMachineStatus()

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	errStyle := lipgloss.NewStyle().Foreground(ui.ErrorColor)
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle

	// Status icon and text
	var icon, statusText string
	switch status {
	case "configured":
		icon = okStyle.Render("✓")
		statusText = okStyle.Render("Configured")
	case "missing":
		icon = warnStyle.Render("○")
		statusText = warnStyle.Render("Not configured")
	case "error":
		icon = errStyle.Render("✗")
		statusText = errStyle.Render("Error")
	default:
		icon = descStyle.Render("?")
		statusText = descStyle.Render("Unknown")
	}

	lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(mc.Description)))
	lines = append(lines, statusText)
	lines = append(lines, "")

	lines = append(lines, headerStyle.Render("DESTINATION"))
	lines = append(lines, descStyle.Render(mc.Destination))
	lines = append(lines, "")

	if len(mc.Prompts) > 0 {
		lines = append(lines, headerStyle.Render("FIELDS"))
		for _, prompt := range mc.Prompts {
			reqMark := ""
			if prompt.Required {
				reqMark = " *"
			}
			lines = append(lines, fmt.Sprintf("• %s%s", prompt.Prompt, reqMark))
			if prompt.Type != "" {
				lines = append(lines, descStyle.Render(fmt.Sprintf("  Type: %s", prompt.Type)))
			}
			if prompt.Default != "" {
				lines = append(lines, descStyle.Render(fmt.Sprintf("  Default: %s", prompt.Default)))
			}
		}
		lines = append(lines, "")
	}

	lines = append(lines, descStyle.Render("Press Enter to configure"))

	return strings.Join(lines, "\n")
}

func (p *DetailsPanel) renderExternalDetails() string {
	if p.externalPanel == nil || p.externalPanel.IsLoading() {
		return ui.SubtleStyle.Render("Loading external dependencies...")
	}

	ext := p.externalPanel.GetSelectedExternal()
	if ext == nil {
		return ui.SubtleStyle.Render("No external dependency selected")
	}

	var lines []string

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := lipgloss.NewStyle().Foreground(ui.WarningColor)
	skipStyle := ui.SubtleStyle
	nameStyle := lipgloss.NewStyle().Foreground(ui.TextColor).Bold(true)
	descStyle := ui.SubtleStyle
	headerStyle := ui.HeaderStyle

	name := ext.Dep.Name
	if name == "" {
		name = ext.Dep.ID
	}

	// Status icon and text
	var icon, statusText string
	switch ext.Status {
	case "installed":
		icon = okStyle.Render("✓")
		statusText = okStyle.Render("Installed")
	case "missing":
		icon = warnStyle.Render("○")
		statusText = warnStyle.Render("Not cloned")
	case "skipped":
		icon = skipStyle.Render("⊘")
		statusText = skipStyle.Render("Skipped (platform mismatch)")
	default:
		icon = skipStyle.Render("?")
		statusText = skipStyle.Render("Unknown")
	}

	lines = append(lines, fmt.Sprintf("%s %s", icon, nameStyle.Render(name)))
	lines = append(lines, statusText)
	lines = append(lines, "")

	lines = append(lines, headerStyle.Render("URL"))
	lines = append(lines, descStyle.Render(ext.Dep.URL))
	lines = append(lines, "")

	lines = append(lines, headerStyle.Render("DESTINATION"))
	lines = append(lines, descStyle.Render(ext.Dep.Destination))
	lines = append(lines, "")

	switch ext.Status {
	case "missing":
		lines = append(lines, descStyle.Render("Press Enter to clone"))
	case "installed":
		lines = append(lines, descStyle.Render("Press Enter to update"))
	}

	return strings.Join(lines, "\n")
}

// addOrphansToTree adds orphan file nodes to the file tree
func addOrphansToTree(root *fileTreeNode, orphanFiles []string) {
	for _, orphanPath := range orphanFiles {
		parts := strings.Split(orphanPath, string(filepath.Separator))
		current := root

		for i, part := range parts {
			if part == "" {
				continue
			}
			isLast := i == len(parts)-1

			if current.children == nil {
				current.children = make(map[string]*fileTreeNode)
			}

			child, exists := current.children[part]
			if !exists {
				child = &fileTreeNode{
					name:     part,
					isDir:    !isLast,
					children: make(map[string]*fileTreeNode),
				}
				current.children[part] = child
			}

			if isLast {
				child.isOrphan = true
				child.isDir = false
			}

			current = child
		}
	}
}

// markContentDriftInTree marks nodes in the tree that have content drift
func markContentDriftInTree(root *fileTreeNode, contentDriftFiles []string) {
	if len(contentDriftFiles) == 0 {
		return
	}
	driftSet := make(map[string]bool, len(contentDriftFiles))
	for _, f := range contentDriftFiles {
		driftSet[f] = true
	}
	markContentDriftRecursive(root, "", driftSet)
}

func markContentDriftRecursive(node *fileTreeNode, prefix string, driftSet map[string]bool) {
	for name, child := range node.children {
		fullPath := name
		if prefix != "" {
			fullPath = filepath.Join(prefix, name)
		}
		if child.isDir {
			markContentDriftRecursive(child, fullPath, driftSet)
		} else if driftSet[fullPath] {
			child.hasContentDrift = true
		}
	}
}

// UpdateState updates the panel's state reference
func (p *DetailsPanel) UpdateState(state State) {
	p.state = state
	p.updateContent()
}
