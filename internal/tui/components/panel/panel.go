// Package panel provides a bordered panel with focus state, title, and scroll support.
package panel

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

const (
	transitionFrames   = 3
	transitionInterval = 50 * time.Millisecond
)

// TransitionTickMsg signals the panel to advance its border transition animation.
type TransitionTickMsg struct{}

// Model is the panel Bubble Tea model.
type Model struct {
	ctx            *tuictx.ProgramContext
	title          string
	width          int
	height         int
	content        string
	child          tea.Model // optional child model whose View() replaces content
	focused        bool
	expandMode     constants.ExpandMode
	viewport       viewport.Model
	overflow       bool // true when content exceeds panel body height
	transitioning  bool
	transitionTick int // 0 to transitionFrames-1
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

// SetContent sets the panel body content. Clears any child model.
func (m *Model) SetContent(content string) {
	m.child = nil
	m.content = content
	m.viewport.SetContent(content)
	lines := strings.Count(content, "\n")
	if !strings.HasSuffix(content, "\n") && len(content) > 0 {
		lines++
	}
	m.overflow = lines > m.viewport.Height
}

// SetModel sets a child tea.Model whose View() output is used as panel content.
func (m *Model) SetModel(child tea.Model) {
	m.child = child
	m.content = ""
}

// Focused returns whether this panel is focused.
func (m Model) Focused() bool { return m.focused }

// CycleExpand cycles the expand mode: normal -> expanded -> full -> normal.
func (m *Model) CycleExpand() {
	m.expandMode = (m.expandMode + 1) % (constants.ExpandFull + 1)
}

// SetExpandFull sets expand mode directly to full.
func (m *Model) SetExpandFull() {
	m.expandMode = constants.ExpandFull
}

// ResetExpand resets expand mode to normal.
func (m *Model) ResetExpand() {
	m.expandMode = constants.ExpandNormal
}

// ExpandMode returns the current expand mode.
func (m Model) ExpandMode() constants.ExpandMode { return m.expandMode }

// Content returns the current string content (empty if child model is set).
func (m Model) Content() string { return m.content }

// Update forwards messages to the child model when focused, if one is set.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.child != nil && m.focused {
		var cmd tea.Cmd
		m.child, cmd = m.child.Update(msg)
		return m, cmd
	}
	return m, nil
}

// SetTitle updates the panel title without recreating the model.
func (m *Model) SetTitle(title string) {
	m.title = title
}

// SetFocused sets the focus state and triggers a border transition animation.
func (m *Model) SetFocused(focused bool) {
	if m.focused != focused {
		m.focused = focused
		m.transitioning = true
		m.transitionTick = 0
	}
}

// TransitionCmd returns a tea.Cmd to start the border transition tick loop.
// Call this after SetFocused when you want animation. Returns nil if no transition is active.
func (m *Model) TransitionCmd() tea.Cmd {
	if !m.transitioning {
		return nil
	}
	return tea.Tick(transitionInterval, func(time.Time) tea.Msg {
		return TransitionTickMsg{}
	})
}

// AdvanceTransition advances the transition animation by one frame.
// Returns a tea.Cmd for the next tick, or nil when the transition completes.
func (m *Model) AdvanceTransition() tea.Cmd {
	if !m.transitioning {
		return nil
	}
	m.transitionTick++
	if m.transitionTick >= transitionFrames {
		m.transitioning = false
		return nil
	}
	return tea.Tick(transitionInterval, func(time.Time) tea.Msg {
		return TransitionTickMsg{}
	})
}

// SetDimensions updates panel width and height.
func (m *Model) SetDimensions(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = bodyWidth(width)
	m.viewport.Height = bodyHeight(height)
	// Re-evaluate overflow only for string content (child models handle their own sizing)
	if m.child == nil {
		m.viewport.SetContent(m.content)
		lines := strings.Count(m.content, "\n")
		if !strings.HasSuffix(m.content, "\n") && len(m.content) > 0 {
			lines++
		}
		m.overflow = lines > m.viewport.Height
	}
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// ScrollDown scrolls the viewport down one line.
func (m *Model) ScrollDown() {
	m.viewport.ScrollDown(1)
}

// ScrollUp scrolls the viewport up one line.
func (m *Model) ScrollUp() {
	m.viewport.ScrollUp(1)
}

// ScrollPosition returns the current scroll offset and total content lines.
func (m *Model) ScrollPosition() (current, total int) {
	return m.viewport.YOffset, m.viewport.TotalLineCount()
}

// View renders the panel with border and title.
func (m Model) View() string {
	style := m.borderStyle()

	titleText := m.title
	if m.overflow {
		cur, total := m.ScrollPosition()
		titleText = fmt.Sprintf("%s (%d/%d)", m.title, cur+1, total)
	}

	titleStyle := m.ctx.Styles.Header.Root
	header := titleStyle.Render(titleText)

	var body string
	if m.child != nil {
		body = m.child.View()
	} else {
		body = m.viewport.View()
	}

	inner := lipgloss.JoinVertical(lipgloss.Left, header, body)
	return style.Width(m.width).Render(inner)
}

func (m Model) borderStyle() lipgloss.Style {
	if !m.transitioning {
		if m.focused {
			return m.ctx.Styles.Panel.Focused
		}
		return m.ctx.Styles.Panel.Root
	}

	// During transition, interpolate between unfocused and focused styles.
	// Frame 0: unfocused border, Frame 1: primary border (mid), Frame 2: focused border.
	progress := float64(m.transitionTick+1) / float64(transitionFrames)
	if !m.focused {
		// Reverse: going from focused to unfocused
		progress = 1.0 - progress
	}

	switch {
	case progress < 0.33:
		return m.ctx.Styles.Panel.Root
	case progress < 0.66:
		// Mid-point: use primary border with rounded border (between the two)
		return m.ctx.Styles.Panel.Border.Border(lipgloss.RoundedBorder())
	default:
		return m.ctx.Styles.Panel.Focused
	}
}

func bodyHeight(panelHeight int) int {
	return max(panelHeight-3, 1) // border (2) + title (1)
}

func bodyWidth(panelWidth int) int {
	return max(panelWidth-2, 1) // border (2)
}
