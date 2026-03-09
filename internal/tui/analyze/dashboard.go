package analyzetui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/metriccard"
	"github.com/gfreschi/k6delta/internal/tui/components/overlay"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/timeline"
	"github.com/gfreschi/k6delta/internal/tui/constants"
)

func (m *Model) initDashboard() {
	w := m.ctx.ContentWidth
	topH, bottomH := constants.CalcPanelHeights(m.ctx.ContentHeight, 30)

	m.statePanel = panel.NewModel(m.ctx, "State [1]", w, topH)
	m.statePanel.SetContent(m.renderStateContent())

	metricsW := w * constants.PanelSplitPct / 100
	eventsW := w - metricsW

	m.metricsPanel = panel.NewModel(m.ctx, "Metrics [2]", metricsW, bottomH)
	m.metricsPanel.SetContent(m.renderMetricsContent())

	m.eventsPanel = panel.NewModel(m.ctx, "Activities [3]", eventsW, bottomH)
	m.eventsPanel.SetContent(m.renderActivitiesTimeline())

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
	topH, bottomH := constants.CalcPanelHeights(m.ctx.ContentHeight, 30)
	m.statePanel.SetDimensions(w, topH)

	if w >= constants.BreakpointSplit {
		metricsW := w * constants.PanelSplitPct / 100
		eventsW := w - metricsW
		m.metricsPanel.SetDimensions(metricsW, bottomH)
		m.eventsPanel.SetDimensions(eventsW, bottomH)
	} else {
		m.metricsPanel.SetDimensions(w, bottomH)
		m.eventsPanel.SetDimensions(w, bottomH)
	}
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

func (m *Model) expandFocusedPanelFull() {
	panels := []*panel.Model{&m.statePanel, &m.metricsPanel, &m.eventsPanel}
	idx := m.focusMgr.Current()
	if idx >= 0 && idx < len(panels) {
		panels[idx].SetExpandFull()
	}
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
	tileW := constants.TileWidthNormal
	var tiles []string

	// Tasks tile
	taskTile := metriccard.NewModel(m.ctx, "tasks", "count", tileW)
	taskTile.SetValue(float64(m.snapshot.TaskRunning), float64(max(m.snapshot.TaskDesired, 1)))
	taskTile.SetDelta(fmt.Sprintf("run=%d des=%d", m.snapshot.TaskRunning, m.snapshot.TaskDesired))
	taskTile.SetSeverity(metriccard.SeverityOK)
	tiles = append(tiles, taskTile.View())

	// ASG tile
	if m.snapshot.ASGName != "" {
		asgTile := metriccard.NewModel(m.ctx, "asg", "count", tileW)
		asgTile.SetValue(float64(m.snapshot.ASGInstances), float64(max(m.snapshot.ASGDesired, 1)))
		asgTile.SetDelta(fmt.Sprintf("in=%d des=%d", m.snapshot.ASGInstances, m.snapshot.ASGDesired))
		asgTile.SetSeverity(metriccard.SeverityOK)
		tiles = append(tiles, asgTile.View())
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, tiles...)
}

func (m Model) renderMetricsContent() string {
	if len(m.metrics) == 0 {
		return m.ctx.Styles.Common.FaintTextStyle.Render("No metrics available")
	}

	tileW := constants.TileWidthNormal
	metricsW := m.ctx.ContentWidth * constants.PanelSplitPct / 100
	innerW := metricsW - 4
	tilesPerRow := max(innerW/(tileW+2), 1)

	var tiles []string
	for _, mr := range m.metrics {
		unit := metricUnit(mr.ID)
		t := metriccard.NewModel(m.ctx, metricShortLabel(mr.ID), unit, tileW)
		if mr.Peak != nil && mr.Avg != nil {
			t.SetReportData(*mr.Peak, *mr.Avg, mr.Values)
		}
		tiles = append(tiles, t.View())
	}

	return common.RenderTileGrid(tiles, tilesPerRow)
}

func metricUnit(id string) string {
	switch {
	case strings.HasSuffix(id, "_cpu"), strings.HasSuffix(id, "_memory"),
		strings.Contains(id, "reservation"):
		return "%"
	case strings.Contains(id, "response_time"):
		return "ms"
	default:
		return ""
	}
}

func metricShortLabel(id string) string {
	switch id {
	case "service_cpu":
		return "cpu"
	case "service_memory":
		return "mem"
	case "alb_requests_per_target":
		return "rps"
	case "alb_response_time":
		return "p95"
	case "alb_5xx":
		return "5xx"
	case "cluster_cpu_reservation":
		return "cpuRes"
	case "cluster_memory_reservation":
		return "memRes"
	default:
		if len(id) > 6 {
			return id[:6]
		}
		return id
	}
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

func (m Model) renderActivitiesTimeline() string {
	start, errS := time.Parse(time.RFC3339, m.startTime)
	end, errE := time.Parse(time.RFC3339, m.endTime)
	if errS != nil || errE != nil {
		return m.renderActivitiesContent()
	}

	eventsW := m.ctx.ContentWidth - m.ctx.ContentWidth*constants.PanelSplitPct/100
	innerW := eventsW - 4

	tl := timeline.NewModel(m.ctx, start, end, innerW)

	// Add metric lanes
	for _, mr := range m.metrics {
		if mr.Peak == nil || len(mr.Values) == 0 {
			continue
		}
		unit := metricUnit(mr.ID)
		label := metricShortLabel(mr.ID)
		lane := timeline.Lane{
			Label:  label,
			Values: mr.Values,
			Peak:   *mr.Peak,
			Unit:   unit,
		}
		if strings.HasSuffix(mr.ID, "_cpu") {
			lane.Threshold = 90
		}
		tl.AddLane(lane)
	}

	// Scaling and alarm events
	tl.AddScalingEvents(m.activities.ServiceScaling)
	tl.AddAlarmEvents(m.activities.Alarms)

	return tl.View()
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
	return overlay.RenderHelp(m.ctx, []overlay.HelpGroup{
		{Title: "Navigation", Keys: [][2]string{
			{"q", "Quit"},
			{"?", "Toggle help"},
			{"esc", "Close help / collapse panel"},
		}},
		{Title: "Panels", Keys: [][2]string{
			{"tab / shift+tab", "Next / previous panel"},
			{"1-3", "Jump to panel"},
			{"+", "Toggle expand (normal / full)"},
			{"↑↓ / j k", "Scroll focused panel"},
		}},
		{Title: "Actions", Keys: [][2]string{
			{"e", "Export JSON report"},
		}},
	})
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
