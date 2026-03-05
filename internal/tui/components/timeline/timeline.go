// Package timeline provides a multi-lane sparkline timeline with event spans.
package timeline

import (
	"fmt"
	"strings"
	"time"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// EventType categorizes timeline events.
type EventType int

const (
	EventAlarm    EventType = iota
	EventScaling
	EventResolved
)

// Block sparkline characters.
var blocks = []rune{'\u2581', '\u2582', '\u2583', '\u2584', '\u2585', '\u2586', '\u2587', '\u2588'}

// Lane is a metric data lane in the timeline.
type Lane struct {
	Label     string
	Values    []float64
	Peak      float64
	Unit      string
	Threshold float64 // optional: 0 means no threshold line
}

// Event is a scaling/alarm event with duration.
type Event struct {
	Start time.Time
	End   time.Time
	Type  EventType
	Label string
}

// Model is the timeline view model.
type Model struct {
	ctx    *tuictx.ProgramContext
	start  time.Time
	end    time.Time
	width  int
	lanes  []Lane
	events []Event
}

// NewModel creates a timeline for the given time range.
func NewModel(ctx *tuictx.ProgramContext, start, end time.Time, width int) Model {
	return Model{
		ctx:   ctx,
		start: start,
		end:   end,
		width: width,
	}
}

// AddLane adds a metric data lane.
func (m *Model) AddLane(lane Lane) {
	m.lanes = append(m.lanes, lane)
}

// AddEvent adds a scaling/alarm event.
func (m *Model) AddEvent(event Event) {
	m.events = append(m.events, event)
}

// Resize updates the timeline width.
func (m *Model) Resize(width int) {
	m.width = width
}

// UpdateContext updates the shared program context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the multi-lane timeline.
// Graceful degradation: >=60 full, 40-59 abbreviated, <40 event list.
func (m Model) View() string {
	if m.width < 40 {
		return m.viewEventList()
	}
	if m.width < 60 {
		return m.viewAbbreviated()
	}

	var lines []string

	lines = append(lines, m.renderTimeAxis(), "")

	for _, lane := range m.lanes {
		lines = append(lines, m.renderLane(lane))
		if lane.Threshold > 0 {
			lines = append(lines, m.renderThresholdLine(lane))
		}
		lines = append(lines, "")
	}

	if len(m.events) > 0 {
		lines = append(lines, m.renderEventLane(), "", m.renderLegend())
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderTimeAxis() string {
	s := m.ctx.Styles.Timeline
	duration := m.end.Sub(m.start)
	axisW := m.width - 8

	tickCount := 5
	if axisW < 40 {
		tickCount = 3
	}

	var axis strings.Builder
	axis.WriteString("  ")
	for i := 0; i < tickCount; i++ {
		offset := time.Duration(int64(float64(duration) * float64(i) / float64(tickCount-1)))
		t := m.start.Add(offset)
		label := t.Format("15:04")
		spacing := axisW/tickCount - len(label)
		if spacing < 1 {
			spacing = 1
		}
		axis.WriteString(s.Lane.Render(label))
		if i < tickCount-1 {
			axis.WriteString(strings.Repeat(" ", spacing))
		}
	}

	return axis.String()
}

func (m Model) renderLane(lane Lane) string {
	s := m.ctx.Styles.Timeline
	sparkW := m.width - 20

	spark := renderBlockSparkline(lane.Values, sparkW, lane.Peak)
	peakStr := fmt.Sprintf("%.0f%s peak", lane.Peak, lane.Unit)

	return fmt.Sprintf("  %s  %s  %s",
		s.Lane.Width(4).Render(lane.Label),
		spark,
		s.Lane.Render(peakStr),
	)
}

func (m Model) renderThresholdLine(lane Lane) string {
	s := m.ctx.Styles.Timeline
	lineW := m.width - 20
	label := fmt.Sprintf("%.0f%s threshold", lane.Threshold, lane.Unit)
	dashLen := (lineW - len(label)) / 2
	if dashLen < 1 {
		dashLen = 1
	}
	dashes := strings.Repeat("\u254c", dashLen)
	return fmt.Sprintf("       %s",
		s.Threshold.Render(dashes+" "+label+" "+dashes),
	)
}

func (m Model) renderEventLane() string {
	s := m.ctx.Styles.Timeline
	laneW := m.width - 20
	if laneW < 1 {
		laneW = 1
	}
	duration := m.end.Sub(m.start)
	if duration <= 0 {
		return ""
	}

	lane := make([]rune, laneW)
	for i := range lane {
		lane[i] = '\u00b7'
	}

	for _, ev := range m.events {
		startPos := int(float64(ev.Start.Sub(m.start)) / float64(duration) * float64(laneW))
		endPos := int(float64(ev.End.Sub(m.start)) / float64(duration) * float64(laneW))
		if startPos < 0 {
			startPos = 0
		}
		if endPos >= laneW {
			endPos = laneW - 1
		}

		startChar := '\u26a0'
		fillChar := '\u2500'
		endChar := '\u2713'
		if ev.Type == EventScaling {
			startChar = '\u25b8'
		}

		if startPos < laneW {
			lane[startPos] = startChar
		}
		for i := startPos + 1; i < endPos; i++ {
			if i < laneW {
				lane[i] = fillChar
			}
		}
		if endPos > startPos && endPos < laneW {
			lane[endPos] = endChar
		}
	}

	return fmt.Sprintf("  %s  %s",
		s.Lane.Width(4).Render("events"),
		s.Alarm.Render(string(lane)),
	)
}

func (m Model) renderLegend() string {
	s := m.ctx.Styles.Timeline
	return fmt.Sprintf("  %s  %s  %s  %s",
		s.Alarm.Render("\u26a0 alarm"),
		s.Scaling.Render("\u25b8 scaling"),
		s.Resolved.Render("\u2713 resolved"),
		s.Lane.Render("\u00b7 quiet"),
	)
}

func (m Model) viewEventList() string {
	var lines []string
	for _, ev := range m.events {
		lines = append(lines, fmt.Sprintf("  %s  %s", ev.Start.Format("15:04"), ev.Label))
	}
	if len(lines) == 0 {
		return "  No events"
	}
	return strings.Join(lines, "\n")
}

func (m Model) viewAbbreviated() string {
	var lines []string
	for _, lane := range m.lanes {
		lines = append(lines, m.renderLane(lane))
	}
	if len(m.events) > 0 {
		lines = append(lines, m.renderEventLane())
	}
	return strings.Join(lines, "\n")
}

func renderBlockSparkline(values []float64, width int, maxVal float64) string {
	if len(values) == 0 || maxVal == 0 {
		return strings.Repeat(" ", width)
	}

	data := resample(values, width)

	var sb strings.Builder
	for _, v := range data {
		idx := int(v / maxVal * 7)
		if idx > 7 {
			idx = 7
		}
		if idx < 0 {
			idx = 0
		}
		sb.WriteRune(blocks[idx])
	}
	return sb.String()
}

func resample(values []float64, targetLen int) []float64 {
	if len(values) == 0 || targetLen <= 0 {
		return nil
	}
	if len(values) <= targetLen {
		result := make([]float64, targetLen)
		offset := targetLen - len(values)
		copy(result[offset:], values)
		return result
	}
	result := make([]float64, targetLen)
	ratio := float64(len(values)) / float64(targetLen)
	for i := range targetLen {
		idx := int(float64(i) * ratio)
		if idx >= len(values) {
			idx = len(values) - 1
		}
		result[i] = values[idx]
	}
	return result
}
