package dashboard

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nvandessel/go4dot/internal/config"
	"github.com/nvandessel/go4dot/internal/stow"
	"github.com/nvandessel/go4dot/internal/ui"
)

// Sidebar is the model for the sidebar component.
type Sidebar struct {
	state        State
	width        int
	height       int
	selectedIdx  int
	listOffset   int // Scroll offset for the config list
	filteredIdxs []int
	selected     map[string]bool
}

// NewSidebar creates a new sidebar component.
func NewSidebar(s State, selected map[string]bool) Sidebar {
	filteredIdxs := make([]int, len(s.Configs))
	for i := range s.Configs {
		filteredIdxs[i] = i
	}
	return Sidebar{
		state:        s,
		selectedIdx:  0,
		listOffset:   0,
		filteredIdxs: filteredIdxs,
		selected:     selected,
	}
}

// Update handles messages for the sidebar.
func (s *Sidebar) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		currentPos := -1
		for i, idx := range s.filteredIdxs {
			if idx == s.selectedIdx {
				currentPos = i
				break
			}
		}

		switch {
		case key.Matches(msg, keys.Up):
			if currentPos > 0 {
				s.selectedIdx = s.filteredIdxs[currentPos-1]
				s.ensureVisible()
			}
		case key.Matches(msg, keys.Down):
			if currentPos < len(s.filteredIdxs)-1 {
				s.selectedIdx = s.filteredIdxs[currentPos+1]
				s.ensureVisible()
			}
		}
	}
	return nil
}

// ensureVisible ensures the selected item is within the visible area of the list
func (s *Sidebar) ensureVisible() {
	currentPos := -1
	for i, idx := range s.filteredIdxs {
		if idx == s.selectedIdx {
			currentPos = i
			break
		}
	}

	if s.height <= 0 || currentPos == -1 {
		return
	}
	if currentPos < s.listOffset {
		s.listOffset = currentPos
	} else if currentPos >= s.listOffset+s.height {
		s.listOffset = currentPos - s.height + 1
	}
}

// View renders the sidebar.
func (s Sidebar) View() string {
	var lines []string

	normalStyle := ui.TextStyle
	selectedStyle := ui.SelectedItemStyle.Width(s.width - 2)
	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)

	// Build a map of drift results for quick lookup
	driftMap := make(map[string]*stow.DriftResult)
	if s.state.DriftSummary != nil {
		for i := range s.state.DriftSummary.Results {
			r := &s.state.DriftSummary.Results[i]
			driftMap[r.ConfigName] = r
		}
	}

	// Calculate visible range
	endIdx := s.listOffset + s.height
	if endIdx > len(s.filteredIdxs) {
		endIdx = len(s.filteredIdxs)
	}

	for i := s.listOffset; i < endIdx; i++ {
		idx := s.filteredIdxs[i]
		cfg := s.state.Configs[idx]

		prefix := "  "
		if idx == s.selectedIdx {
			prefix = "> "
		}

		checkbox := "[ ]"
		if s.selected[cfg.Name] {
			checkbox = okStyle.Render("[✓]")
		}

		// Get link status for this config
		linkStatus := s.state.LinkStatus[cfg.Name]
		drift := driftMap[cfg.Name]

		// Get enhanced status info
		statusInfo := s.getConfigStatusInfo(cfg, linkStatus, drift)

		// Calculate name width
		nameWidth := s.width - 10
		if nameWidth < 5 {
			nameWidth = 5
		}
		name := cfg.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-3] + "..."
		} else {
			name = fmt.Sprintf("%-*s", nameWidth, name)
		}

		content := fmt.Sprintf("%s%s %s %s",
			prefix,
			checkbox,
			name,
			statusInfo.icon,
		)

		content = fmt.Sprintf("%-*s", s.width-2, content)

		if idx == s.selectedIdx {
			lines = append(lines, selectedStyle.Render(content))
		} else {
			lines = append(lines, normalStyle.Render(content))
		}
	}

	for len(lines) < s.height {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// configStatusInfo holds detailed status information for a config
type configStatusInfo struct {
	icon       string   // Primary status icon
	statusText string   // "X/Y" display
	statusTags []string // Additional status tags (conflicts, deps, external)
}

// getConfigStatusInfo analyzes a config and returns detailed status information
func (s Sidebar) getConfigStatusInfo(cfg config.ConfigItem, linkStatus *stow.ConfigLinkStatus, drift *stow.DriftResult) configStatusInfo {
	info := configStatusInfo{
		statusTags: []string{},
	}

	okStyle := lipgloss.NewStyle().Foreground(ui.SecondaryColor)
	warnStyle := ui.WarningStyle
	errStyle := ui.ErrorStyle

	if linkStatus != nil {
		conflictCount := 0
		for _, f := range linkStatus.Files {
			if !f.IsLinked && (strings.Contains(strings.ToLower(f.Issue), "conflict") ||
				strings.Contains(strings.ToLower(f.Issue), "exists") ||
				strings.Contains(strings.ToLower(f.Issue), "elsewhere")) {
				conflictCount++
			}
		}

		if conflictCount > 0 {
			info.icon = warnStyle.Render("⚠")
			info.statusTags = append(info.statusTags, fmt.Sprintf("conflicts (%d)", conflictCount))
		} else if linkStatus.IsFullyLinked() {
			info.icon = okStyle.Render("✓")
		} else if linkStatus.LinkedCount > 0 {
			info.icon = warnStyle.Render("◆") // Partial link indicator
		} else {
			info.icon = errStyle.Render("✗")
		}

		info.statusText = fmt.Sprintf("%d/%d", linkStatus.LinkedCount, linkStatus.TotalCount)
	} else {
		// Fallback if LinkStatus is missing (should be rare)
		if drift != nil && drift.HasDrift {
			info.icon = warnStyle.Render("◆")
			info.statusText = fmt.Sprintf("%d new", len(drift.NewFiles))
		} else {
			info.icon = ui.SubtleStyle.Render("•")
			info.statusText = "unknown"
		}
	}

	if len(cfg.ExternalDeps) > 0 {
		missingExternal := false
		home := os.Getenv("HOME")
		for _, ext := range cfg.ExternalDeps {
			dest := ext.Destination
			if dest == "" {
				continue
			}
			fullDest := dest
			if !filepath.IsAbs(dest) {
				if home == "" {
					continue
				}
				fullDest = filepath.Join(home, dest)
			}
			if _, err := os.Stat(fullDest); os.IsNotExist(err) {
				missingExternal = true
				break
			}
		}
		if missingExternal {
			info.statusTags = append(info.statusTags, "external")
		}
	}

	if len(cfg.DependsOn) > 0 && s.state.LinkStatus != nil {
		missingDep := false
		for _, depName := range cfg.DependsOn {
			depStatus, ok := s.state.LinkStatus[depName]
			if !ok || !depStatus.IsFullyLinked() {
				missingDep = true
				break
			}
		}
		if missingDep {
			info.statusTags = append(info.statusTags, "deps")
		}
	}

	return info
}
