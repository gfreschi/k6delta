// Package context provides shared state for all TUI components.
package context

import (
	"github.com/gfreschi/k6delta/internal/tui/constants"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

// ViewType identifies the active dashboard view for rendering decisions.
type ViewType int

const (
	ViewBrowsing  ViewType = iota // app browser (default)
	ViewRunning                   // k6 run in progress
	ViewAnalyzing                 // analyze metrics
	ViewComparing                 // compare reports
	ViewReport                    // viewing a saved report
)

// ProgramContext is the central state container shared by all TUI models.
type ProgramContext struct {
	ScreenWidth   int
	ScreenHeight  int
	ContentWidth  int
	ContentHeight int
	Theme         theme.Theme
	Styles        Styles
	ActiveView    ViewType
}

// New creates a ProgramContext with default theme and dimensions.
func New(width, height int) *ProgramContext {
	t := theme.DefaultTheme
	ctx := &ProgramContext{
		ScreenWidth:  width,
		ScreenHeight: height,
		Theme:        t,
		Styles:       InitStyles(t),
	}
	ctx.updateDimensions()
	return ctx
}

// Resize updates dimensions from terminal size.
func (ctx *ProgramContext) Resize(width, height int) {
	ctx.ScreenWidth = width
	ctx.ScreenHeight = height
	ctx.updateDimensions()
}

func (ctx *ProgramContext) updateDimensions() {
	w := ctx.ScreenWidth
	if w < constants.MinContentWidth {
		w = constants.MinContentWidth
	}
	h := ctx.ScreenHeight
	if h < constants.MinContentHeight {
		h = constants.MinContentHeight
	}
	ctx.ContentWidth = w - 2*constants.PanelPadding
	ctx.ContentHeight = h - constants.HeaderHeight - constants.FooterHeight
}
