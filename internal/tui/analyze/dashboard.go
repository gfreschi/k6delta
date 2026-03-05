package analyzetui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/constants"
)

func (m *Model) initDashboard() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	m.statePanel = panel.NewModel(m.ctx, "State [1]", w, panelH)
	m.statePanel.SetContent(m.renderStateContent())

	metricsW := w * 55 / 100
	eventsW := w - metricsW

	m.metricsPanel = panel.NewModel(m.ctx, "Metrics [2]", metricsW, panelH)
	m.metricsPanel.SetContent(m.renderMetricsContent())

	m.eventsPanel = panel.NewModel(m.ctx, "Activities [3]", eventsW, panelH)
	m.eventsPanel.SetContent(m.renderActivitiesContent())

	m.focusMgr = focus.New(3)
	m.statePanel.SetFocused(true)

	m.footerComp.SetHints([]footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "tab", Action: "next panel"},
		{Key: "\u2191\u2193", Action: "scroll"},
		{Key: "e", Action: "export JSON"},
	})
}

func (m *Model) resizeDashboardPanels() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2
	m.statePanel.SetDimensions(w, panelH)

	if w >= constants.BreakpointSplit {
		metricsW := w * 55 / 100
		eventsW := w - metricsW
		m.metricsPanel.SetDimensions(metricsW, panelH)
		m.eventsPanel.SetDimensions(eventsW, panelH)
	} else {
		m.metricsPanel.SetDimensions(w, panelH)
		m.eventsPanel.SetDimensions(w, panelH)
	}
}

func (m Model) viewDashboard() string {
	width := m.ctx.ContentWidth
	stateView := m.statePanel.View()

	switch {
	case width >= constants.BreakpointSplit:
		middle := lipgloss.JoinHorizontal(lipgloss.Top,
			m.metricsPanel.View(),
			m.eventsPanel.View(),
		)
		return lipgloss.JoinVertical(lipgloss.Left, stateView, middle)
	case width >= constants.BreakpointStacked:
		return lipgloss.JoinVertical(lipgloss.Left,
			stateView, m.metricsPanel.View(), m.eventsPanel.View())
	default:
		return m.renderRawDisplay()
	}
}

func (m *Model) updateDashboardFocus() tea.Cmd {
	m.statePanel.SetFocused(m.focusMgr.IsFocused(0))
	m.metricsPanel.SetFocused(m.focusMgr.IsFocused(1))
	m.eventsPanel.SetFocused(m.focusMgr.IsFocused(2))
	return tea.Batch(
		m.statePanel.TransitionCmd(),
		m.metricsPanel.TransitionCmd(),
		m.eventsPanel.TransitionCmd(),
	)
}

func (m *Model) scrollFocusedPanel(dir int) {
	var p *panel.Model
	switch m.focusMgr.Current() {
	case 0:
		p = &m.statePanel
	case 1:
		p = &m.metricsPanel
	case 2:
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

func (m Model) renderStateContent() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("ECS Tasks:  running=%d  desired=%d",
		m.snapshot.TaskRunning, m.snapshot.TaskDesired))
	if m.snapshot.ASGName != "" {
		lines = append(lines, fmt.Sprintf("ASG:        in_service=%d  desired=%d",
			m.snapshot.ASGInstances, m.snapshot.ASGDesired))
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderMetricsContent() string {
	var metricRows []table.Row
	for _, mr := range m.metrics {
		if mr.Peak == nil || mr.Avg == nil {
			metricRows = append(metricRows, table.Row{mr.ID, "-", "-", fmt.Sprintf("%d", len(mr.Values))})
		} else {
			metricRows = append(metricRows, table.Row{mr.ID,
				fmt.Sprintf("%.2f", *mr.Peak),
				fmt.Sprintf("%.2f", *mr.Avg),
				fmt.Sprintf("%d", len(mr.Values))})
		}
	}
	metricTbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Metric", Width: 35},
		{Title: "Peak", Width: 10, Align: lipgloss.Right},
		{Title: "Avg", Width: 10, Align: lipgloss.Right},
		{Title: "Points", Width: 8, Align: lipgloss.Right},
	})
	metricTbl.SetRows(metricRows)
	return metricTbl.View()
}

func (m Model) renderActivitiesContent() string {
	var lines []string

	if len(m.activities.ServiceScaling) > 0 {
		lines = append(lines, "ECS Task Scaling:")
		for _, a := range m.activities.ServiceScaling {
			desc := a.Description
			if desc == "" {
				desc = a.Cause
			}
			lines = append(lines, fmt.Sprintf("  %s  %s  %s", a.Time, a.Status, desc))
		}
	}
	if len(m.activities.NodeScaling) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "ASG Scaling:")
		for _, a := range m.activities.NodeScaling {
			desc := a.Description
			if desc == "" {
				desc = a.Cause
			}
			lines = append(lines, fmt.Sprintf("  %s  %s  %s", a.Time, a.Status, desc))
		}
	}
	if len(m.activities.Alarms) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "Alarm History:")
		for _, a := range m.activities.Alarms {
			lines = append(lines, fmt.Sprintf("  %s  %s  %s -> %s", a.Time, a.AlarmName, a.OldState, a.NewState))
		}
	}

	if len(lines) == 0 {
		return "No scaling activities during test window"
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderRawDisplay() string {
	s := m.ctx.Styles
	var sections []string

	sections = append(sections, s.Header.Root.Render("Current State"), "")
	sections = append(sections, fmt.Sprintf("  ECS Tasks:  running=%d  desired=%d",
		m.snapshot.TaskRunning, m.snapshot.TaskDesired))
	if m.snapshot.ASGName != "" {
		sections = append(sections, fmt.Sprintf("  ASG:        in_service=%d  desired=%d",
			m.snapshot.ASGInstances, m.snapshot.ASGDesired))
	}
	sections = append(sections, "")

	sections = append(sections, s.Header.Root.Render("Metrics"))
	sections = append(sections, m.renderMetricsContent(), "")

	sections = append(sections, s.Header.Root.Render("Scaling Activities"), "")
	sections = append(sections, m.renderActivitiesContent())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- JSON output ---

func (m Model) writeOutputFile() error {
	data := m.buildJSONOutput()
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	return os.WriteFile(m.outputFile, encoded, 0o644)
}

type jsonOutput struct {
	Metadata          jsonMetadata          `json:"metadata"`
	Metrics           map[string]jsonMetric `json:"metrics"`
	ScalingActivities interface{}           `json:"scaling_activities"`
	AlarmHistory      interface{}           `json:"alarm_history"`
}

type jsonMetadata struct {
	App         string         `json:"app"`
	Environment string         `json:"environment"`
	Cluster     string         `json:"cluster"`
	Service     string         `json:"service"`
	TimeWindow  jsonTimeWindow `json:"time_window"`
	GeneratedAt string         `json:"generated_at"`
}

type jsonTimeWindow struct {
	Start         string `json:"start"`
	End           string `json:"end"`
	PeriodSeconds int32  `json:"period_seconds"`
}

type jsonMetric struct {
	Values     []float64 `json:"values"`
	Timestamps []string  `json:"timestamps"`
	DataPoints int       `json:"data_points"`
	Peak       *float64  `json:"peak"`
	Avg        *float64  `json:"avg"`
}

func (m Model) buildJSONOutput() jsonOutput {
	metricsMap := make(map[string]jsonMetric, len(m.metrics))
	for _, mr := range m.metrics {
		timestamps := make([]string, len(mr.Timestamps))
		for i, t := range mr.Timestamps {
			timestamps[i] = t.Format(time.RFC3339)
		}
		metricsMap[mr.ID] = jsonMetric{
			Values:     mr.Values,
			Timestamps: timestamps,
			DataPoints: len(mr.Values),
			Peak:       mr.Peak,
			Avg:        mr.Avg,
		}
	}

	return jsonOutput{
		Metadata: jsonMetadata{
			App:         m.app.Name,
			Environment: m.app.Env,
			Cluster:     m.app.Cluster,
			Service:     m.app.Service,
			TimeWindow: jsonTimeWindow{
				Start:         m.startTime,
				End:           m.endTime,
				PeriodSeconds: m.period,
			},
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		},
		Metrics:           metricsMap,
		ScalingActivities: m.activities,
		AlarmHistory:      m.activities.Alarms,
	}
}
