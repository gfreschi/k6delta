// Package footer provides a context-sensitive keybinding bar.
package footer

import (
	"strings"

	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// KeyHint represents a single key-action pair.
type KeyHint struct {
	Key    string
	Action string
}

// Model is the footer Bubble Tea model.
type Model struct {
	ctx   *tuictx.ProgramContext
	hints []KeyHint
}

// NewModel creates a footer with the given key hints.
func NewModel(ctx *tuictx.ProgramContext, hints []KeyHint) Model {
	return Model{ctx: ctx, hints: hints}
}

// SetHints replaces the displayed key hints.
func (m *Model) SetHints(hints []KeyHint) {
	m.hints = hints
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the footer bar.
func (m Model) View() string {
	s := m.ctx.Styles.Footer
	var parts []string
	for _, h := range m.hints {
		part := s.Key.Render(h.Key) + " " + s.Action.Render(h.Action)
		parts = append(parts, part)
	}
	sep := " " + s.Separator.Render(constants.IconBullet) + " "
	return "  " + strings.Join(parts, sep)
}
