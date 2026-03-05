package runtui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/tui/constants"
)

func (m *Model) handleK6Point(point k6.K6Point) {
	pointTime, err := time.Parse(time.RFC3339, point.Time)
	if err != nil {
		return
	}

	switch point.Metric {
	case "http_req_duration":
		m.latencyChart.Push(pointTime, point.Value)
	case "http_reqs":
		second := pointTime.Truncate(time.Second)
		if second != m.liveRPSTime {
			if !m.liveRPSTime.IsZero() {
				m.rpsChart.Push(m.liveRPSTime, float64(m.liveRPSCount))
			}
			m.liveRPSCount = 0
			m.liveRPSTime = second
		}
		m.liveRPSCount++
	}
}

func (m *Model) updateGaugesFromMetrics(metrics []provider.MetricResult) {
	for _, mr := range metrics {
		if len(mr.Values) == 0 {
			continue
		}
		latest := mr.Values[len(mr.Values)-1]
		switch mr.ID {
		case "service_cpu":
			m.cpuGauge.SetValue(latest, 100.0)
			m.cpuTrend.Push(latest)
		case "service_memory":
			m.memGauge.SetValue(latest, 100.0)
			m.memTrend.Push(latest)
		case "capacity_provider_reservation":
			m.reservGauge.SetValue(latest, 100.0)
			m.reservTrend.Push(latest)
		}
	}
}

func (m *Model) updateLivePanelContent() {
	m.liveGraphPanel.SetContent(m.viewLiveGraphs())
	m.liveInfraPanel.SetContent(m.renderInfraLivePanel())
}

func (m *Model) resizeLivePanels() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight - 8
	switch {
	case w >= constants.BreakpointSplit:
		leftW := w * 55 / 100
		m.liveGraphPanel.SetDimensions(leftW, panelH)
		m.liveInfraPanel.SetDimensions(w-leftW, panelH)
	case w >= constants.BreakpointStacked:
		m.liveGraphPanel.SetDimensions(w, m.ctx.ContentHeight*2/3)
		m.liveInfraPanel.SetDimensions(w, m.ctx.ContentHeight/3)
	}
}

func (m Model) viewLiveDashboard() string {
	width := m.ctx.ContentWidth
	switch {
	case width >= constants.BreakpointSplit:
		return m.viewLiveSplit()
	case width >= constants.BreakpointStacked:
		return m.viewLiveStacked()
	default:
		return m.viewLiveFallback()
	}
}

func (m Model) viewLiveHeader() ([]string, string) {
	var sections []string
	sections = append(sections, m.headerComp.View(), "")

	elapsed := time.Since(m.startTime)
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	elapsedStr := m.ctx.Styles.Common.FaintTextStyle.Render(
		fmt.Sprintf("  Elapsed: %dm %ds", mins, secs))

	return sections, elapsedStr
}

func (m Model) viewLiveGraphs() string {
	if m.graphMode {
		return lipgloss.JoinVertical(lipgloss.Left,
			m.rpsChart.View(),
			"",
			m.latencyChart.View(),
		)
	}
	return m.stepper.View()
}

// viewLiveSplit renders side-by-side layout for wide terminals (>=120).
func (m Model) viewLiveSplit() string {
	sections, elapsedStr := m.viewLiveHeader()
	sections = append(sections, elapsedStr, "")

	middle := lipgloss.JoinHorizontal(lipgloss.Top,
		m.liveGraphPanel.View(),
		m.liveInfraPanel.View(),
	)

	sections = append(sections, middle)
	sections = append(sections, "", m.renderHealthBar(), "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewLiveStacked renders vertical stack for medium terminals (>=80).
func (m Model) viewLiveStacked() string {
	sections, elapsedStr := m.viewLiveHeader()
	sections = append(sections, elapsedStr, "")

	sections = append(sections, m.liveGraphPanel.View())
	sections = append(sections, m.liveInfraPanel.View())
	sections = append(sections, "", m.renderHealthBar(), "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewLiveFallback renders minimal view for narrow terminals (<80).
func (m Model) viewLiveFallback() string {
	sections, elapsedStr := m.viewLiveHeader()
	sections = append(sections, elapsedStr, "")

	sections = append(sections, m.stepper.View())
	sections = append(sections, "", m.renderHealthBar(), "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderInfraLivePanel() string {
	s := m.ctx.Styles
	var lines []string

	lines = append(lines, s.Header.Root.Render("─ Infrastructure "))
	if m.infraError != nil {
		lines = append(lines, s.Common.WarnStyle.Render("  Warning: "+m.infraError.Error()))
	}
	lines = append(lines, "")
	lines = append(lines, m.cpuGauge.View())
	lines = append(lines, m.cpuTrend.View())
	lines = append(lines, m.memGauge.View())
	lines = append(lines, m.memTrend.View())
	lines = append(lines, m.reservGauge.View())
	lines = append(lines, m.reservTrend.View())
	lines = append(lines, "")

	lines = append(lines, s.Common.BoldStyle.Render("Tasks"))
	lines = append(lines, fmt.Sprintf("  Running: %d  Desired: %d",
		m.liveSnapshot.TaskRunning, m.liveSnapshot.TaskDesired))
	lines = append(lines, "")
	lines = append(lines, s.Common.BoldStyle.Render("ASG"))
	lines = append(lines, fmt.Sprintf("  Instances: %d  Desired: %d",
		m.liveSnapshot.ASGInstances, m.liveSnapshot.ASGDesired))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderHealthBar() string {
	s := m.ctx.Styles
	var checks []string

	// CPU check
	cpuOK := true
	for _, mr := range m.liveMetrics {
		if mr.ID == "service_cpu" && mr.Peak != nil && *mr.Peak >= 90 {
			cpuOK = false
		}
	}
	if cpuOK {
		checks = append(checks, s.Verdict.Pass.Render("✓ CPU < 90%"))
	} else {
		checks = append(checks, s.Verdict.Warn.Render("⚠ CPU ≥ 90%"))
	}

	// Task stability check
	if m.liveSnapshot.TaskRunning >= m.preSnapshot.TaskRunning {
		checks = append(checks, s.Verdict.Pass.Render("✓ Tasks stable"))
	} else {
		checks = append(checks, s.Verdict.Warn.Render("⚠ Tasks decreased"))
	}

	// 5xx check
	has5xx := false
	for _, mr := range m.liveMetrics {
		if mr.ID == "alb_5xx" && mr.Peak != nil && *mr.Peak > 0 {
			has5xx = true
		}
	}
	if !has5xx {
		checks = append(checks, s.Verdict.Pass.Render("✓ Zero 5xx"))
	} else {
		checks = append(checks, s.Verdict.Warn.Render("⚠ 5xx detected"))
	}

	return "  " + strings.Join(checks, "  ")
}
