package common_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/common"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

func deltaTiers(ds tuictx.DeltaStyles) common.DeltaStyleTiers {
	return common.DeltaStyleTiers{
		Better:       ds.Better,
		BetterMild:   ds.BetterMild,
		BetterStrong: ds.BetterStrong,
		Worse:        ds.Worse,
		WorseMild:    ds.WorseMild,
		WorseSevere:  ds.WorseSevere,
		Neutral:      ds.Neutral,
	}
}

func TestDeltaStyle_intensity(t *testing.T) {
	s := tuictx.InitStyles(theme.DefaultTheme)
	tiers := deltaTiers(s.Delta)

	tests := []struct {
		name          string
		pctChange     float64
		lowerIsBetter bool
		wantStyle     lipgloss.Style
	}{
		// Neutral (< 2%)
		{"zero change", 0.0, true, tiers.Neutral},
		{"tiny improvement", -1.5, true, tiers.Neutral},
		{"tiny regression", 1.9, true, tiers.Neutral},

		// Boundary at 2%
		{"exactly 2% improvement lower-better", -2.0, true, tiers.BetterMild},
		{"exactly 2% regression lower-better", 2.0, true, tiers.WorseMild},

		// Mild (2-5%)
		{"mild improvement lower-better", -3.0, true, tiers.BetterMild},
		{"mild regression lower-better", 3.0, true, tiers.WorseMild},

		// Boundary at 5%
		{"exactly 5% improvement", -5.0, true, tiers.Better},
		{"exactly 5% regression", 5.0, true, tiers.Worse},

		// Moderate (5-15%)
		{"moderate improvement", -10.0, true, tiers.Better},
		{"moderate regression", 10.0, true, tiers.Worse},

		// Boundary at 15%
		{"exactly 15% improvement", -15.0, true, tiers.BetterStrong},
		{"exactly 15% regression", 15.0, true, tiers.WorseSevere},

		// Strong/Severe (>= 15%)
		{"strong improvement", -25.0, true, tiers.BetterStrong},
		{"severe regression", 25.0, true, tiers.WorseSevere},

		// Higher-is-better direction (inverted)
		{"improvement higher-better", 10.0, false, tiers.Better},
		{"regression higher-better", -10.0, false, tiers.Worse},
		{"severe regression higher-better", -25.0, false, tiers.WorseSevere},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := common.DeltaStyle(tiers, tt.pctChange, tt.lowerIsBetter)
			if got.Render("x") != tt.wantStyle.Render("x") {
				t.Errorf("DeltaStyle(%v, lowerBetter=%v) = %q, want %q",
					tt.pctChange, tt.lowerIsBetter,
					got.Render("x"), tt.wantStyle.Render("x"))
			}
		})
	}
}
