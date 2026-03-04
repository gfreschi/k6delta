// Package timechart wraps ntcharts TimeSeriesChart for static time-series graphs with braille rendering.
package timechart

import (
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model wraps a TimeSeriesChart for static data display.
type Model struct {
	ctx   *tuictx.ProgramContext
	title string
	unit  string
	chart tslc.Model
}

// NewModel creates a static time-series chart with braille rendering.
func NewModel(ctx *tuictx.ProgramContext, title, unit string, width, height int) Model {
	axisStyle := lipgloss.NewStyle().
		Foreground(ctx.Theme.FaintText)
	labelStyle := lipgloss.NewStyle().
		Foreground(ctx.Theme.FaintText)
	lineStyle := lipgloss.NewStyle().
		Foreground(ctx.Theme.PrimaryText)

	chart := tslc.New(width, height,
		tslc.WithXLabelFormatter(tslc.HourTimeLabelFormatter()),
		tslc.WithUpdateHandler(tslc.SecondNoZoomUpdateHandler(60)),
		tslc.WithAxesStyles(axisStyle, labelStyle),
		tslc.WithStyle(lineStyle),
		tslc.WithLineStyle(runes.ArcLineStyle),
	)

	return Model{
		ctx:   ctx,
		title: title,
		unit:  unit,
		chart: chart,
	}
}

// SetData loads a complete time-series and renders it.
func (m *Model) SetData(times []time.Time, values []float64) {
	m.chart.ClearAllData()
	n := len(times)
	if len(values) < n {
		n = len(values)
	}
	for i := 0; i < n; i++ {
		m.chart.Push(tslc.TimePoint{Time: times[i], Value: values[i]})
	}
	m.chart.DrawBrailleAll()
}

// Resize updates chart dimensions and redraws.
func (m *Model) Resize(width, height int) {
	m.chart.Resize(width, height)
	m.chart.DrawBrailleAll()
}

// UpdateContext updates the shared program context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the chart with a title header.
func (m Model) View() string {
	s := m.ctx.Styles
	header := s.Header.Root.Render("─ " + m.title + " (" + m.unit + ") ")
	return lipgloss.JoinVertical(lipgloss.Left, header, m.chart.View())
}
