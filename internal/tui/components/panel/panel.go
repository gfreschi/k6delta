// Package panel provides a bordered panel with focus state, title, and scroll support.
package panel

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model is the panel Bubble Tea model.
type Model struct {
	ctx      *tuictx.ProgramContext
	title    string
	width    int
	height   int
	content  string
	focused  bool
	viewport viewport.Model
	overflow bool // true when content exceeds panel body height
}

// NewModel creates a panel with title and dimensions.
func NewModel(ctx *tuictx.ProgramContext, title string, width, height int) Model {
	bodyH := bodyHeight(height)
	bodyW := bodyWidth(width)
	vp := viewport.New(bodyW, bodyH)
	return Model{
		ctx:      ctx,
		title:    title,
		width:    width,
		height:   height,
		viewport: vp,
	}
}

// SetContent sets the panel body content.
func (m *Model) SetContent(content string) {
	m.content = content
	m.viewport.SetContent(content)
	lines := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") && len(content) > 0 {
		lines++
	}
	m.overflow = lines > m.viewport.Height
}

// SetFocused sets the focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetDimensions updates panel width and height.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = bodyWidth(width)
	m.viewport.Height = bodyHeight(height)
	// Re-evaluate overflow
	m.SetContent(m.content)
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// ScrollDown scrolls the viewport down one line.
func (m *Model) ScrollDown() {
	m.viewport.LineDown(1)
}

// ScrollUp scrolls the viewport up one line.
func (m *Model) ScrollUp() {
	m.viewport.LineUp(1)
}

// ScrollPosition returns the current scroll offset and total content lines.
func (m *Model) ScrollPosition() (current, total int) {
	return m.viewport.YOffset, m.viewport.TotalLineCount()
}

// View renders the panel with border and title.
func (m Model) View() string {
	style := m.ctx.Styles.Panel.Root
	if m.focused {
		style = m.ctx.Styles.Panel.Focused
	}

	titleText := m.title
	if m.overflow {
		cur, total := m.ScrollPosition()
		titleText = fmt.Sprintf("%s (%d/%d)", m.title, cur+1, total)
	}

	titleStyle := m.ctx.Styles.Header.Root
	header := titleStyle.Render(titleText)

	body := m.viewport.View()

	inner := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(m.width).Render(inner)
}

func bodyHeight(panelHeight int) int {
	h := panelHeight - 3 // border (2) + title (1)
	if h < 1 {
		h = 1
	}
	return h
}

func bodyWidth(panelWidth int) int {
	w := panelWidth - 2 // border (2)
	if w < 1 {
		w = 1
	}
	return w
}
