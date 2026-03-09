package common_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func testCommonStyles() common.CommonStyles {
	ctx := tuictx.New(120, 40)
	return ctx.Styles.Common
}

func TestRenderEmptyState_noData(t *testing.T) {
	styles := testCommonStyles()
	result := common.RenderEmptyState(styles, common.EmptyNoData, "No metrics available", "")
	if !strings.Contains(result, "No metrics available") {
		t.Errorf("expected title in output, got %q", result)
	}
}

func TestRenderEmptyState_withSubtitle(t *testing.T) {
	styles := testCommonStyles()
	result := common.RenderEmptyState(styles, common.EmptyError, "Failed to load", "connection timeout")
	if !strings.Contains(result, "Failed to load") {
		t.Errorf("expected title in output, got %q", result)
	}
	if !strings.Contains(result, "connection timeout") {
		t.Errorf("expected subtitle in output, got %q", result)
	}
}
