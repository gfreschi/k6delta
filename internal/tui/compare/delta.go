package comparetui

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/components/metriccard"
	"github.com/gfreschi/k6delta/internal/tui/components/overlay"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

type sortMode int

const (
	sortDefault sortMode = iota
	sortWorstFirst
	sortBestFirst
)

func (s sortMode) String() string {
	switch s {
	case sortWorstFirst:
		return "worst first"
	case sortBestFirst:
		return "best first"
	default:
		return "default"
	}
}

// --- Export ---

func (m Model) exportComparison() tea.Cmd {
	return func() tea.Msg {
		data, err := report.CompareReportsJSON(m.pathA, m.pathB)
		if err != nil {
			return errMsg{err: fmt.Errorf("export comparison: %w", err)}
		}
		path := "comparison-report.json"
		if writeErr := os.WriteFile(path, data, 0o644); writeErr != nil {
			return errMsg{err: fmt.Errorf("write comparison: %w", writeErr)}
		}
		return exportDoneMsg{path: path}
	}
}

// --- Delta KPI Strip ---

func (m Model) renderDeltaStrip() string {
	if m.result == nil {
		return ""
	}

	tileW := constants.TileWidth(m.ctx.ContentWidth)
	if tileW == 0 {
		return ""
	}
	width := m.ctx.ContentWidth
	tilesPerRow := max(width/(tileW+2), 1)

	allRows := make([]report.ComparisonRow, 0, len(m.result.K6Rows)+len(m.result.InfraRows))
	allRows = append(allRows, m.result.K6Rows...)
	allRows = append(allRows, m.result.InfraRows...)
	var tiles []string
	for _, row := range allRows {
		if row.Direction == "" {
			continue
		}
		pct := math.Abs(parsePctChange(row.Delta))
		label := metricShortName(row.Metric)
		t := metriccard.NewModel(m.ctx, label, "%", tileW)
		t.SetValue(pct, 50) // max 50% for severity scaling
		if row.Direction == "better" {
			t.SetSeverity(metriccard.SeverityOK)
		} else if pct >= 5 {
			t.SetSeverity(metriccard.SeverityErr)
		} else {
			t.SetSeverity(metriccard.SeverityWarn)
		}
		tiles = append(tiles, t.View())
		if len(tiles) >= 6 {
			break
		}
	}

	if len(tiles) == 0 {
		return ""
	}
	return common.RenderTileGrid(tiles, tilesPerRow)
}

func metricShortName(metric string) string {
	switch metric {
	case "p95":
		return "p95"
	case "p90":
		return "p90"
	case "error_rate":
		return "err"
	case "throughput":
		return "rps"
	case "checks_rate":
		return "chk"
	case "total_requests":
		return "reqs"
	case "service_cpu_peak":
		return "cpu"
	case "service_memory_peak":
		return "mem"
	case "alb_5xx":
		return "5xx"
	default:
		if len(metric) > 4 {
			return metric[:4]
		}
		return metric
	}
}

// --- Regression Verdict Tile ---

func (m Model) renderRegressionVerdict() string {
	if m.result == nil {
		return ""
	}
	sum := computeSummary(m.result)
	s := m.ctx.Styles
	width := m.ctx.ContentWidth

	var word string
	var borderStyle, textStyle lipgloss.Style

	switch {
	case sum.regressed == 0:
		word = "N O   R E G R E S S I O N"
		borderStyle = s.Tile.BorderOK
		textStyle = s.Verdict.Pass
	case sum.worstPct < 5 && sum.worstPct > -5:
		word = "M I N O R   R E G R E S S I O N"
		borderStyle = s.Tile.BorderWarn
		textStyle = s.Verdict.Warn
	default:
		word = "R E G R E S S I O N"
		borderStyle = s.Tile.BorderError
		textStyle = s.Verdict.Fail
	}

	titleLine := textStyle.Render("  " + word)
	countLine := s.Common.FaintTextStyle.Render(fmt.Sprintf("  %d improved · %d regressed · %d unchanged",
		sum.improved, sum.regressed, sum.unchanged))

	inner := lipgloss.JoinVertical(lipgloss.Left, titleLine, countLine)
	return borderStyle.Width(width).Render(inner)
}

// --- Inline Delta Badges ---

func formatDeltaCell(row report.ComparisonRow, tiers common.DeltaStyleTiers, wide bool) string {
	if row.Direction == "" {
		return constants.IconQuiet
	}

	pct := parsePctChange(row.Delta)
	absPct := math.Abs(pct)
	style := common.DeltaStyle(tiers, pct, isLowerBetter(row.Metric))

	if absPct < 2 {
		return style.Render(constants.IconQuiet)
	}

	var icon string
	switch row.Direction {
	case "better":
		icon = constants.IconDone
	case "worse":
		icon = constants.IconWarning
	}

	text := row.Delta
	if wide {
		if pct > 0 {
			text += " " + constants.IconUp
		} else {
			text += " " + constants.IconDown
		}
	}

	return style.Render(icon + " " + text)
}

// --- Drill-Down Rendering ---

func (m Model) renderDrillDown() string {
	if m.result == nil || !m.drillActive {
		return ""
	}

	var rows []report.ComparisonRow
	if m.focusMgr.Current() == 0 {
		rows = m.sortedRows(m.result.K6Rows)
	} else {
		rows = m.result.InfraRows
	}

	s := m.ctx.Styles
	width := m.ctx.ContentWidth - 4
	var lines []string

	tiers := s.Delta.Tiers()
	barWidth := max(width-20, 10)
	for _, row := range rows {
		label, valA, valB := formatK6Row(row)
		badge := formatDeltaCell(row, tiers, true)

		hdr := s.Common.BoldStyle.Render(label) + "  " + badge
		lines = append(lines, hdr)

		// A/B values with proportional bar visualization
		numA := parseNumericValue(row.ValueA)
		numB := parseNumericValue(row.ValueB)
		maxVal := max(numA, numB)
		ratioA, ratioB := 0.0, 0.0
		if maxVal > 0 {
			ratioA = numA / maxVal
			ratioB = numB / maxVal
		}
		lines = append(lines, renderABBar("A", valA, ratioA, barWidth, s))
		lines = append(lines, renderABBar("B", valB, ratioB, barWidth, s))
		lines = append(lines, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderABBar(label, value string, ratio float64, barWidth int, s tuictx.Styles) string {
	prefix := s.Common.FaintTextStyle.Render(fmt.Sprintf("  %s: %-12s", label, value))
	fillLen := max(int(float64(barWidth)*ratio), 1)
	bar := strings.Repeat("\u2593", fillLen)
	return prefix + " " + s.Common.FaintTextStyle.Render(bar)
}

func parseNumericValue(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

// --- Side-by-Side Rendering ---

func (m Model) renderSideBySide() string {
	if m.result == nil {
		return ""
	}
	width := m.ctx.ContentWidth
	halfW := width / 2

	// Left panel: Run A
	leftTitle := m.ctx.Styles.Common.BoldStyle.Render(fmt.Sprintf("Run A: %s", m.result.RunA.Start))
	leftTable := m.renderSingleRunTable(func(r report.ComparisonRow) string { return r.ValueA })

	// Right panel: Run B
	rightTitle := m.ctx.Styles.Common.BoldStyle.Render(fmt.Sprintf("Run B: %s", m.result.RunB.Start))
	rightTable := m.renderSingleRunTable(func(r report.ComparisonRow) string { return r.ValueB })

	col := m.ctx.Styles.Layout.Column
	leftCol := col.Width(halfW).Render(
		lipgloss.JoinVertical(lipgloss.Left, leftTitle, leftTable))
	rightCol := col.Width(width - halfW).Render(
		lipgloss.JoinVertical(lipgloss.Left, rightTitle, rightTable))

	return lipgloss.JoinHorizontal(lipgloss.Top, leftCol, rightCol)
}

func (m Model) renderSingleRunTable(valueFn func(report.ComparisonRow) string) string {
	k6Sorted := m.sortedRows(m.result.K6Rows)
	var rows []table.Row
	for _, row := range k6Sorted {
		label, _, _ := formatK6Row(row)
		val := valueFn(row)
		rows = append(rows, table.Row{label, val})
	}
	halfW := m.ctx.ContentWidth / 2
	tbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Metric", Width: min(22, halfW-16)},
		{Title: "Value", Width: 12, Align: lipgloss.Right},
	})
	tbl.SetRows(rows)
	return tbl.View()
}

// --- Table Rendering ---

func (m Model) renderK6Table() string {
	k6Sorted := m.sortedRows(m.result.K6Rows)
	tiers := m.ctx.Styles.Delta.Tiers()
	wide := m.ctx.ContentWidth >= constants.BreakpointNarrow
	var k6Rows []table.Row
	for _, row := range k6Sorted {
		label, valA, valB := formatK6Row(row)
		delta := formatDeltaCell(row, tiers, wide)
		k6Rows = append(k6Rows, table.Row{label, valA, valB, delta})
	}
	tbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Metric", Width: 22},
		{Title: "Run A", Width: 12, Align: lipgloss.Right},
		{Title: "Run B", Width: 12, Align: lipgloss.Right},
		{Title: "Delta", Width: 20, Align: lipgloss.Right},
	})
	tbl.SetRows(k6Rows)
	return tbl.View()
}

func (m Model) renderInfraTable() string {
	var cpuRow, memRow, alb5xxRow *report.ComparisonRow
	var tasksBefore, tasksAfter, asgBefore, asgAfter *report.ComparisonRow

	for i := range m.result.InfraRows {
		row := &m.result.InfraRows[i]
		switch row.Metric {
		case "service_cpu_peak":
			cpuRow = row
		case "service_memory_peak":
			memRow = row
		case "tasks_before":
			tasksBefore = row
		case "tasks_after":
			tasksAfter = row
		case "asg_before":
			asgBefore = row
		case "asg_after":
			asgAfter = row
		case "alb_5xx":
			alb5xxRow = row
		}
	}

	tiers := m.ctx.Styles.Delta.Tiers()
	wide := m.ctx.ContentWidth >= constants.BreakpointNarrow

	var infraRows []table.Row
	if cpuRow != nil && (cpuRow.ValueA != "-" || cpuRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS CPU peak", cpuRow.ValueA + "%", cpuRow.ValueB + "%", formatDeltaCell(*cpuRow, tiers, wide)})
	}
	if memRow != nil && (memRow.ValueA != "-" || memRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS memory peak", memRow.ValueA + "%", memRow.ValueB + "%", formatDeltaCell(*memRow, tiers, wide)})
	}
	if tasksBefore != nil && tasksAfter != nil {
		infraRows = append(infraRows, table.Row{"Tasks (before/after)",
			tasksBefore.ValueA + "/" + tasksAfter.ValueA,
			tasksBefore.ValueB + "/" + tasksAfter.ValueB, ""})
	}
	if asgBefore != nil && asgAfter != nil {
		infraRows = append(infraRows, table.Row{"ASG (before/after)",
			asgBefore.ValueA + "/" + asgAfter.ValueA,
			asgBefore.ValueB + "/" + asgAfter.ValueB, ""})
	}
	if alb5xxRow != nil && (alb5xxRow.ValueA != "-" || alb5xxRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ALB 5xx total", alb5xxRow.ValueA, alb5xxRow.ValueB, formatDeltaCell(*alb5xxRow, tiers, wide)})
	}

	if len(infraRows) == 0 {
		return "  No infrastructure data"
	}

	tbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Infrastructure", Width: 22},
		{Title: "Run A", Width: 12, Align: lipgloss.Right},
		{Title: "Run B", Width: 12, Align: lipgloss.Right},
		{Title: "Delta", Width: 20, Align: lipgloss.Right},
	})
	tbl.SetRows(infraRows)
	return tbl.View()
}

// --- Sort ---

func (m Model) sortedRows(rows []report.ComparisonRow) []report.ComparisonRow {
	if m.sort == sortDefault {
		return rows
	}
	sorted := make([]report.ComparisonRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		absI := math.Abs(parsePctChange(sorted[i].Delta))
		absJ := math.Abs(parsePctChange(sorted[j].Delta))
		if m.sort == sortWorstFirst {
			return absI > absJ
		}
		return absI < absJ
	})
	return sorted
}

// --- Helpers ---

func formatK6Row(row report.ComparisonRow) (label, valA, valB string) {
	switch row.Metric {
	case "p95":
		return "p95 latency", row.ValueA + "ms", row.ValueB + "ms"
	case "p90":
		return "p90 latency", row.ValueA + "ms", row.ValueB + "ms"
	case "error_rate":
		return "Error rate", row.ValueA, row.ValueB
	case "throughput":
		return "Throughput", row.ValueA + "/s", row.ValueB + "/s"
	case "checks_rate":
		return "Checks rate", row.ValueA, row.ValueB
	case "total_requests":
		return "Total requests", row.ValueA, row.ValueB
	default:
		return row.Metric, row.ValueA, row.ValueB
	}
}

func parsePctChange(delta string) float64 {
	s := strings.TrimSuffix(delta, "%")
	s = strings.TrimPrefix(s, "+")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func isLowerBetter(metric string) bool {
	switch metric {
	case "p95", "p90", "error_rate", "service_cpu_peak", "service_memory_peak", "alb_5xx":
		return true
	default:
		return false
	}
}

func (m Model) renderHelpOverlay() string {
	return overlay.RenderHelp(m.ctx, []overlay.HelpGroup{
		{Title: "Navigation", Keys: [][2]string{
			{"q", "Quit"},
			{"?", "Toggle help"},
			{"esc", "Close help / close drill-down / collapse panel"},
		}},
		{Title: "Panels", Keys: [][2]string{
			{"tab / shift+tab", "Next / previous panel"},
			{"1-2", "Jump to panel"},
			{"+", "Toggle expand (normal / full)"},
			{"enter", "Drill into focused panel (A/B detail)"},
			{"↑↓ / j k", "Scroll focused panel"},
		}},
		{Title: "Actions", Keys: [][2]string{
			{"s", "Cycle sort (default → worst → best)"},
			{"d", "Toggle side-by-side diff (≥140 wide)"},
			{"e", "Export comparison JSON"},
		}},
	})
}

// --- Summary ---

type comparisonSummary struct {
	improved    int
	regressed   int
	unchanged   int
	worstMetric string
	worstPct    float64
}

func computeSummary(result *report.ComparisonResult) comparisonSummary {
	var sum comparisonSummary
	allRows := make([]report.ComparisonRow, 0, len(result.K6Rows)+len(result.InfraRows))
	allRows = append(allRows, result.K6Rows...)
	allRows = append(allRows, result.InfraRows...)
	for _, row := range allRows {
		switch row.Direction {
		case "better":
			sum.improved++
		case "worse":
			sum.regressed++
			pct := math.Abs(parsePctChange(row.Delta))
			if pct > math.Abs(sum.worstPct) {
				sum.worstPct = parsePctChange(row.Delta)
				sum.worstMetric = row.Metric
			}
		default:
			sum.unchanged++
		}
	}
	return sum
}

