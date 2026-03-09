// Package constants defines shared layout values and icons for the TUI.
package constants

// Dimensions represents a width/height pair.
type Dimensions struct {
	Width  int
	Height int
}

// Layout constants.
const (
	MinContentWidth  = 80
	MinContentHeight = 24
	PanelPadding     = 1
	FooterHeight     = 1
	HeaderHeight     = 2
)

// Responsive breakpoints.
const (
	BreakpointSplit   = 120 // side-by-side panel layout
	BreakpointStacked = 80  // vertical stack layout
	BreakpointNarrow  = 100 // reduced detail (e.g., omit direction words)
	BreakpointWide    = 140 // extra-wide layout
)

// ExpandMode controls how panels share available space.
type ExpandMode int

// Panel expand modes.
const (
	ExpandNormal ExpandMode = iota // all panels visible at default sizes
	ExpandFull                     // only focused panel renders
)

// Minimum dimensions for panels and components.
const (
	MinPanelWidth    = 30
	MinTimelineWidth = 60
)

// Panel layout split ratio (percentage given to the left/infra panel).
const PanelSplitPct = 55

// PanelSplitPctNarrow is the left panel width percentage at 100-119 width (compact split).
const PanelSplitPctNarrow = 50

// Tile widths for KPI metric cards at different display contexts.
const (
	TileWidthNarrow = 12 // health micro-tiles (post-k6 summary strip)
	TileWidthNormal = 14 // dashboard tiles (vital signs, infra, analyze)
	TileWidthWide   = 16 // live dashboard tiles
)

// Layout overhead accounts for header, stepper gap, statusbar, footer, and margins.
// header(2) + stepper-gap(2) + statusbar(1) + footer(1) + margins(2) = 8
const LayoutOverhead = 8

// LiveChartHeight is the default chart height in split layout.
const LiveChartHeight = 12

// LiveChartHeightStacked is the chart height in stacked layout.
const LiveChartHeightStacked = 10

// PanelBorderWidth is the total horizontal space consumed by panel borders (left + right).
const PanelBorderWidth = 2

// PanelInnerPadding is the total horizontal space consumed by panel inner padding.
const PanelInnerPadding = 2

// Timeline layout constants.
const (
	TimelineLabelReserve = 20 // space reserved for lane label + peak text
	TimelineAxisReserve  = 8  // space reserved for axis padding
)

// Icons used across TUI components.
const (
	IconPending  = "◯"
	IconRunning  = "▸"
	IconDone     = "✓"
	IconFailed   = "✗"
	IconWarning  = "⚠"
	IconUp       = "↑"
	IconDown     = "↓"
	IconBullet   = "·"
	IconAlarm    = "🔔"
	IconScaling  = "⇅"
	IconResolved = "✔"
	IconQuiet    = "—"
	IconArrowUp  = "▲"
	IconArrowDn  = "▼"
)

// Chart height limits.
const (
	MinChartHeight = 6
	MaxChartHeight = 20
)

// CalcChartHeight returns a chart height clamped between MinChartHeight and MaxChartHeight.
func CalcChartHeight(availableHeight int) int {
	if availableHeight < MinChartHeight {
		return MinChartHeight
	}
	if availableHeight > MaxChartHeight {
		return MaxChartHeight
	}
	return availableHeight
}

// CalcPanelHeights splits total height into two portions by percentage.
// The top portion gets topPct% of totalHeight (minimum 4 lines).
// The bottom portion gets the remainder (minimum 4 lines).
func CalcPanelHeights(totalHeight, topPct int) (int, int) {
	topH := max(totalHeight*topPct/100, 4)
	bottomH := max(totalHeight-topH, 4)
	return topH, bottomH
}

// TileWidth returns the appropriate tile width for a given content width.
// Returns TileWidthNormal at >=120, TileWidthNarrow at >=80, 0 at <80 (skip tiles).
func TileWidth(contentWidth int) int {
	switch {
	case contentWidth >= BreakpointSplit:
		return TileWidthNormal
	case contentWidth >= BreakpointStacked:
		return TileWidthNarrow
	default:
		return 0
	}
}
