// Package streamchart wraps ntcharts TimeSeriesChart for live streaming data with braille rendering.
package streamchart

import (
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model wraps a TimeSeriesChart for streaming use.
type Model struct {
	ctx   *tuictx.ProgramContext
	title string
	unit  string
	chart tslc.Model
}

// NewModel creates a streaming chart with braille rendering.
func NewModel(ctx *tuictx.ProgramContext, title, unit string, width, height int) Model {
	cs := ctx.Styles.Chart
	chart := tslc.New(width, height,
		tslc.WithXLabelFormatter(tslc.HourTimeLabelFormatter()),
		tslc.WithUpdateHandler(tslc.SecondNoZoomUpdateHandler(10)),
		tslc.WithAxesStyles(cs.Axis, cs.Label),
		tslc.WithStyle(cs.Line),
		tslc.WithLineStyle(runes.ArcLineStyle),
	)

	return Model{
		ctx:   ctx,
		title: title,
		unit:  unit,
		chart: chart,
	}
}

// Push adds a timestamped data point and redraws.
func (m *Model) Push(t time.Time, value float64) {
	m.chart.Push(tslc.TimePoint{Time: t, Value: value})
	m.chart.DrawBrailleAll()
}

// Resize updates chart dimensions and redraws.
func (m *Model) Resize(width, height int) {
	m.chart.Resize(width, height)
	if m.chart.Width() > 0 {
		m.chart.DrawBrailleAll()
	}
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
