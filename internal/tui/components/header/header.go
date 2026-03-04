// Package header provides the app/env/phase context bar.
package header

import (
	"fmt"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model is the header Bubble Tea model.
type Model struct {
	ctx    *tuictx.ProgramContext
	app    string
	env    string
	phase  string
	suffix string // optional right-side text (elapsed time, verdict, etc.)
}

// NewModel creates a header.
func NewModel(ctx *tuictx.ProgramContext, app, env, phase string) Model {
	return Model{ctx: ctx, app: app, env: env, phase: phase}
}

// SetSuffix sets the right-aligned suffix text.
func (m *Model) SetSuffix(s string) {
	m.suffix = s
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the header bar.
func (m Model) View() string {
	s := m.ctx.Styles.Header
	left := s.Root.Render(fmt.Sprintf("k6delta: %s (%s) -- %s", m.app, m.env, m.phase))
	if m.suffix != "" {
		left += "  " + s.Context.Render(m.suffix)
	}
	return left
}
