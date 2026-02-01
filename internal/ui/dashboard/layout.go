package dashboard

// PanelRegion represents the position and size of a panel in the layout
type PanelRegion struct {
	X      int
	Y      int
	Width  int
	Height int
}

// LayoutConfig holds configuration for the dashboard layout
type LayoutConfig struct {
	// Minimum dimensions
	MinMiniColWidth  int
	MinConfigsWidth  int
	MinDetailsWidth  int
	MinOutputWidth   int
	MinPanelHeight   int
	MinOutputHeight  int

	// Proportions (as percentages)
	MiniColPercent    int // Left mini-column width percentage
	ConfigsPercent    int // Configs panel width percentage
	DetailsPercent    int // Details panel width percentage
	OutputPercent     int // Output panel width percentage
	OutputHeightRatio int // Output height as fraction (e.g., 3 means 1/3)
}

// DefaultLayoutConfig returns the default layout configuration
func DefaultLayoutConfig() LayoutConfig {
	return LayoutConfig{
		MinMiniColWidth:   12,
		MinConfigsWidth:   24, // Enough for checkbox + ~12 char name + icon
		MinDetailsWidth:   25,
		MinOutputWidth:    30,
		MinPanelHeight:    3,
		MinOutputHeight:   4,
		MiniColPercent:    15,
		ConfigsPercent:    24, // Narrower - just needs checkbox + name + icon
		DetailsPercent:    33, // Wider - file trees need space
		OutputPercent:     33, // Wider - logs need room
		OutputHeightRatio: 3,
	}
}

// Layout calculates all panel regions based on terminal dimensions
type Layout struct {
	config LayoutConfig

	// Calculated regions
	Summary   PanelRegion
	Health    PanelRegion
	Overrides PanelRegion
	External  PanelRegion
	Configs   PanelRegion
	Details   PanelRegion
	Output    PanelRegion

	// Total dimensions
	Width  int
	Height int

	// Reserved space
	HeaderHeight int
	FooterHeight int
}

// NewLayout creates a new layout calculator
func NewLayout() *Layout {
	return &Layout{
		config:       DefaultLayoutConfig(),
		HeaderHeight: 0, // Panels start at top
		FooterHeight: 2, // Footer with keybindings + header info
	}
}

// Calculate computes all panel regions for the given terminal size
func (l *Layout) Calculate(width, height int) {
	l.Width = width
	l.Height = height

	// Guard against very small terminals
	if width < 40 {
		width = 40
	}
	if height < 10 {
		height = 10
	}

	// Calculate available content area (excluding header/footer)
	contentHeight := height - l.HeaderHeight - l.FooterHeight
	if contentHeight < 10 {
		contentHeight = 10
	}

	// Calculate column widths
	miniColWidth := width * l.config.MiniColPercent / 100
	if miniColWidth < l.config.MinMiniColWidth {
		miniColWidth = l.config.MinMiniColWidth
	}

	remainingWidth := width - miniColWidth
	if remainingWidth < 20 {
		remainingWidth = 20
	}

	// Distribute remaining width among configs, details, output columns
	// These share the remaining space proportionally
	totalPercent := l.config.ConfigsPercent + l.config.DetailsPercent + l.config.OutputPercent
	configsWidth := remainingWidth * l.config.ConfigsPercent / totalPercent
	if configsWidth < l.config.MinConfigsWidth {
		configsWidth = l.config.MinConfigsWidth
	}

	detailsWidth := remainingWidth * l.config.DetailsPercent / totalPercent
	if detailsWidth < l.config.MinDetailsWidth {
		detailsWidth = l.config.MinDetailsWidth
	}

	outputWidth := remainingWidth - configsWidth - detailsWidth
	if outputWidth < l.config.MinOutputWidth {
		outputWidth = l.config.MinOutputWidth
	}

	// Clamp total width to prevent overflow
	totalWidth := miniColWidth + configsWidth + detailsWidth + outputWidth
	if totalWidth > l.Width {
		// Scale down proportionally
		scale := float64(l.Width) / float64(totalWidth)
		miniColWidth = int(float64(miniColWidth) * scale)
		configsWidth = int(float64(configsWidth) * scale)
		detailsWidth = int(float64(detailsWidth) * scale)
		outputWidth = l.Width - miniColWidth - configsWidth - detailsWidth
	}

	// Calculate mini-column panel heights (4 stacked panels sharing contentHeight)
	miniPanelHeight := contentHeight / 4
	if miniPanelHeight < l.config.MinPanelHeight {
		miniPanelHeight = l.config.MinPanelHeight
	}

	// Set panel regions
	// Mini column (stacked vertically on left)
	l.Summary = PanelRegion{
		X:      0,
		Y:      l.HeaderHeight,
		Width:  miniColWidth,
		Height: miniPanelHeight,
	}

	l.Health = PanelRegion{
		X:      0,
		Y:      l.HeaderHeight + miniPanelHeight,
		Width:  miniColWidth,
		Height: miniPanelHeight,
	}

	l.Overrides = PanelRegion{
		X:      0,
		Y:      l.HeaderHeight + miniPanelHeight*2,
		Width:  miniColWidth,
		Height: miniPanelHeight,
	}

	l.External = PanelRegion{
		X:      0,
		Y:      l.HeaderHeight + miniPanelHeight*3,
		Width:  miniColWidth,
		Height: contentHeight - miniPanelHeight*3, // Last panel takes remaining space
	}

	// Main panels (all span full content height for visual alignment)
	l.Configs = PanelRegion{
		X:      miniColWidth,
		Y:      l.HeaderHeight,
		Width:  configsWidth,
		Height: contentHeight,
	}

	l.Details = PanelRegion{
		X:      miniColWidth + configsWidth,
		Y:      l.HeaderHeight,
		Width:  detailsWidth,
		Height: contentHeight,
	}

	l.Output = PanelRegion{
		X:      miniColWidth + configsWidth + detailsWidth,
		Y:      l.HeaderHeight,
		Width:  outputWidth,
		Height: contentHeight,
	}
}

// GetPanelRegion returns the region for a given panel ID
func (l *Layout) GetPanelRegion(id PanelID) PanelRegion {
	switch id {
	case PanelSummary:
		return l.Summary
	case PanelHealth:
		return l.Health
	case PanelOverrides:
		return l.Overrides
	case PanelExternal:
		return l.External
	case PanelConfigs:
		return l.Configs
	case PanelDetails:
		return l.Details
	case PanelOutput:
		return l.Output
	default:
		return PanelRegion{}
	}
}

// GetMainContentHeight returns the height of the main content area
func (l *Layout) GetMainContentHeight() int {
	return l.Height - l.HeaderHeight - l.FooterHeight
}

// GetMiniColumnWidth returns the width of the mini column
func (l *Layout) GetMiniColumnWidth() int {
	return l.Summary.Width
}

// ApplyToPanels updates panel sizes based on calculated layout
func (l *Layout) ApplyToPanels(panels map[PanelID]Panel) {
	for id, panel := range panels {
		region := l.GetPanelRegion(id)
		panel.SetSize(region.Width, region.Height)
	}
}
