// Package statusbar provides a persistent operation context bar.
package statusbar

import (
	"strings"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Item represents a label-value pair displayed in the status bar.
type Item struct {
	Label string
	Value string
}

// Model is the status bar Bubble Tea model.
type Model struct {
	ctx   *tuictx.ProgramContext
	items []Item
}

// NewModel creates a status bar.
func NewModel(ctx *tuictx.ProgramContext) Model {
	return Model{ctx: ctx}
}

// SetItems replaces all status bar items.
func (m *Model) SetItems(items []Item) {
	m.items = items
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the status bar as "── label: value ── label: value ──".
// Returns empty string when no items are set.
func (m Model) View() string {
	if len(m.items) == 0 {
		return ""
	}

	s := m.ctx.Styles.StatusBar
	var parts []string
	for _, item := range m.items {
		part := s.Label.Render(item.Label+":") + " " + s.Value.Render(item.Value)
		parts = append(parts, part)
	}

	sep := s.Root.Render(" ── ")
	content := strings.Join(parts, sep)
	return s.Root.Render("── ") + content + s.Root.Render(" ──")
}
