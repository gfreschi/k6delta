// Package metriccard provides KPI tile components with severity-colored borders.
//
// Three variants controlled by the unit field:
//   - Percentage (unit="%"): gauge + block sparkline + value% + severity border
//   - Count (unit="count"): value + delta arrows + label (no gauge, no sparkline)
//   - Rate (anything else): block sparkline + value + label (no gauge)
package metriccard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Re-export severity types so callers can use metriccard.SeverityOK etc.
type Severity = common.Severity

// SeverityThresholds is a re-export of common.SeverityThresholds.
type SeverityThresholds = common.SeverityThresholds

const (
	SeverityOK   = common.SeverityOK
	SeverityWarn = common.SeverityWarn
	SeverityErr  = common.SeverityError
)

// DefaultSeverityThresholds returns the standard severity thresholds.
func DefaultSeverityThresholds() common.SeverityThresholds {
	return common.DefaultSeverityThresholds
}

// blockCharsRunes is the pre-computed rune slice for sparkline rendering.
var blockCharsRunes = []rune(blockChars)

// Block sparkline characters (U+2581 through U+2588).
const blockChars = "\u2581\u2582\u2583\u2584\u2585\u2586\u2587\u2588"

// Model is a KPI tile with bar, sparkline, or count display.
type Model struct {
	ctx        *tuictx.ProgramContext
	label      string
	unit       string
	width      int
	value      float64
	max        float64
	severity   common.Severity
	hasData    bool
	thresholds common.SeverityThresholds
	values     []float64 // sparkline data points
	delta      string    // for count variant: "5->3"

	// Report mode fields
	peak float64
	avg  float64
}

// NewModel creates a metric card tile with default severity thresholds.
func NewModel(ctx *tuictx.ProgramContext, label, unit string, width int) Model {
	return Model{
		ctx:        ctx,
		label:      label,
		unit:       unit,
		width:      width,
		thresholds: common.DefaultSeverityThresholds,
	}
}

// SetValue updates the tile for live mode.
func (m *Model) SetValue(value, max float64) {
	m.value = value
	m.max = max
	m.hasData = true
	m.severity = computeSeverity(value, max, m.thresholds.WarnRatio, m.thresholds.ErrorRatio)
}

// SetDelta sets the delta string for count-variant tiles (e.g., "5->3").
func (m *Model) SetDelta(delta string) {
	m.delta = delta
}

// PushSparkline appends a data point for the block sparkline.
func (m *Model) PushSparkline(value float64) {
	m.values = append(m.values, value)
	// Keep only enough points for the tile width minus borders/padding
	maxPoints := m.width - 4
	if maxPoints < 1 {
		maxPoints = 1
	}
	if len(m.values) > maxPoints {
		m.values = m.values[len(m.values)-maxPoints:]
	}
}

// SetReportData configures the tile for post-k6 report mode.
func (m *Model) SetReportData(peak, avg float64, values []float64) {
	m.peak = peak
	m.avg = avg
	m.values = values
	m.hasData = true
	m.value = peak
	if m.max > 0 {
		m.severity = computeSeverity(peak, m.max, m.thresholds.WarnRatio, m.thresholds.ErrorRatio)
	}
}

// SetSeverity overrides the auto-computed severity.
func (m *Model) SetSeverity(sev common.Severity) {
	m.severity = sev
}

// SetThresholds overrides the default severity thresholds.
func (m *Model) SetThresholds(th common.SeverityThresholds) {
	m.thresholds = th
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the tile with a severity-colored rounded border.
func (m Model) View() string {
	border := m.borderStyle()
	innerW := m.width - 2 // account for border chars

	if !m.hasData {
		content := m.ctx.Styles.Common.FaintTextStyle.Render(
			lipgloss.PlaceHorizontal(innerW, lipgloss.Center, "—"),
		)
		label := lipgloss.PlaceHorizontal(innerW, lipgloss.Center, m.label)
		return border.Width(m.width).Render(
			lipgloss.JoinVertical(lipgloss.Center, label, content),
		)
	}

	switch m.unit {
	case "%":
		return m.viewPercentage(border, innerW)
	case "count":
		return m.viewCount(border, innerW)
	default:
		return m.viewRate(border, innerW)
	}
}

func (m Model) viewPercentage(border lipgloss.Style, innerW int) string {
	// Value line: large percentage
	valStr := fmt.Sprintf("%.0f%%", m.value)
	valueLine := m.ctx.Styles.Common.BoldStyle.
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, valStr))

	// Inline gauge bar
	barLine := m.renderBar(innerW)

	// Block sparkline
	sparkLine := m.renderBlockSparkline(innerW)

	// Label
	labelLine := m.ctx.Styles.Common.FaintTextStyle.
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, m.label))

	lines := []string{valueLine, barLine}
	if sparkLine != "" {
		lines = append(lines, sparkLine)
	}
	lines = append(lines, labelLine)

	return border.Width(m.width).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) viewCount(border lipgloss.Style, innerW int) string {
	// Value line
	valStr := fmt.Sprintf("%.0f", m.value)
	valueLine := m.ctx.Styles.Common.BoldStyle.
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, valStr))

	// Delta arrows
	var deltaLine string
	if m.delta != "" {
		arrow := constants.IconArrowDn
		if m.severity == SeverityOK {
			arrow = constants.IconArrowUp
		}
		deltaLine = m.ctx.Styles.Common.FaintTextStyle.
			Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, arrow+" "+m.delta))
	}

	// Label
	labelLine := m.ctx.Styles.Common.FaintTextStyle.
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, m.label))

	lines := []string{valueLine}
	if deltaLine != "" {
		lines = append(lines, deltaLine)
	}
	lines = append(lines, labelLine)

	return border.Width(m.width).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) viewRate(border lipgloss.Style, innerW int) string {
	// Value line
	valStr := fmt.Sprintf("%.1f", m.value)
	if m.unit != "" {
		valStr += m.unit
	}
	valueLine := m.ctx.Styles.Common.BoldStyle.
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, valStr))

	// Block sparkline
	sparkLine := m.renderBlockSparkline(innerW)

	// Label
	labelLine := m.ctx.Styles.Common.FaintTextStyle.
		Render(lipgloss.PlaceHorizontal(innerW, lipgloss.Center, m.label))

	lines := []string{valueLine}
	if sparkLine != "" {
		lines = append(lines, sparkLine)
	}
	lines = append(lines, labelLine)

	return border.Width(m.width).Render(
		lipgloss.JoinVertical(lipgloss.Left, lines...),
	)
}

func (m Model) renderBar(width int) string {
	pct := 0.0
	if m.max > 0 {
		pct = m.value / m.max
	}
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(float64(width) * pct)
	empty := width - filled

	color := m.barColor(pct)
	bar := color.Render(strings.Repeat("\u25b0", filled)) +
		m.ctx.Styles.Common.FaintTextStyle.Render(strings.Repeat("\u2591", empty))
	return bar
}

func (m Model) barColor(pct float64) lipgloss.Style {
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

func (m Model) renderBlockSparkline(width int) string {
	if len(m.values) == 0 {
		return ""
	}

	// Find min/max for scaling
	minV, maxV := m.values[0], m.values[0]
	for _, v := range m.values {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}

	numLevels := len(blockCharsRunes)
	var b strings.Builder

	for _, v := range m.values {
		idx := 0
		if maxV > minV {
			idx = int((v - minV) / (maxV - minV) * float64(numLevels-1))
		}
		if idx >= numLevels {
			idx = numLevels - 1
		}
		b.WriteRune(blockCharsRunes[idx])
	}

	return lipgloss.PlaceHorizontal(width, lipgloss.Center, b.String())
}

func (m Model) borderStyle() lipgloss.Style {
	ts := m.ctx.Styles.Tile
	switch m.severity {
	case common.SeverityWarn:
		return ts.BorderWarn
	case common.SeverityError:
		return ts.BorderError
	default:
		return ts.BorderOK
	}
}

func computeSeverity(value, max, warnRatio, errorRatio float64) common.Severity {
	if max <= 0 {
		return common.SeverityOK
	}
	ratio := value / max
	return common.SeverityFromRatio(ratio, common.SeverityThresholds{
		WarnRatio:  warnRatio,
		ErrorRatio: errorRatio,
	})
}
