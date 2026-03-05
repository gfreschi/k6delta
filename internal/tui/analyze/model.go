// Package analyzetui implements the Bubble Tea model for k6delta analyze.
package analyzetui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/stepper"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/keys"
)

type analyzePhase int

const (
	phaseAuth analyzePhase = iota
	phaseFetchState
	phaseFetchMetrics
	phaseFetchActivities
	phaseDisplay
)

const (
	stepAuth       = 0
	stepState      = 1
	stepMetrics    = 2
	stepActivities = 3
)

// Model is the Bubble Tea model for the analyze command.
type Model struct {
	app  config.ResolvedApp
	prov provider.InfraProvider

	startTime string
	endTime   string
	period    int32

	jsonOutput bool
	outputFile string

	ctx        *tuictx.ProgramContext
	stepper    stepper.Model
	headerComp header.Model
	footerComp footer.Model
	phase      analyzePhase

	snapshot   provider.Snapshot
	metrics    []provider.MetricResult
	activities provider.Activities

	spinner  spinner.Model
	err      error
	quitting bool

	// Dashboard state (active in phaseDisplay)
	focusMgr     *focus.Manager
	statePanel   panel.Model
	metricsPanel panel.Model
	eventsPanel  panel.Model
	rawMode      bool
}

// NewModel creates a new analyze TUI model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, startTime, endTime string, period int32, jsonOutput bool, outputFile string) Model {
	ctx := tuictx.New(80, 24)

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(ctx.Theme.HeaderText)))

	st := stepper.NewModel(ctx, "AWS credentials", "Current state", "CloudWatch metrics", "Scaling activities")

	hdr := header.NewModel(ctx, app.Name, app.Env, "analyze")
	ftr := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
	})

	return Model{
		app:        app,
		prov:       prov,
		startTime:  startTime,
		endTime:    endTime,
		period:     period,
		jsonOutput: jsonOutput,
		outputFile: outputFile,
		ctx:        ctx,
		stepper:    st,
		headerComp: hdr,
		footerComp: ftr,
		phase:      phaseAuth,
		spinner:    s,
	}
}

type authOKMsg struct{}
type stateDoneMsg struct{ snapshot provider.Snapshot }
type metricsDoneMsg struct{ metrics []provider.MetricResult }
type activitiesDoneMsg struct{ activities provider.Activities }
type errMsg struct{ err error }

// ProgressMsg is sent by the provider's OnProgress callback via tea.Program.Send.
type ProgressMsg struct {
	ID      string
	Current int
	Total   int
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.checkAuth())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.phase == phaseDisplay {
			switch {
			case key.Matches(msg, keys.Keys.NextPanel):
				m.focusMgr.Next()
				return m, m.updateDashboardFocus()
			case key.Matches(msg, keys.Keys.PrevPanel):
				m.focusMgr.Prev()
				return m, m.updateDashboardFocus()
			case key.Matches(msg, keys.Keys.Down):
				m.scrollFocusedPanel(1)
				return m, nil
			case key.Matches(msg, keys.Keys.Up):
				m.scrollFocusedPanel(-1)
				return m, nil
			case key.Matches(msg, keys.AnalyzeKeys.Export):
				if m.outputFile != "" {
					if err := m.writeOutputFile(); err != nil {
						m.err = err
					}
				}
				return m, nil
			}
		}
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.ctx.Resize(msg.Width, msg.Height)
		m.headerComp.UpdateContext(m.ctx)
		m.stepper.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
		if m.focusMgr != nil {
			m.statePanel.UpdateContext(m.ctx)
			m.metricsPanel.UpdateContext(m.ctx)
			m.eventsPanel.UpdateContext(m.ctx)
			m.resizeDashboardPanels()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case ProgressMsg:
		detail := fmt.Sprintf("%s [%d/%d]", msg.ID, msg.Current, msg.Total)
		for i, step := range m.stepper.Steps() {
			if step.Status == stepper.StepRunning {
				m.stepper.SetDetail(i, detail)
				break
			}
		}
		return m, nil

	case authOKMsg:
		flashCmd := m.stepper.MarkDone(stepAuth, "verified")
		m.phase = phaseFetchState
		m.stepper.MarkRunning(stepState)
		return m, tea.Batch(flashCmd, m.fetchState())

	case stateDoneMsg:
		m.snapshot = msg.snapshot
		detail := fmt.Sprintf("tasks=%d/%d", msg.snapshot.TaskRunning, msg.snapshot.TaskDesired)
		flashCmd := m.stepper.MarkDone(stepState, detail)
		m.phase = phaseFetchMetrics
		m.stepper.MarkRunning(stepMetrics)
		return m, tea.Batch(flashCmd, m.fetchMetrics())

	case metricsDoneMsg:
		m.metrics = msg.metrics
		flashCmd := m.stepper.MarkDone(stepMetrics, fmt.Sprintf("%d metric series", len(msg.metrics)))
		m.phase = phaseFetchActivities
		m.stepper.MarkRunning(stepActivities)
		return m, tea.Batch(flashCmd, m.fetchActivities())

	case activitiesDoneMsg:
		m.activities = msg.activities
		detail := fmt.Sprintf("%d ECS, %d ASG, %d alarms",
			len(msg.activities.ServiceScaling), len(msg.activities.NodeScaling), len(msg.activities.Alarms))
		flashCmd := m.stepper.MarkDone(stepActivities, detail)
		m.phase = phaseDisplay
		m.initDashboard()

		if m.outputFile != "" {
			if err := m.writeOutputFile(); err != nil {
				m.err = err
			}
		}
		return m, flashCmd

	case stepper.ClearFlashMsg:
		m.stepper.ClearFlash(msg.Index)
		return m, nil

	case panel.TransitionTickMsg:
		if m.focusMgr != nil {
			cmd := tea.Batch(
				m.statePanel.AdvanceTransition(),
				m.metricsPanel.AdvanceTransition(),
				m.eventsPanel.AdvanceTransition(),
			)
			return m, cmd
		}

	case errMsg:
		m.err = msg.err
		for i, step := range m.stepper.Steps() {
			if step.Status == stepper.StepRunning {
				m.stepper.MarkFailed(i, msg.err.Error())
				break
			}
		}
		return m, nil
	}

	return m, nil
}

// --- Commands ---

func (m Model) checkAuth() tea.Cmd {
	return func() tea.Msg {
		if err := m.prov.CheckCredentials(context.Background()); err != nil {
			return errMsg{err: err}
		}
		return authOKMsg{}
	}
}

func (m Model) fetchState() tea.Cmd {
	return func() tea.Msg {
		snap, err := m.prov.TakeSnapshot(context.Background())
		if err != nil {
			return errMsg{err: fmt.Errorf("snapshot: %w", err)}
		}
		return stateDoneMsg{snapshot: snap}
	}
}

func (m Model) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		start, err := time.Parse(time.RFC3339, m.startTime)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse start time: %w", err)}
		}
		end, err := time.Parse(time.RFC3339, m.endTime)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse end time: %w", err)}
		}
		metrics, err := m.prov.FetchMetrics(context.Background(), start, end, m.period)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch metrics: %w", err)}
		}
		return metricsDoneMsg{metrics: metrics}
	}
}

func (m Model) fetchActivities() tea.Cmd {
	return func() tea.Msg {
		start, err := time.Parse(time.RFC3339, m.startTime)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse start time: %w", err)}
		}
		end, err := time.Parse(time.RFC3339, m.endTime)
		if err != nil {
			return errMsg{err: fmt.Errorf("parse end time: %w", err)}
		}
		activities, err := m.prov.FetchActivities(context.Background(), start, end)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch activities: %w", err)}
		}
		return activitiesDoneMsg{activities: activities}
	}
}

// --- View ---

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	s := m.ctx.Styles
	var sections []string

	sections = append(sections, m.headerComp.View(), "")
	sections = append(sections,
		fmt.Sprintf("  %s  %s", s.Common.FaintTextStyle.Render("Cluster:"), m.app.Cluster),
		fmt.Sprintf("  %s  %s", s.Common.FaintTextStyle.Render("Service:"), m.app.Service),
		fmt.Sprintf("  %s   %s -> %s", s.Common.FaintTextStyle.Render("Window:"), m.startTime, m.endTime),
		fmt.Sprintf("  %s   %ds", s.Common.FaintTextStyle.Render("Period:"), m.period),
		"")

	if m.phase != phaseDisplay {
		sections = append(sections, m.stepper.View())
		if m.err == nil {
			sections = append(sections, "  "+m.spinner.View())
		}
		if m.err != nil {
			sections = append(sections, "",
				s.Common.ErrorStyle.Render(fmt.Sprintf("  Error: %s", m.err.Error())),
				"", m.footerComp.View())
		}
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	sections = append(sections, m.viewDashboard())

	if m.outputFile != "" && m.err == nil {
		sections = append(sections, s.Common.SuccessStyle.Render(fmt.Sprintf("JSON report written to: %s", m.outputFile)))
	}

	sections = append(sections, "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- Dashboard ---

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
	if m.rawMode {
		return m.renderRawDisplay()
	}

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

// RunJSON executes the analyze pipeline synchronously and outputs JSON.
func RunJSON(app config.ResolvedApp, prov provider.InfraProvider, startTime, endTime string, period int32, outputFile string) error {
	ctx := context.Background()

	if err := prov.CheckCredentials(ctx); err != nil {
		return err
	}

	start, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return fmt.Errorf("parse start time: %w", err)
	}
	end, err := time.Parse(time.RFC3339, endTime)
	if err != nil {
		return fmt.Errorf("parse end time: %w", err)
	}

	metrics, err := prov.FetchMetrics(ctx, start, end, period)
	if err != nil {
		return fmt.Errorf("fetch metrics: %w", err)
	}

	activities, err := prov.FetchActivities(ctx, start, end)
	if err != nil {
		return fmt.Errorf("fetch activities: %w", err)
	}

	m := &Model{
		app:        app,
		startTime:  startTime,
		endTime:    endTime,
		period:     period,
		metrics:    metrics,
		activities: activities,
	}

	data := m.buildJSONOutput()
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	fmt.Println(string(encoded))

	if outputFile != "" {
		return os.WriteFile(outputFile, encoded, 0o644)
	}
	return nil
}
