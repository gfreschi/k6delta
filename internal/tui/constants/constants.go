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

// Panel expand modes.
const (
	ExpandNormal   = 0 // all panels visible at default sizes
	ExpandExpanded = 1 // focused panel gets most height, others title-only
	ExpandFull     = 2 // only focused panel renders
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
