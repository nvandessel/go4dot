package dashboard

// PanelID identifies each panel in the dashboard layout
type PanelID int

const (
	PanelSummary PanelID = iota
	PanelHealth
	PanelOverrides
	PanelExternal
	PanelConfigs
	PanelDetails
	PanelOutput
)

// panelCount is the total number of panels
const panelCount = 7

// String returns the panel name for display
func (p PanelID) String() string {
	switch p {
	case PanelSummary:
		return "Summary"
	case PanelHealth:
		return "Health"
	case PanelOverrides:
		return "Overrides"
	case PanelExternal:
		return "External"
	case PanelConfigs:
		return "Configs"
	case PanelDetails:
		return "Details"
	case PanelOutput:
		return "Output"
	default:
		return "Unknown"
	}
}

// IsNavigable returns true if the panel supports list navigation
func (p PanelID) IsNavigable() bool {
	switch p {
	case PanelHealth, PanelOverrides, PanelExternal, PanelConfigs:
		return true
	default:
		return false
	}
}

// IsScrollable returns true if the panel supports scrolling when focused
func (p PanelID) IsScrollable() bool {
	switch p {
	case PanelDetails, PanelOutput:
		return true
	default:
		return false
	}
}

// FocusManager tracks focus state and handles navigation between panels
type FocusManager struct {
	currentFocus PanelID
	grid         [][]PanelID // 2D grid for directional navigation
}

// NewFocusManager creates a focus manager with the dashboard panel layout
//
// Layout grid (for directional navigation):
//
//	Col 0       Col 1       Col 2       Col 3
//
// Row 0: Summary    Configs     Details     Output
// Row 1: Health     Configs     Details     Output
// Row 2: Overrides  Configs     Details     Output
// Row 3: External   Configs     Details     Output
func NewFocusManager() *FocusManager {
	return &FocusManager{
		currentFocus: PanelConfigs, // Start with Configs panel focused
		grid: [][]PanelID{
			{PanelSummary, PanelConfigs, PanelDetails, PanelOutput},
			{PanelHealth, PanelConfigs, PanelDetails, PanelOutput},
			{PanelOverrides, PanelConfigs, PanelDetails, PanelOutput},
			{PanelExternal, PanelConfigs, PanelDetails, PanelOutput},
		},
	}
}

// CurrentFocus returns the currently focused panel
func (fm *FocusManager) CurrentFocus() PanelID {
	return fm.currentFocus
}

// SetFocus directly sets the focused panel
func (fm *FocusManager) SetFocus(panel PanelID) {
	fm.currentFocus = panel
}

// CycleNext cycles to the next navigable panel (Tab)
func (fm *FocusManager) CycleNext() {
	navigable := fm.getNavigablePanels()
	if len(navigable) == 0 {
		return
	}

	// Find current index in navigable list
	currentIdx := -1
	for i, p := range navigable {
		if p == fm.currentFocus {
			currentIdx = i
			break
		}
	}

	// Move to next navigable panel
	if currentIdx == -1 {
		fm.currentFocus = navigable[0]
	} else {
		fm.currentFocus = navigable[(currentIdx+1)%len(navigable)]
	}
}

// CyclePrev cycles to the previous navigable panel (Shift+Tab)
func (fm *FocusManager) CyclePrev() {
	navigable := fm.getNavigablePanels()
	if len(navigable) == 0 {
		return
	}

	// Find current index in navigable list
	currentIdx := -1
	for i, p := range navigable {
		if p == fm.currentFocus {
			currentIdx = i
			break
		}
	}

	// Move to previous navigable panel
	if currentIdx == -1 {
		fm.currentFocus = navigable[len(navigable)-1]
	} else {
		fm.currentFocus = navigable[(currentIdx-1+len(navigable))%len(navigable)]
	}
}

// getNavigablePanels returns panels that can receive focus
func (fm *FocusManager) getNavigablePanels() []PanelID {
	// Include navigable list panels plus scrollable panels
	return []PanelID{
		PanelHealth,
		PanelOverrides,
		PanelExternal,
		PanelConfigs,
		PanelDetails,
		PanelOutput,
	}
}

// findPosition finds the row and column of a panel in the grid
func (fm *FocusManager) findPosition(panel PanelID) (row, col int) {
	for r, rowPanels := range fm.grid {
		for c, p := range rowPanels {
			if p == panel {
				return r, c
			}
		}
	}
	return 0, 0
}

// MoveLeft moves focus to the panel on the left (Ctrl+h)
func (fm *FocusManager) MoveLeft() {
	row, col := fm.findPosition(fm.currentFocus)
	if col > 0 {
		fm.currentFocus = fm.grid[row][col-1]
	}
}

// MoveRight moves focus to the panel on the right (Ctrl+l)
func (fm *FocusManager) MoveRight() {
	row, col := fm.findPosition(fm.currentFocus)
	if col < len(fm.grid[row])-1 {
		fm.currentFocus = fm.grid[row][col+1]
	}
}

// MoveUp moves focus to the panel above (Ctrl+k)
func (fm *FocusManager) MoveUp() {
	row, col := fm.findPosition(fm.currentFocus)
	if row > 0 {
		fm.currentFocus = fm.grid[row-1][col]
	}
}

// MoveDown moves focus to the panel below (Ctrl+j)
func (fm *FocusManager) MoveDown() {
	row, col := fm.findPosition(fm.currentFocus)
	if row < len(fm.grid)-1 {
		fm.currentFocus = fm.grid[row+1][col]
	}
}

// JumpToPanel directly focuses a panel by number (0-6)
// 0=Output, 1=Summary, 2=Health, 3=Overrides, 4=External, 5=Configs, 6=Details
func (fm *FocusManager) JumpToPanel(num int) {
	if num < 0 || num >= panelCount {
		return
	}
	// Map numbers to panels: 0=Output (console), then 1-6 for others
	panels := []PanelID{
		PanelOutput,    // 0
		PanelSummary,   // 1
		PanelHealth,    // 2
		PanelOverrides, // 3
		PanelExternal,  // 4
		PanelConfigs,   // 5
		PanelDetails,   // 6
	}
	fm.currentFocus = panels[num]
}
