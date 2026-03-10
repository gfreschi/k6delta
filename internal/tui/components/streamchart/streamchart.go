// Package streamchart wraps ntcharts TimeSeriesChart for live streaming data with braille rendering.
package streamchart

import (
	"fmt"
	"strings"
	"time"

	"github.com/NimbleMarkets/ntcharts/canvas/runes"
	tslc "github.com/NimbleMarkets/ntcharts/linechart/timeserieslinechart"
	"github.com/charmbracelet/lipgloss"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// maxAnnotations is the maximum number of annotations shown below the chart.
const maxAnnotations = 3

// Annotation marks a point in time on the chart with a label and style.
type Annotation struct {
	Time  time.Time
	Label string
	Style lipgloss.Style
}

// Model wraps a TimeSeriesChart for streaming use.
type Model struct {
	ctx         *tuictx.ProgramContext
	title       string
	unit        string
	chart       tslc.Model
	annotations []Annotation
	startTime   time.Time
	idle        bool
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
	if m.startTime.IsZero() {
		m.startTime = t
	}
	m.chart.Push(tslc.TimePoint{Time: t, Value: value})
	m.chart.DrawBrailleAll()
}

// AddAnnotation adds a marker to the chart timeline.
// Only the most recent annotations are retained.
func (m *Model) AddAnnotation(a Annotation) {
	m.annotations = append(m.annotations, a)
	if len(m.annotations) > maxAnnotations*2 {
		m.annotations = m.annotations[len(m.annotations)-maxAnnotations:]
	}
}

// Resize updates chart dimensions and redraws.
func (m *Model) Resize(width, height int) {
	m.chart.Resize(width, height)
	if m.chart.Width() > 0 {
		m.chart.DrawBrailleAll()
	}
}

// SetIdle marks the chart as paused (no new data flowing).
func (m *Model) SetIdle(idle bool) {
	m.idle = idle
}

// UpdateContext updates the shared program context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the chart with a title header and annotation markers.
func (m Model) View() string {
	s := m.ctx.Styles
	headerText := "─ " + m.title + " (" + m.unit + ") "
	if m.idle {
		headerText += s.Common.FaintTextStyle.Render("(paused)")
	}
	header := s.Header.Root.Render(headerText)
	parts := []string{header, m.chart.View()}

	if ann := m.renderAnnotations(); ann != "" {
		parts = append(parts, ann)
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderAnnotations renders the last maxAnnotations as inline markers.
func (m Model) renderAnnotations() string {
	if len(m.annotations) == 0 {
		return ""
	}

	// Show only the last maxAnnotations
	start := 0
	if len(m.annotations) > maxAnnotations {
		start = len(m.annotations) - maxAnnotations
	}

	recent := m.annotations[start:]
	var parts []string
	for _, a := range recent {
		offset := ""
		if !m.startTime.IsZero() {
			elapsed := a.Time.Sub(m.startTime).Truncate(time.Second)
			offset = fmt.Sprintf("@%s ", elapsed)
		}
		marker := a.Style.Render("▼ " + offset + a.Label)
		parts = append(parts, marker)
	}

	return strings.Join(parts, "  ")
}
