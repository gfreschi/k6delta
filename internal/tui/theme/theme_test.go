package theme_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/theme"
)

func TestDefaultTheme_hasAllColors(t *testing.T) {
	th := theme.DefaultTheme

	// Verify no zero-value AdaptiveColors (both Light and Dark must be set)
	colors := []struct {
		name  string
		color string
	}{
		{"PrimaryBorder.Dark", th.PrimaryBorder.Dark},
		{"FocusedBorder.Dark", th.FocusedBorder.Dark},
		{"PrimaryText.Dark", th.PrimaryText.Dark},
		{"SuccessText.Dark", th.SuccessText.Dark},
		{"WarningText.Dark", th.WarningText.Dark},
		{"ErrorText.Dark", th.ErrorText.Dark},
		{"FaintText.Dark", th.FaintText.Dark},
		{"HeaderText.Dark", th.HeaderText.Dark},
		{"DeltaBetter.Dark", th.DeltaBetter.Dark},
		{"DeltaWorse.Dark", th.DeltaWorse.Dark},
		{"TileBorder.Dark", th.TileBorder.Dark},
		{"TileBorderOK.Dark", th.TileBorderOK.Dark},
		{"TileBorderWarn.Dark", th.TileBorderWarn.Dark},
		{"TileBorderError.Dark", th.TileBorderError.Dark},
		{"TimelineAlarm.Dark", th.TimelineAlarm.Dark},
		{"TimelineScaling.Dark", th.TimelineScaling.Dark},
		{"TimelineResolved.Dark", th.TimelineResolved.Dark},
	}
	for _, c := range colors {
		if c.color == "" {
			t.Errorf("DefaultTheme.%s is empty", c.name)
		}
	}
}
