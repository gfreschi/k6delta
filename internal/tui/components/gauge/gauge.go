// Package gauge provides a horizontal progress bar with threshold coloring and animated fill.
package gauge

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/tui/common"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// TickMsg triggers gauge animation frame.
type TickMsg time.Time

// Model is the gauge Bubble Tea model with animated fill.
type Model struct {
	ctx     *tuictx.ProgramContext
	label   string
	width   int
	current float64 // target ratio (0.0-1.0)
	display float64 // animated display ratio (lerps toward current)
	max     float64
	hasData bool
}

// NewModel creates a gauge with a label and bar width.
func NewModel(ctx *tuictx.ProgramContext, label string, barWidth int) Model {
	return Model{ctx: ctx, label: label, width: barWidth, max: 100}
}

// SetValue updates the target value. Animation will lerp display toward it.
func (m *Model) SetValue(value, max float64) {
	m.max = max
	if max > 0 {
		m.current = value / max
	}
	m.hasData = true
	// Until animation ticks are wired (0.1.5e), snap display to current
	m.display = m.current
}

// Init returns nil (gauge ticks are driven by parent).
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles TickMsg to animate the gauge fill.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if _, ok := msg.(TickMsg); ok && m.hasData {
		m.display = common.Lerp(m.display, m.current, 0.3)
	}
	return m, nil
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the gauge bar with animated fill.
func (m Model) View() string {
	labelStr := fmt.Sprintf("%-8s", m.label)
	if !m.hasData {
		bar := m.ctx.Styles.Common.FaintTextStyle.Render(strings.Repeat("░", m.width))
		return fmt.Sprintf("%s %s  %s", labelStr, bar, m.ctx.Styles.Common.FaintTextStyle.Render("—"))
	}

	pct := m.display
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(float64(m.width) * pct)
	empty := m.width - filled

	color := m.thresholdColor(pct)
	bar := color.Render(strings.Repeat("▰", filled)) +
		m.ctx.Styles.Common.FaintTextStyle.Render(strings.Repeat("░", empty))

	pctStr := fmt.Sprintf("%.0f%%", m.current*100)
	return fmt.Sprintf("%s %s  %s", labelStr, bar, pctStr)
}

func (m Model) thresholdColor(pct float64) lipgloss.Style {
	switch {
	case pct >= 0.95:
		return m.ctx.Styles.Common.ErrorStyle
	case pct > 0.85:
		return m.ctx.Styles.Common.WarnStyle
	case pct > 0.70:
		return m.ctx.Styles.Common.SuccessStyle
	default:
		return m.ctx.Styles.Common.MainTextStyle
	}
}
