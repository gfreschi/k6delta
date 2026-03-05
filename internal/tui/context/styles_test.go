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

func TestInitStyles_tileStyles(t *testing.T) {
	s := tuictx.InitStyles(theme.DefaultTheme)

	// Tile styles should render without panic
	_ = s.Tile.Border.Render("test")
	_ = s.Tile.BorderOK.Render("test")
	_ = s.Tile.BorderWarn.Render("test")
	_ = s.Tile.BorderError.Render("test")
}

func TestInitStyles_timelineStyles(t *testing.T) {
	s := tuictx.InitStyles(theme.DefaultTheme)

	_ = s.Timeline.Alarm.Render("test")
	_ = s.Timeline.Scaling.Render("test")
	_ = s.Timeline.Resolved.Render("test")
}

func TestInitStyles_statusBarStyles(t *testing.T) {
	s := tuictx.InitStyles(theme.DefaultTheme)

	_ = s.StatusBar.Root.Render("test")
	_ = s.StatusBar.Label.Render("test")
	_ = s.StatusBar.Value.Render("test")
}
