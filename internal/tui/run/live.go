package runtui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/tui/components/metriccard"
	"github.com/gfreschi/k6delta/internal/tui/components/statusbar"
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

func (m *Model) updateTilesFromMetrics(metrics []provider.MetricResult) {
	for _, mr := range metrics {
		if len(mr.Values) == 0 {
			continue
		}
		latest := mr.Values[len(mr.Values)-1]
		switch mr.ID {
		case "service_cpu":
			m.cpuTile.SetValue(latest, 100.0)
			m.cpuTile.PushSparkline(latest)
		case "service_memory":
			m.memTile.SetValue(latest, 100.0)
			m.memTile.PushSparkline(latest)
		case "capacity_provider_reservation":
			m.reservTile.SetValue(latest, 100.0)
			m.reservTile.PushSparkline(latest)
		}
	}
}

func (m *Model) updateTilesFromSnapshot(snap provider.Snapshot) {
	m.tasksTile.SetValue(float64(snap.TaskRunning), float64(max(snap.TaskDesired, 1)))
	if m.preSnapshot.TaskRunning > 0 {
		m.tasksTile.SetDelta(fmt.Sprintf("%d\u2192%d", m.preSnapshot.TaskRunning, snap.TaskRunning))
		if snap.TaskRunning < m.preSnapshot.TaskRunning {
			m.tasksTile.SetSeverity(metriccard.SeverityWarn)
		}
	}
	m.asgTile.SetValue(float64(snap.ASGInstances), float64(max(snap.ASGDesired, 1)))
	if m.preSnapshot.ASGInstances > 0 {
		m.asgTile.SetDelta(fmt.Sprintf("%d\u2192%d", m.preSnapshot.ASGInstances, snap.ASGInstances))
	}
}

func (m *Model) updateStatusBar() {
	if !m.liveMode {
		return
	}
	var items []statusbar.Item

	elapsed := time.Since(m.startTime)
	mins := int(elapsed.Minutes())
	secs := int(elapsed.Seconds()) % 60
	items = append(items, statusbar.Item{Label: "elapsed", Value: fmt.Sprintf("%dm %ds", mins, secs)})

	if m.liveRPSCount > 0 {
		items = append(items, statusbar.Item{Label: "RPS", Value: fmt.Sprintf("%d", m.liveRPSCount)})
	}

	for _, mr := range m.liveMetrics {
		if mr.ID == "service_cpu" && len(mr.Values) > 0 {
			latest := mr.Values[len(mr.Values)-1]
			items = append(items, statusbar.Item{Label: "CPU", Value: fmt.Sprintf("%.0f%%", latest)})
			break
		}
	}

	m.statusBar.SetItems(items)
}

func (m *Model) updateLivePanelContent() {
	m.liveGraphPanel.SetContent(m.viewLiveGraphs())
	m.liveInfraPanel.SetContent(m.renderLiveInfraTiles())
	m.updateStatusBar()
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
	if sb := m.statusBar.View(); sb != "" {
		sections = append(sections, "", sb)
	}
	sections = append(sections, "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewLiveStacked renders vertical stack for medium terminals (>=80).
func (m Model) viewLiveStacked() string {
	sections, elapsedStr := m.viewLiveHeader()
	sections = append(sections, elapsedStr, "")

	sections = append(sections, m.liveGraphPanel.View())
	sections = append(sections, m.liveInfraPanel.View())
	if sb := m.statusBar.View(); sb != "" {
		sections = append(sections, "", sb)
	}
	sections = append(sections, "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// viewLiveFallback renders minimal view for narrow terminals (<80).
func (m Model) viewLiveFallback() string {
	sections, elapsedStr := m.viewLiveHeader()
	sections = append(sections, elapsedStr, "")

	sections = append(sections, m.stepper.View())
	if sb := m.statusBar.View(); sb != "" {
		sections = append(sections, "", sb)
	}
	sections = append(sections, "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderLiveInfraTiles() string {
	s := m.ctx.Styles
	var lines []string

	lines = append(lines, s.Header.Root.Render("─ Infrastructure "))
	if m.infraError != nil {
		lines = append(lines, s.Common.WarnStyle.Render("  Warning: "+m.infraError.Error()))
	}
	lines = append(lines, "")

	// Top row: CPU, Memory, Reservation
	topRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.cpuTile.View(), m.memTile.View(), m.reservTile.View())
	lines = append(lines, topRow)

	// Bottom row: Tasks, ASG
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
		m.tasksTile.View(), m.asgTile.View())
	lines = append(lines, "", bottomRow)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

