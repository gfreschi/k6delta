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
)

// Icons used across TUI components.
const (
	IconPending = "◯"
	IconRunning = "▸"
	IconDone    = "✓"
	IconFailed  = "✗"
	IconWarning = "⚠"
	IconUp      = "↑"
	IconDown    = "↓"
	IconBullet  = "·"
)
