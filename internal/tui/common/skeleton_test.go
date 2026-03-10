package common_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

func testStyles() common.CommonStyles {
	return common.BuildStyles(theme.DefaultTheme)
}

func TestSkeletonTable(t *testing.T) {
	styles := testStyles()
	result := common.SkeletonTable(styles, 40, 3)
	lines := strings.Split(result, "\n")
	// Header + 3 rows = 4+ lines
	if len(lines) < 4 {
		t.Errorf("expected at least 4 lines, got %d", len(lines))
	}
}

func TestSkeletonTileRow(t *testing.T) {
	styles := testStyles()
	result := common.SkeletonTileRow(styles, 60, 3)
	if result == "" {
		t.Error("SkeletonTileRow returned empty string")
	}
}

func TestSkeletonChart(t *testing.T) {
	styles := testStyles()
	result := common.SkeletonChart(styles, 40, 6)
	lines := strings.Split(result, "\n")
	if len(lines) < 6 {
		t.Errorf("expected at least 6 lines, got %d", len(lines))
	}
}
