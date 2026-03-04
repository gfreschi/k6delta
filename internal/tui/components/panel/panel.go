// Package panel provides a bordered panel with focus state and title.
package panel

import (
	"github.com/charmbracelet/lipgloss"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model is the panel Bubble Tea model.
type Model struct {
	ctx     *tuictx.ProgramContext
	title   string
	width   int
	height  int
	content string
	focused bool
}

// NewModel creates a panel with title and dimensions.
func NewModel(ctx *tuictx.ProgramContext, title string, width, height int) Model {
	return Model{ctx: ctx, title: title, width: width, height: height}
}

// SetContent sets the panel body content.
func (m *Model) SetContent(content string) {
	m.content = content
}

// SetFocused sets the focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetDimensions updates panel width and height.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the panel with border and title.
func (m Model) View() string {
	style := m.ctx.Styles.Panel.Root
	if m.focused {
		style = m.ctx.Styles.Panel.Focused
	}

	titleStyle := m.ctx.Styles.Header.Root
	header := titleStyle.Render(m.title)

	body := lipgloss.NewStyle().
		Width(m.width - 2).  // account for border
		Height(m.height - 3). // account for border + title
		Render(m.content)

	inner := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(m.width).Render(inner)
}
