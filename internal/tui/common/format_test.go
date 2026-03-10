package common_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
)

func TestCompactNumber(t *testing.T) {
	tests := []struct {
		input float64
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{15000, "15.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{1000000000, "1.0B"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := common.CompactNumber(tt.input)
			if got != tt.want {
				t.Errorf("CompactNumber(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
