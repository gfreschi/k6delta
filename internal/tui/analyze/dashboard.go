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
	topH, bottomH := calcPanelHeights(m.ctx.ContentHeight, 30, 70)

	m.statePanel = panel.NewModel(m.ctx, "State [1]", w, topH)
	m.statePanel.SetContent(m.renderStateContent())

	metricsW := w * 55 / 100
	eventsW := w - metricsW

	m.metricsPanel = panel.NewModel(m.ctx, "Metrics [2]", metricsW, bottomH)
	m.metricsPanel.SetContent(m.renderMetricsContent())

	m.eventsPanel = panel.NewModel(m.ctx, "Activities [3]", eventsW, bottomH)
	m.eventsPanel.SetContent(m.renderActivitiesContent())

	m.focusMgr = focus.New(3)
	m.statePanel.SetFocused(true)

	m.footerComp.SetHints([]footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "tab", Action: "panel"},
		{Key: "1-3", Action: "jump"},
		{Key: "+", Action: "expand"},
		{Key: "↑↓", Action: "scroll"},
		{Key: "e", Action: "export"},
		{Key: "?", Action: "help"},
	})
}

func (m *Model) resizeDashboardPanels() {
	w := m.ctx.ContentWidth
	topH, bottomH := calcPanelHeights(m.ctx.ContentHeight, 30, 70)
	m.statePanel.SetDimensions(w, topH)

	if w >= constants.BreakpointSplit {
		metricsW := w * 55 / 100
		eventsW := w - metricsW
		m.metricsPanel.SetDimensions(metricsW, bottomH)
		m.eventsPanel.SetDimensions(eventsW, bottomH)
	} else {
		m.metricsPanel.SetDimensions(w, bottomH)
		m.eventsPanel.SetDimensions(w, bottomH)
	}
}

// calcPanelHeights splits total height into two portions by percentage.
func calcPanelHeights(totalHeight, topPct, _ int) (int, int) {
	topH := max(totalHeight*topPct/100, 4)
	bottomH := max(totalHeight-topH, 4)
	return topH, bottomH
}

func (m Model) viewDashboard() string {
	width := m.ctx.ContentWidth
	focused := m.focusMgr.Current()
	panels := [3]panel.Model{m.statePanel, m.metricsPanel, m.eventsPanel}

	// Full expand: only the focused panel renders
	if panels[focused].ExpandMode() == constants.ExpandFull {
		return panels[focused].View()
	}

	switch {
	case width >= constants.BreakpointSplit:
		middle := lipgloss.JoinHorizontal(lipgloss.Top,
			m.metricsPanel.View(),
			m.eventsPanel.View(),
		)
		return lipgloss.JoinVertical(lipgloss.Left, m.statePanel.View(), middle)
	case width >= constants.BreakpointStacked:
		return lipgloss.JoinVertical(lipgloss.Left,
			m.statePanel.View(), m.metricsPanel.View(), m.eventsPanel.View())
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

func (m *Model) cycleExpandFocusedPanel() {
	panels := []*panel.Model{&m.statePanel, &m.metricsPanel, &m.eventsPanel}
	idx := m.focusMgr.Current()
	if idx >= 0 && idx < len(panels) {
		panels[idx].CycleExpand()
	}
}

func (m Model) anyPanelExpanded() bool {
	return m.statePanel.ExpandMode() != constants.ExpandNormal ||
		m.metricsPanel.ExpandMode() != constants.ExpandNormal ||
		m.eventsPanel.ExpandMode() != constants.ExpandNormal
}

func (m *Model) resetAllPanelExpand() {
	m.statePanel.ResetExpand()
	m.metricsPanel.ResetExpand()
	m.eventsPanel.ResetExpand()
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

func (m Model) renderHelpOverlay() string {
	s := m.ctx.Styles
	w := m.ctx.ContentWidth
	h := m.ctx.ContentHeight

	groups := []struct {
		title string
		keys  [][2]string
	}{
		{"Navigation", [][2]string{
			{"q", "Quit"},
			{"?", "Toggle help"},
			{"esc", "Close help / collapse panel"},
		}},
		{"Panels", [][2]string{
			{"tab / shift+tab", "Next / previous panel"},
			{"1-3", "Jump to panel"},
			{"+", "Cycle expand (normal → expanded → full)"},
			{"↑↓ / j k", "Scroll focused panel"},
		}},
		{"Actions", [][2]string{
			{"e", "Export JSON report"},
		}},
	}

	var lines []string
	lines = append(lines, s.Header.Root.Render("Keyboard Shortcuts"), "")
	for _, g := range groups {
		lines = append(lines, s.Common.BoldStyle.Render("  "+g.title))
		for _, kv := range g.keys {
			lines = append(lines, fmt.Sprintf("    %-22s %s", s.Footer.Key.Render(kv[0]), kv[1]))
		}
		lines = append(lines, "")
	}
	lines = append(lines, s.Common.FaintTextStyle.Render("  Press ? or esc to close"))

	content := strings.Join(lines, "\n")
	overlay := lipgloss.NewStyle().
		Width(min(w-4, 60)).
		Height(min(h-2, len(lines)+2)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Panel.Focused.GetBorderTopForeground()).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, overlay)
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
