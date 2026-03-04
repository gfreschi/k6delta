package common

import (
	"math"

	"github.com/charmbracelet/lipgloss"
)

// DeltaStyleTiers holds pre-built styles for each intensity tier.
// Mirrors context.DeltaStyles to avoid import cycles (common → context → common).
type DeltaStyleTiers struct {
	Better        lipgloss.Style
	BetterMild    lipgloss.Style
	BetterStrong  lipgloss.Style
	Worse       lipgloss.Style
	WorseMild   lipgloss.Style
	WorseSevere lipgloss.Style
	Neutral       lipgloss.Style
}

// DeltaStyle returns a lipgloss.Style based on the magnitude and direction of a change.
// pctChange is the percentage change (positive = increase, negative = decrease).
// lowerIsBetter indicates if a decrease is an improvement (e.g., latency, error rate).
func DeltaStyle(ds DeltaStyleTiers, pctChange float64, lowerIsBetter bool) lipgloss.Style {
	absPct := math.Abs(pctChange)

	isImprovement := (lowerIsBetter && pctChange < 0) || (!lowerIsBetter && pctChange > 0)
	isRegression := (lowerIsBetter && pctChange > 0) || (!lowerIsBetter && pctChange < 0)

	switch {
	case absPct < 2:
		return ds.Neutral
	case isImprovement && absPct < 5:
		return ds.BetterMild
	case isImprovement && absPct < 15:
		return ds.Better
	case isImprovement:
		return ds.BetterStrong
	case isRegression && absPct < 5:
		return ds.WorseMild
	case isRegression && absPct < 15:
		return ds.Worse
	case isRegression:
		return ds.WorseSevere
	default:
		return ds.Neutral
	}
}
