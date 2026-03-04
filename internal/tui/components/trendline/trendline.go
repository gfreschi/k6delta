// Package trendline wraps ntcharts Sparkline for compact inline trend visualization.
package trendline

import (
	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/charmbracelet/lipgloss"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model wraps a sparkline for compact braille trend display.
type Model struct {
	ctx   *tuictx.ProgramContext
	spark sparkline.Model
}

// NewModel creates a compact sparkline with braille rendering.
func NewModel(ctx *tuictx.ProgramContext, width, height int) Model {
	style := lipgloss.NewStyle().
		Foreground(ctx.Theme.PrimaryText)

	spark := sparkline.New(width, height,
		sparkline.WithMaxValue(100),
		sparkline.WithStyle(style),
	)

	return Model{
		ctx:   ctx,
		spark: spark,
	}
}

// Push adds a value and redraws with braille.
func (m *Model) Push(value float64) {
	m.spark.Push(value)
	m.spark.DrawBraille()
}

// SetMax updates the maximum value for scaling.
func (m *Model) SetMax(max float64) {
	m.spark.SetMax(max)
}

// Resize updates sparkline dimensions and redraws.
func (m *Model) Resize(width, height int) {
	m.spark.Resize(width, height)
	m.spark.DrawBraille()
}

// UpdateContext updates the shared program context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View returns the rendered sparkline string.
func (m Model) View() string {
	return m.spark.View()
}
