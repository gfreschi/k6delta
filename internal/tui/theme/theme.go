// Package theme defines the color palette for the k6delta TUI.
// Colors only — no lipgloss.Style values. Styles are built in context/styles.go.
package theme

import "github.com/charmbracelet/lipgloss"

// Theme holds semantic colors used to build styles.
// All colors use AdaptiveColor for automatic light/dark terminal detection.
type Theme struct {
	PrimaryBorder lipgloss.AdaptiveColor
	FocusedBorder lipgloss.AdaptiveColor
	FaintBorder   lipgloss.AdaptiveColor
	PrimaryText   lipgloss.AdaptiveColor
	SecondaryText lipgloss.AdaptiveColor
	FaintText     lipgloss.AdaptiveColor
	SuccessText   lipgloss.AdaptiveColor
	WarningText   lipgloss.AdaptiveColor
	ErrorText     lipgloss.AdaptiveColor
	HeaderText    lipgloss.AdaptiveColor
	DeltaBetter      lipgloss.AdaptiveColor
	DeltaWorse       lipgloss.AdaptiveColor
	DeltaNeutral     lipgloss.AdaptiveColor
	TileBorder       lipgloss.AdaptiveColor
	TileBorderOK     lipgloss.AdaptiveColor
	TileBorderWarn   lipgloss.AdaptiveColor
	TileBorderError  lipgloss.AdaptiveColor
	TimelineAlarm    lipgloss.AdaptiveColor
	TimelineScaling  lipgloss.AdaptiveColor
	TimelineResolved lipgloss.AdaptiveColor
}

// DefaultTheme provides the muted professional palette.
var DefaultTheme = Theme{
	PrimaryBorder: lipgloss.AdaptiveColor{Light: "008", Dark: "008"},
	FocusedBorder: lipgloss.AdaptiveColor{Light: "012", Dark: "012"},
	FaintBorder:   lipgloss.AdaptiveColor{Light: "236", Dark: "236"},
	PrimaryText:   lipgloss.AdaptiveColor{Light: "000", Dark: "015"},
	SecondaryText: lipgloss.AdaptiveColor{Light: "008", Dark: "007"},
	FaintText:     lipgloss.AdaptiveColor{Light: "008", Dark: "243"},
	SuccessText:   lipgloss.AdaptiveColor{Light: "002", Dark: "002"},
	WarningText:   lipgloss.AdaptiveColor{Light: "003", Dark: "003"},
	ErrorText:     lipgloss.AdaptiveColor{Light: "001", Dark: "001"},
	HeaderText:    lipgloss.AdaptiveColor{Light: "012", Dark: "012"},
	DeltaBetter:      lipgloss.AdaptiveColor{Light: "002", Dark: "002"},
	DeltaWorse:       lipgloss.AdaptiveColor{Light: "001", Dark: "001"},
	DeltaNeutral:     lipgloss.AdaptiveColor{Light: "008", Dark: "243"},
	TileBorder:       lipgloss.AdaptiveColor{Light: "008", Dark: "008"},
	TileBorderOK:     lipgloss.AdaptiveColor{Light: "002", Dark: "002"},
	TileBorderWarn:   lipgloss.AdaptiveColor{Light: "003", Dark: "003"},
	TileBorderError:  lipgloss.AdaptiveColor{Light: "001", Dark: "001"},
	TimelineAlarm:    lipgloss.AdaptiveColor{Light: "001", Dark: "001"},
	TimelineScaling:  lipgloss.AdaptiveColor{Light: "003", Dark: "003"},
	TimelineResolved: lipgloss.AdaptiveColor{Light: "002", Dark: "002"},
}
