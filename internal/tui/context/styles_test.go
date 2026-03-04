package context_test

import (
	"testing"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

func TestInitStyles_allGroupsPopulated(t *testing.T) {
	s := tuictx.InitStyles(theme.DefaultTheme)

	// Verify key style groups exist and render non-empty strings
	if s.Common.CheckMark == "" {
		t.Error("Common.CheckMark is empty")
	}
	// Table styles should render without panic
	_ = s.Table.Header.Render("test")
	_ = s.Panel.Focused.Render("test")
	_ = s.Footer.Key.Render("test")
	_ = s.Verdict.Pass.Render("test")
}
