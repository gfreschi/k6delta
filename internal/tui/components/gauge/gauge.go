// Package gauge provides a horizontal progress bar with threshold coloring.
package gauge

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model is the gauge Bubble Tea model.
type Model struct {
	ctx     *tuictx.ProgramContext
	label   string
	width   int
	value   float64
	max     float64
	hasData bool
}

// NewModel creates a gauge with a label and bar width.
func NewModel(ctx *tuictx.ProgramContext, label string, barWidth int) Model {
	return Model{ctx: ctx, label: label, width: barWidth, max: 100}
}

// SetValue updates the current and max values.
func (m *Model) SetValue(value, max float64) {
	m.value = value
	m.max = max
	m.hasData = true
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the gauge bar.
func (m Model) View() string {
	labelStr := fmt.Sprintf("%-8s", m.label)
	if !m.hasData {
		bar := m.ctx.Styles.Common.FaintTextStyle.Render(strings.Repeat("░", m.width))
		return fmt.Sprintf("%s %s  %s", labelStr, bar, m.ctx.Styles.Common.FaintTextStyle.Render("—"))
	}

	pct := m.value / m.max
	if pct > 1 {
		pct = 1
	}
	filled := int(float64(m.width) * pct)
	empty := m.width - filled

	color := m.thresholdColor(pct)
	bar := color.Render(strings.Repeat("▰", filled)) +
		m.ctx.Styles.Common.FaintTextStyle.Render(strings.Repeat("░", empty))

	pctStr := fmt.Sprintf("%.0f%%", m.value/m.max*100)
	return fmt.Sprintf("%s %s  %s", labelStr, bar, pctStr)
}

func (m Model) thresholdColor(pct float64) lipgloss.Style {
	switch {
	case pct > 0.95:
		return m.ctx.Styles.Common.ErrorStyle
	case pct > 0.85:
		return m.ctx.Styles.Common.WarnStyle
	case pct > 0.70:
		return m.ctx.Styles.Common.SuccessStyle
	default:
		return m.ctx.Styles.Common.MainTextStyle
	}
}
