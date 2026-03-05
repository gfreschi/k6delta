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
	ExpandNormal   ExpandMode = iota // all panels visible at default sizes
	ExpandExpanded                   // focused panel gets most height, others title-only
	ExpandFull                       // only focused panel renders
)

// Minimum dimensions for panels and components.
const (
	MinPanelWidth    = 30
	MinTimelineWidth = 60
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

// CalcPanelHeights splits total height into two portions by percentage.
// The top portion gets topPct% of totalHeight (minimum 4 lines).
// The bottom portion gets the remainder (minimum 4 lines).
func CalcPanelHeights(totalHeight, topPct int) (int, int) {
	topH := max(totalHeight*topPct/100, 4)
	bottomH := max(totalHeight-topH, 4)
	return topH, bottomH
}
