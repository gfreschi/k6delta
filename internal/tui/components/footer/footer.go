// Package footer provides a context-sensitive keybinding bar with responsive label collapsing.
package footer

import (
	"strings"

	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// FooterState represents the current interaction mode.
type FooterState int

const (
	StateNormal    FooterState = iota
	StateExpanded              // a panel is fully expanded
	StateDrillDown             // drill-down active
	StateHelp                  // help overlay active
)

// KeyHint represents a single key-action pair (legacy type).
type KeyHint struct {
	Key    string
	Action string
}

// Hint represents a key-action pair with responsive label tiers.
type Hint struct {
	Key   string
	Label string // full label (>= BreakpointSplit)
	Short string // abbreviated (< BreakpointSplit)
}

// Model is the footer Bubble Tea model.
type Model struct {
	ctx   *tuictx.ProgramContext
	hints []Hint
	width int
	state FooterState
}

// NewModel creates a footer with legacy KeyHint entries.
func NewModel(ctx *tuictx.ProgramContext, hints []KeyHint) Model {
	return Model{ctx: ctx, hints: convertKeyHints(hints), width: ctx.ScreenWidth}
}

// NewModelWithHints creates a footer with full Hint entries.
func NewModelWithHints(ctx *tuictx.ProgramContext, hints []Hint) Model {
	return Model{ctx: ctx, hints: hints, width: ctx.ScreenWidth}
}

// SetHints replaces the displayed key hints.
func (m *Model) SetHints(hints []KeyHint) {
	m.hints = convertKeyHints(hints)
}

// SetState sets the footer interaction mode (normal, expanded, drill-down, help).
func (m *Model) SetState(state FooterState) {
	m.state = state
}

// SetWidth sets the available width for responsive label selection.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
	m.width = ctx.ScreenWidth
}

// View renders the footer bar.
func (m Model) View() string {
	s := m.ctx.Styles.Footer
	hints := m.effectiveHints()
	var parts []string
	for _, h := range hints {
		label := m.pickLabel(h)
		part := s.Key.Render(h.Key) + " " + s.Action.Render(label)
		parts = append(parts, part)
	}
	sep := " " + s.Separator.Render(constants.IconBullet) + " "
	return "  " + strings.Join(parts, sep)
}

func (m Model) effectiveHints() []Hint {
	switch m.state {
	case StateHelp:
		return []Hint{{Key: "?", Label: "close", Short: "close"}, {Key: "esc", Label: "close", Short: "close"}}
	case StateExpanded:
		out := make([]Hint, 0, len(m.hints))
		for _, h := range m.hints {
			if h.Label == "expand" {
				out = append(out, Hint{Key: "esc", Label: "collapse", Short: "collapse"})
				continue
			}
			out = append(out, h)
		}
		return out
	case StateDrillDown:
		out := make([]Hint, 0, len(m.hints))
		for _, h := range m.hints {
			switch h.Label {
			case "expand", "panel", "jump":
				continue // hide panel nav in drill-down
			default:
				out = append(out, h)
			}
		}
		out = append(out, Hint{Key: "esc", Label: "close", Short: "close"})
		return out
	default:
		return m.hints
	}
}

func (m Model) pickLabel(h Hint) string {
	if m.width >= constants.BreakpointSplit {
		return h.Label
	}
	if h.Short != "" {
		return h.Short
	}
	return h.Label
}

func convertKeyHints(hints []KeyHint) []Hint {
	converted := make([]Hint, len(hints))
	for i, h := range hints {
		converted[i] = Hint{Key: h.Key, Label: h.Action, Short: h.Action}
	}
	return converted
}
