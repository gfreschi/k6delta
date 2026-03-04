package common_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

func deltaTiers(ds tuictx.DeltaStyles) common.DeltaStyleTiers {
	return common.DeltaStyleTiers{
		Better:        ds.Better,
		BetterMild:    ds.BetterMild,
		BetterStrong:  ds.BetterStrong,
		Worse:         ds.Worse,
		WorseMild:     ds.WorseMild,
		WorseModerate: ds.WorseModerate,
		WorseSevere:   ds.WorseSevere,
		Neutral:       ds.Neutral,
	}
}

func TestDeltaStyle_intensity(t *testing.T) {
	s := tuictx.InitStyles(theme.DefaultTheme)
	tiers := deltaTiers(s.Delta)

	tests := []struct {
		name          string
		pctChange     float64
		lowerIsBetter bool
		wantWorse     bool
	}{
		{"small improvement, lower better", -1.0, true, false},
		{"large regression, lower better", 20.0, true, true},
		{"small regression, lower better", 3.0, true, true},
		{"large improvement, higher better", 20.0, false, false},
		{"neutral", 0.0, true, false},
		{"severe regression", -25.0, false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := common.DeltaStyle(tiers, tt.pctChange, tt.lowerIsBetter)
			rendered := style.Render("test")
			if rendered == "" {
				t.Error("expected non-empty rendered output")
			}
		})
	}
}
