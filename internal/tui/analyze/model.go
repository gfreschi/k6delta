// Package analyzetui implements the Bubble Tea model for k6delta analyze.
package analyzetui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/stepper"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
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
}

// NewModel creates a new analyze TUI model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, startTime, endTime string, period int32, jsonOutput bool, outputFile string) Model {
	ctx := tuictx.New(80, 24)

	s := spinner.New(spinner.WithSpinner(spinner.Dot),
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
		m.stepper.MarkDone(stepAuth, "verified")
		m.phase = phaseFetchState
		m.stepper.MarkRunning(stepState)
		return m, m.fetchState()

	case stateDoneMsg:
		m.snapshot = msg.snapshot
		detail := fmt.Sprintf("tasks=%d/%d", msg.snapshot.TaskRunning, msg.snapshot.TaskDesired)
		m.stepper.MarkDone(stepState, detail)
		m.phase = phaseFetchMetrics
		m.stepper.MarkRunning(stepMetrics)
		return m, m.fetchMetrics()

	case metricsDoneMsg:
		m.metrics = msg.metrics
		m.stepper.MarkDone(stepMetrics, fmt.Sprintf("%d metric series", len(msg.metrics)))
		m.phase = phaseFetchActivities
		m.stepper.MarkRunning(stepActivities)
		return m, m.fetchActivities()

	case activitiesDoneMsg:
		m.activities = msg.activities
		detail := fmt.Sprintf("%d ECS, %d ASG, %d alarms",
			len(msg.activities.ECSScaling), len(msg.activities.ASGScaling), len(msg.activities.Alarms))
		m.stepper.MarkDone(stepActivities, detail)
		m.phase = phaseDisplay

		if m.outputFile != "" {
			if err := m.writeOutputFile(); err != nil {
				m.err = err
			}
		}
		return m, nil

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

	// Current State
	sections = append(sections, s.Header.Root.Render("Current State"), "")
	sections = append(sections, fmt.Sprintf("  ECS Tasks:  running=%d  desired=%d",
		m.snapshot.TaskRunning, m.snapshot.TaskDesired))
	if m.snapshot.ASGName != "" {
		sections = append(sections, fmt.Sprintf("  ASG:        in_service=%d  desired=%d",
			m.snapshot.ASGInstances, m.snapshot.ASGDesired))
	}
	sections = append(sections, "")

	// Metrics
	sections = append(sections, s.Header.Root.Render("Metrics"))
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
	sections = append(sections, metricTbl.View(), "")

	// Scaling Activities
	sections = append(sections, s.Header.Root.Render("Scaling Activities"), "")
	hasActivities := len(m.activities.ECSScaling) > 0 || len(m.activities.ASGScaling) > 0
	if hasActivities {
		if len(m.activities.ECSScaling) > 0 {
			sections = append(sections, s.Common.FaintTextStyle.Render("  ECS Task Scaling:"))
			for _, a := range m.activities.ECSScaling {
				desc := a.Description
				if desc == "" {
					desc = a.Cause
				}
				sections = append(sections, fmt.Sprintf("    %s  %s  %s", a.Time, a.Status, desc))
			}
			sections = append(sections, "")
		}
		if len(m.activities.ASGScaling) > 0 {
			sections = append(sections, s.Common.FaintTextStyle.Render("  ASG Scaling:"))
			for _, a := range m.activities.ASGScaling {
				desc := a.Description
				if desc == "" {
					desc = a.Cause
				}
				sections = append(sections, fmt.Sprintf("    %s  %s  %s", a.Time, a.Status, desc))
			}
			sections = append(sections, "")
		}
	} else {
		sections = append(sections, s.Common.FaintTextStyle.Render("  No scaling activities during test window"), "")
	}

	// Alarm History
	if len(m.activities.Alarms) > 0 {
		sections = append(sections, s.Header.Root.Render("Alarm History"), "")
		for _, a := range m.activities.Alarms {
			sections = append(sections, fmt.Sprintf("  %s  %s  %s -> %s", a.Time, a.AlarmName, a.OldState, a.NewState))
		}
		sections = append(sections, "")
	}

	if m.outputFile != "" && m.err == nil {
		sections = append(sections, s.Common.SuccessStyle.Render(fmt.Sprintf("JSON report written to: %s", m.outputFile)), "")
	}

	sections = append(sections, m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- View helpers ---

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
