package common_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

func TestBuildStyles_producesNonEmptyGlyphs(t *testing.T) {
	cs := common.BuildStyles(theme.DefaultTheme)
	if cs.CheckMark == "" {
		t.Error("CheckMark glyph is empty")
	}
	if cs.XMark == "" {
		t.Error("XMark glyph is empty")
	}
	if cs.WarningSign == "" {
		t.Error("WarningSign glyph is empty")
	}
}
