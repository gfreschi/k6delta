package comparetui

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/constants"
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

// --- Table Rendering ---

func (m Model) renderK6Table() string {
	k6Sorted := m.sortedRows(m.result.K6Rows)
	var k6Rows []table.Row
	for _, row := range k6Sorted {
		label, valA, valB := formatK6Row(row)
		delta := m.formatDelta(row.Delta, row.Direction, row.Metric)
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

	var infraRows []table.Row
	if cpuRow != nil && (cpuRow.ValueA != "-" || cpuRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS CPU peak", cpuRow.ValueA + "%", cpuRow.ValueB + "%", m.formatDelta(cpuRow.Delta, cpuRow.Direction, cpuRow.Metric)})
	}
	if memRow != nil && (memRow.ValueA != "-" || memRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS memory peak", memRow.ValueA + "%", memRow.ValueB + "%", m.formatDelta(memRow.Delta, memRow.Direction, memRow.Metric)})
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
		infraRows = append(infraRows, table.Row{"ALB 5xx total", alb5xxRow.ValueA, alb5xxRow.ValueB, m.formatDelta(alb5xxRow.Delta, alb5xxRow.Direction, alb5xxRow.Metric)})
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

func (m Model) formatDelta(delta, direction, metric string) string {
	if direction == "" {
		return delta
	}

	pct := parsePctChange(delta)
	tiers := m.ctx.Styles.Delta.Tiers()
	style := common.DeltaStyle(tiers, pct, isLowerBetter(metric))

	arrow := ""
	switch {
	case strings.HasPrefix(delta, "+"):
		arrow = " " + constants.IconUp
	case strings.HasPrefix(delta, "-"):
		arrow = " " + constants.IconDown
	}

	// Only show direction word when terminal is wide enough
	dirWord := ""
	if m.ctx.ContentWidth >= constants.BreakpointNarrow {
		switch direction {
		case "better":
			dirWord = " better"
		case "worse":
			dirWord = " worse"
		}
	}

	return style.Render(delta + arrow + dirWord)
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

func (m Model) renderSummary() string {
	sum := computeSummary(m.result)
	s := m.ctx.Styles

	improved := s.Common.SuccessStyle.Render(fmt.Sprintf("%d improved", sum.improved))
	regressed := s.Common.ErrorStyle.Render(fmt.Sprintf("%d regressed", sum.regressed))
	unchanged := s.Common.FaintTextStyle.Render(fmt.Sprintf("%d unchanged", sum.unchanged))

	line1 := fmt.Sprintf("  %s %s %s %s %s", improved, constants.IconBullet, regressed, constants.IconBullet, unchanged)

	if sum.worstMetric != "" {
		worst := s.Delta.WorseSevere.Render(fmt.Sprintf("%s %+.1f%%", sum.worstMetric, sum.worstPct))
		return line1 + "\n" + fmt.Sprintf("  Worst regression: %s", worst)
	}

	return line1
}
