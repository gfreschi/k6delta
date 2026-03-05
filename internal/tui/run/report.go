package runtui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/components/timechart"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	"github.com/gfreschi/k6delta/internal/verdict"
)

func (m Model) renderReport() string {
	s := m.ctx.Styles
	var sections []string

	sections = append(sections, s.Header.Title.Render("Load Test Report"), "")

	// Duration and k6 exit
	duration := m.endTime.Sub(m.startTime)
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60
	sections = append(sections, fmt.Sprintf("  %s%s -> %s  (%dm %ds)",
		s.Table.Label.Render("Duration"),
		m.startTime.Format("15:04:05"),
		m.endTime.Format("15:04:05"),
		minutes, seconds))

	exitStr := "0 (all thresholds passed)"
	exitStyle := s.Verdict.Pass
	if m.k6Result != nil && m.k6Result.ExitCode != 0 {
		exitStr = fmt.Sprintf("%d (threshold failures)", m.k6Result.ExitCode)
		exitStyle = s.Verdict.Warn
	}
	sections = append(sections, fmt.Sprintf("  %s%s",
		s.Table.Label.Render("k6 exit"),
		exitStyle.Render(exitStr)))

	// k6 metrics table
	if m.report != nil && m.report.K6 != nil {
		k6 := m.report.K6
		tbl := table.NewModel(m.ctx, []table.Column{
			{Title: "Metric", Width: 26},
			{Title: "Value", Width: 24},
		})
		tbl.SetRows([]table.Row{
			{"p95 latency", fmtFloatMs(k6.P95ms)},
			{"Error rate", fmtPct(k6.ErrorRate)},
			{"Throughput", fmtFloatRate(k6.RPSAvg)},
			{"Checks", fmtPctRate(k6.ChecksRate)},
			{"Thresholds", fmt.Sprintf("%d passed, %d failed", k6.Thresholds.Passed, k6.Thresholds.Failed)},
		})
		sections = append(sections, "", tbl.View())
	}

	// Infrastructure with delta column
	infraTbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Infrastructure", Width: 26},
		{Title: "Before", Width: 12},
		{Title: "After", Width: 12},
		{Title: "Delta", Width: 10},
	})
	infraTbl.SetRows([]table.Row{
		{"ECS tasks", fmt.Sprintf("%d", m.preSnapshot.TaskRunning), fmt.Sprintf("%d", m.postSnapshot.TaskRunning),
			fmtDelta(m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning)},
		{"EC2 instances", fmt.Sprintf("%d", m.preSnapshot.ASGInstances), fmt.Sprintf("%d", m.postSnapshot.ASGInstances),
			fmtDelta(m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances)},
	})
	sections = append(sections, "", infraTbl.View())

	// CloudWatch peaks
	if len(m.metrics) > 0 {
		var rows []table.Row
		for _, mr := range m.metrics {
			label := metricLabel(mr.ID)
			if label == "" {
				continue
			}
			if mr.Peak != nil && mr.Avg != nil {
				rows = append(rows, table.Row{label,
					fmtMetricValue(mr.ID, *mr.Peak),
					fmtMetricValue(mr.ID, *mr.Avg)})
			}
		}
		if len(rows) > 0 {
			cwTbl := table.NewModel(m.ctx, []table.Column{
				{Title: "CloudWatch Peaks", Width: 26},
				{Title: "Peak", Width: 12, Align: lipgloss.Right},
				{Title: "Avg", Width: 12, Align: lipgloss.Right},
			})
			cwTbl.SetRows(rows)
			sections = append(sections, "", cwTbl.View())
		}
	}

	// Verdict
	if m.computedVerdict == nil {
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}
	v := m.computedVerdict

	var verdictStyle lipgloss.Style
	switch v.Level {
	case verdict.Fail:
		verdictStyle = s.Verdict.Fail
	case verdict.Warn:
		verdictStyle = s.Verdict.Warn
	default:
		verdictStyle = s.Verdict.Pass
	}
	sections = append(sections, "", "  "+s.Common.BoldStyle.Render("Verdict: ")+verdictStyle.Render(v.Level.String()))
	for _, reason := range v.Reasons {
		icon := "\u2713"
		switch v.Level {
		case verdict.Fail:
			icon = "\u2717"
		case verdict.Warn:
			icon = "\u26a0"
		}
		sections = append(sections, fmt.Sprintf("  %s %s", icon, reason))
	}

	// Output files
	sections = append(sections, "", "  Output Files")
	k6SummaryPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+"-summary.json")
	if _, err := os.Stat(k6SummaryPath); err == nil {
		sections = append(sections, fmt.Sprintf("  %-18s %s", "k6 summary:", k6SummaryPath))
	}
	htmlPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+".html")
	if _, err := os.Stat(htmlPath); err == nil {
		sections = append(sections, fmt.Sprintf("  %-18s %s", "HTML report:", htmlPath))
	}
	if m.reportPath != "" {
		sections = append(sections, fmt.Sprintf("  %-18s %s", "Unified report:", m.reportPath))
	}

	sections = append(sections, "")
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- Report Dashboard ---

func (m *Model) initDashboard() {
	w := m.ctx.ContentWidth
	topH, bottomH := constants.CalcPanelHeights(m.ctx.ContentHeight, 55)

	halfW := w / 2

	// Top row: k6 (left) + graphs (right)
	m.k6Panel = panel.NewModel(m.ctx, "k6 Summary [1]", halfW, topH)
	m.k6Panel.SetContent(m.renderK6SummaryGrid())

	m.graphsPanel = panel.NewModel(m.ctx, "Graphs [2]", w-halfW, topH)
	m.reportRPSChart = timechart.NewModel(m.ctx, "Throughput", "req/s", w-halfW-4, topH/2-2)
	m.reportLatencyChart = timechart.NewModel(m.ctx, "Latency", "ms", w-halfW-4, topH/2-2)
	m.populateReportCharts()
	m.graphsPanel.SetContent(m.renderGraphsPanelContent())

	// Bottom row: infra (left) + events (right)
	infraW := w * 55 / 100
	eventsW := w - infraW
	m.infraPanel = panel.NewModel(m.ctx, "Infrastructure [3]", infraW, bottomH)
	m.infraPanel.SetContent(m.renderInfraTable())

	m.eventsPanel = panel.NewModel(m.ctx, "Scaling Events [4]", eventsW, bottomH)
	m.eventsPanel.SetContent(m.renderEventsList())

	m.focusMgr = focus.New(4)
	m.k6Panel.SetFocused(true)

	m.footerComp.SetHints([]footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "tab", Action: "panel"},
		{Key: "1-4", Action: "jump"},
		{Key: "+", Action: "expand"},
		{Key: "↑↓", Action: "scroll"},
		{Key: "e", Action: "export"},
		{Key: "o", Action: "open"},
		{Key: "r", Action: "raw"},
		{Key: "?", Action: "help"},
	})
}

func (m *Model) resizeDashboardPanels() {
	w := m.ctx.ContentWidth
	topH, bottomH := constants.CalcPanelHeights(m.ctx.ContentHeight, 55)

	if w >= constants.BreakpointSplit {
		halfW := w / 2
		m.k6Panel.SetDimensions(halfW, topH)
		m.graphsPanel.SetDimensions(w-halfW, topH)

		infraW := w * 55 / 100
		eventsW := w - infraW
		m.infraPanel.SetDimensions(infraW, bottomH)
		m.eventsPanel.SetDimensions(eventsW, bottomH)

		m.reportRPSChart.Resize(w-halfW-4, topH/2-2)
		m.reportLatencyChart.Resize(w-halfW-4, topH/2-2)
		m.graphsPanel.SetContent(m.renderGraphsPanelContent())
	} else {
		m.k6Panel.SetDimensions(w, topH)
		m.graphsPanel.SetDimensions(w, topH)
		m.infraPanel.SetDimensions(w, bottomH)
		m.eventsPanel.SetDimensions(w, bottomH)

		m.reportRPSChart.Resize(w-4, topH/2-2)
		m.reportLatencyChart.Resize(w-4, topH/2-2)
		m.graphsPanel.SetContent(m.renderGraphsPanelContent())
	}
}

func (m Model) viewReportDashboard() string {
	if m.rawMode {
		return m.renderReport()
	}

	width := m.ctx.ContentWidth
	verdict := m.renderVerdictBar()

	focused := m.focusMgr.Current()
	panels := [4]panel.Model{m.k6Panel, m.graphsPanel, m.infraPanel, m.eventsPanel}

	// Full expand: only the focused panel renders
	if panels[focused].ExpandMode() == constants.ExpandFull {
		return lipgloss.JoinVertical(lipgloss.Left, panels[focused].View(), verdict)
	}

	switch {
	case width >= constants.BreakpointSplit:
		topRow := lipgloss.JoinHorizontal(lipgloss.Top,
			m.k6Panel.View(),
			m.graphsPanel.View(),
		)
		bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
			m.infraPanel.View(),
			m.eventsPanel.View(),
		)
		return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow, verdict)
	case width >= constants.BreakpointStacked:
		return lipgloss.JoinVertical(lipgloss.Left,
			m.k6Panel.View(),
			m.graphsPanel.View(),
			m.infraPanel.View(),
			m.eventsPanel.View(),
			verdict,
		)
	default:
		return m.renderReport()
	}
}

func (m *Model) updateDashboardFocus() tea.Cmd {
	m.k6Panel.SetFocused(m.focusMgr.IsFocused(0))
	m.graphsPanel.SetFocused(m.focusMgr.IsFocused(1))
	m.infraPanel.SetFocused(m.focusMgr.IsFocused(2))
	m.eventsPanel.SetFocused(m.focusMgr.IsFocused(3))
	return tea.Batch(
		m.k6Panel.TransitionCmd(),
		m.graphsPanel.TransitionCmd(),
		m.infraPanel.TransitionCmd(),
		m.eventsPanel.TransitionCmd(),
	)
}

func (m *Model) cycleExpandFocusedPanel() {
	panels := []*panel.Model{&m.k6Panel, &m.graphsPanel, &m.infraPanel, &m.eventsPanel}
	idx := m.focusMgr.Current()
	if idx >= 0 && idx < len(panels) {
		panels[idx].CycleExpand()
	}
}

func (m Model) anyPanelExpanded() bool {
	return m.k6Panel.ExpandMode() != constants.ExpandNormal ||
		m.graphsPanel.ExpandMode() != constants.ExpandNormal ||
		m.infraPanel.ExpandMode() != constants.ExpandNormal ||
		m.eventsPanel.ExpandMode() != constants.ExpandNormal
}

func (m *Model) resetAllPanelExpand() {
	m.k6Panel.ResetExpand()
	m.graphsPanel.ResetExpand()
	m.infraPanel.ResetExpand()
	m.eventsPanel.ResetExpand()
}

func (m *Model) scrollFocusedPanel(dir int) {
	var p *panel.Model
	switch m.focusMgr.Current() {
	case 0:
		p = &m.k6Panel
	case 1:
		p = &m.graphsPanel
	case 2:
		p = &m.infraPanel
	case 3:
		p = &m.eventsPanel
	}
	if p == nil {
		return
	}
	if dir > 0 {
		p.ScrollDown()
	} else {
		p.ScrollUp()
	}
}

func (m Model) renderK6SummaryGrid() string {
	if m.report == nil || m.report.K6 == nil {
		if m.k6Result != nil && m.k6Result.ExitCode != 0 {
			return m.ctx.Styles.Common.FaintTextStyle.Render(
				fmt.Sprintf("k6 exited with code %d \u2014 no summary data available", m.k6Result.ExitCode))
		}
		return m.ctx.Styles.Common.FaintTextStyle.Render("No k6 data available")
	}
	k := m.report.K6
	s := m.ctx.Styles

	type pair struct{ left, right string }
	rows := []pair{
		{
			s.Table.Label.Render("p95 latency") + "  " + fmtFloatMs(k.P95ms),
			s.Table.Label.Render("Throughput") + "  " + fmtFloatRate(k.RPSAvg),
		},
		{
			s.Table.Label.Render("p90 latency") + "  " + fmtFloatMs(k.P90ms),
			s.Table.Label.Render("Error rate") + "  " + fmtPct(k.ErrorRate),
		},
		{
			s.Table.Label.Render("Checks") + "  " + fmtPctRate(k.ChecksRate),
			s.Table.Label.Render("VUs max") + "  " + fmtIntPtr(k.VUsMax),
		},
		{
			s.Table.Label.Render("Total reqs") + "  " + fmtIntPtr(k.TotalRequests),
			s.Table.Label.Render("Thresholds") + "  " + fmt.Sprintf("%d passed, %d failed", k.Thresholds.Passed, k.Thresholds.Failed),
		},
	}

	var b strings.Builder
	panelInnerW := m.ctx.ContentWidth - 2
	if m.ctx.ContentWidth >= constants.BreakpointSplit {
		panelInnerW = m.ctx.ContentWidth/2 - 2
	}
	colWidth := panelInnerW / 2
	for _, r := range rows {
		left := lipgloss.NewStyle().Width(colWidth).Render(r.left)
		right := lipgloss.NewStyle().Width(colWidth).Render(r.right)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderInfraTable() string {
	s := m.ctx.Styles

	if len(m.metrics) == 0 && m.preSnapshot.TaskRunning == 0 && m.postSnapshot.TaskRunning == 0 {
		return s.Common.FaintTextStyle.Render("Infrastructure metrics pending")
	}

	var lines []string

	// Snapshot deltas
	lines = append(lines,
		fmt.Sprintf("%s  %d -> %d  %s",
			s.Table.Label.Render("ECS tasks"),
			m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning,
			fmtDelta(m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning)),
		fmt.Sprintf("%s  %d -> %d  %s",
			s.Table.Label.Render("EC2 instances"),
			m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances,
			fmtDelta(m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances)),
		"",
	)

	// CloudWatch peaks
	for _, mr := range m.metrics {
		label := metricLabel(mr.ID)
		if label == "" {
			continue
		}
		if mr.Peak != nil && mr.Avg != nil {
			lines = append(lines, fmt.Sprintf("%s  peak=%s  avg=%s",
				s.Table.Label.Render(label),
				fmtMetricValue(mr.ID, *mr.Peak),
				fmtMetricValue(mr.ID, *mr.Avg)))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderEventsList() string {
	var lines []string

	if len(m.activities.ServiceScaling) > 0 {
		for _, ev := range m.activities.ServiceScaling {
			lines = append(lines, fmt.Sprintf("[%s] %s", ev.Time, ev.Description))
		}
	}
	if len(m.activities.NodeScaling) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		for _, ev := range m.activities.NodeScaling {
			lines = append(lines, fmt.Sprintf("[%s] %s %s", ev.Time, ev.Status, ev.Cause))
		}
	}
	if len(m.activities.Alarms) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		for _, a := range m.activities.Alarms {
			lines = append(lines, fmt.Sprintf("[%s] %s: %s -> %s", a.Time, a.AlarmName, a.OldState, a.NewState))
		}
	}

	if len(lines) == 0 {
		return "No scaling events recorded"
	}
	return strings.Join(lines, "\n")
}

func (m *Model) populateReportCharts() {
	for _, mr := range m.metrics {
		switch mr.ID {
		case "alb_requests_per_target":
			if len(mr.Timestamps) > 0 && len(mr.Values) > 0 {
				m.reportRPSChart.SetData(mr.Timestamps, mr.Values)
				m.hasRPSData = true
			}
		case "alb_response_time":
			if len(mr.Timestamps) > 0 && len(mr.Values) > 0 {
				m.reportLatencyChart.SetData(mr.Timestamps, mr.Values)
				m.hasLatencyData = true
			}
		}
	}
}

func (m Model) renderGraphsPanelContent() string {
	if !m.hasRPSData && !m.hasLatencyData {
		return m.ctx.Styles.Common.FaintTextStyle.Render("No metric data for graphs")
	}

	var sections []string
	if m.hasRPSData {
		sections = append(sections, m.reportRPSChart.View())
	}
	if m.hasLatencyData {
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections = append(sections, m.reportLatencyChart.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderVerdictBar() string {
	s := m.ctx.Styles

	if m.computedVerdict == nil {
		return ""
	}
	v := m.computedVerdict

	var verdictStyle lipgloss.Style
	switch v.Level {
	case verdict.Fail:
		verdictStyle = s.Verdict.Fail
	case verdict.Warn:
		verdictStyle = s.Verdict.Warn
	default:
		verdictStyle = s.Verdict.Pass
	}

	var b strings.Builder
	b.WriteString("  " + s.Common.BoldStyle.Render("Verdict: ") + verdictStyle.Render(v.Level.String()))
	for _, reason := range v.Reasons {
		icon := "\u2713"
		switch v.Level {
		case verdict.Fail:
			icon = "\u2717"
		case verdict.Warn:
			icon = "\u26a0"
		}
		b.WriteString("\n  " + icon + " " + reason)
	}
	return b.String()
}
