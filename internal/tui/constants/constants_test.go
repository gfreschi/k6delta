package constants_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/constants"
)

func TestCalcChartHeight(t *testing.T) {
	tests := []struct {
		name      string
		available int
		want      int
	}{
		{"below floor", 4, 6},      // floor
		{"at floor", 6, 6},         // floor boundary
		{"small terminal", 10, 10}, // pass-through
		{"normal", 20, 20},         // pass-through
		{"large terminal", 30, 20}, // cap
		{"huge terminal", 50, 20},  // cap
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constants.CalcChartHeight(tt.available)
			if got != tt.want {
				t.Errorf("CalcChartHeight(%d) = %d, want %d", tt.available, got, tt.want)
			}
		})
	}
}

func TestCalcPanelHeights(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		topPct  int
		wantTop int
		wantBot int
	}{
		{"normal", 40, 55, 22, 18},
		{"minimum top", 6, 50, 4, 4},
		{"minimum bottom", 10, 90, 9, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTop, gotBot := constants.CalcPanelHeights(tt.total, tt.topPct)
			if gotTop != tt.wantTop {
				t.Errorf("CalcPanelHeights(%d, %d) top = %d, want %d", tt.total, tt.topPct, gotTop, tt.wantTop)
			}
			if gotBot != tt.wantBot {
				t.Errorf("CalcPanelHeights(%d, %d) bottom = %d, want %d", tt.total, tt.topPct, gotBot, tt.wantBot)
			}
		})
	}
}
