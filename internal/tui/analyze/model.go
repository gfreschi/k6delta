// Package analyzetui implements the Bubble Tea model for k6delta analyze.
package analyzetui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/tui"
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

	steps *tui.StepTracker
	phase analyzePhase

	snapshot   provider.Snapshot
	metrics    []provider.MetricResult
	activities provider.Activities

	spinner  spinner.Model
	width    int
	height   int
	err      error
	quitting bool
}

// NewModel creates a new analyze TUI model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, startTime, endTime string, period int32, jsonOutput bool, outputFile string) Model {
	s := spinner.New(spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("12"))))

	return Model{
		app:        app,
		prov:       prov,
		startTime:  startTime,
		endTime:    endTime,
		period:     period,
		jsonOutput: jsonOutput,
		outputFile: outputFile,
		steps:      tui.NewStepTracker("AWS credentials", "Current state", "CloudWatch metrics", "Scaling activities"),
		phase:      phaseAuth,
		spinner:    s,
	}
}

type authOKMsg struct{}
type stateDoneMsg struct{ snapshot provider.Snapshot }
type metricsDoneMsg struct{ metrics []provider.MetricResult }
type activitiesDoneMsg struct{ activities provider.Activities }
type errMsg struct{ err error }

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
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case authOKMsg:
		m.steps.MarkDone(stepAuth, "verified")
		m.phase = phaseFetchState
		m.steps.MarkRunning(stepState)
		return m, m.fetchState()

	case stateDoneMsg:
		m.snapshot = msg.snapshot
		detail := fmt.Sprintf("tasks=%d/%d", msg.snapshot.TaskRunning, msg.snapshot.TaskDesired)
		m.steps.MarkDone(stepState, detail)
		m.phase = phaseFetchMetrics
		m.steps.MarkRunning(stepMetrics)
		return m, m.fetchMetrics()

	case metricsDoneMsg:
		m.metrics = msg.metrics
		m.steps.MarkDone(stepMetrics, fmt.Sprintf("%d metric series", len(msg.metrics)))
		m.phase = phaseFetchActivities
		m.steps.MarkRunning(stepActivities)
		return m, m.fetchActivities()

	case activitiesDoneMsg:
		m.activities = msg.activities
		detail := fmt.Sprintf("%d ECS, %d ASG, %d alarms",
			len(msg.activities.ECSScaling), len(msg.activities.ASGScaling), len(msg.activities.Alarms))
		m.steps.MarkDone(stepActivities, detail)
		m.phase = phaseDisplay

		if m.outputFile != "" {
			if err := m.writeOutputFile(); err != nil {
				m.err = err
			}
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		for i := range m.steps.Steps {
			if m.steps.Steps[i].Status == tui.StepRunning {
				m.steps.MarkFailed(i, msg.err.Error())
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

	var b strings.Builder

	b.WriteString(tui.HeaderStyle.Render(fmt.Sprintf("Scaling Analysis: %s (%s)", m.app.Name, m.app.Env)))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  %s  %s\n", tui.DimStyle.Render("Cluster:"), m.app.Cluster))
	b.WriteString(fmt.Sprintf("  %s  %s\n", tui.DimStyle.Render("Service:"), m.app.Service))
	b.WriteString(fmt.Sprintf("  %s   %s -> %s\n", tui.DimStyle.Render("Window:"), m.startTime, m.endTime))
	b.WriteString(fmt.Sprintf("  %s   %ds\n", tui.DimStyle.Render("Period:"), m.period))
	b.WriteByte('\n')

	if m.phase != phaseDisplay {
		b.WriteString(m.steps.View())
		if m.err == nil {
			b.WriteString(fmt.Sprintf("\n  %s\n", m.spinner.View()))
		}
		if m.err != nil {
			b.WriteByte('\n')
			b.WriteString(tui.ErrorStyle.Render(fmt.Sprintf("  Error: %s", m.err.Error())))
			b.WriteByte('\n')
			b.WriteByte('\n')
			b.WriteString(tui.DimStyle.Render("  Press q to quit"))
			b.WriteByte('\n')
		}
		return b.String()
	}

	// Current State
	b.WriteString(tui.HeaderStyle.Render("Current State"))
	b.WriteByte('\n')
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  ECS Tasks:  running=%d  desired=%d\n",
		m.snapshot.TaskRunning, m.snapshot.TaskDesired))
	if m.snapshot.ASGName != "" {
		b.WriteString(fmt.Sprintf("  ASG:        in_service=%d  desired=%d\n",
			m.snapshot.ASGInstances, m.snapshot.ASGDesired))
	}
	b.WriteByte('\n')

	// Metrics
	b.WriteString(tui.HeaderStyle.Render("Metrics"))
	b.WriteByte('\n')
	b.WriteString(renderMetricTableHeader())
	for _, mr := range m.metrics {
		b.WriteString(renderMetricRow(mr))
	}
	b.WriteByte('\n')

	// Scaling Activities
	b.WriteString(tui.HeaderStyle.Render("Scaling Activities"))
	b.WriteByte('\n')
	b.WriteByte('\n')
	hasActivities := len(m.activities.ECSScaling) > 0 || len(m.activities.ASGScaling) > 0
	if hasActivities {
		if len(m.activities.ECSScaling) > 0 {
			b.WriteString(tui.DimStyle.Render("  ECS Task Scaling:"))
			b.WriteByte('\n')
			for _, a := range m.activities.ECSScaling {
				desc := a.Description
				if desc == "" {
					desc = a.Cause
				}
				b.WriteString(fmt.Sprintf("    %s  %s  %s\n", a.Time, a.Status, desc))
			}
			b.WriteByte('\n')
		}
		if len(m.activities.ASGScaling) > 0 {
			b.WriteString(tui.DimStyle.Render("  ASG Scaling:"))
			b.WriteByte('\n')
			for _, a := range m.activities.ASGScaling {
				desc := a.Description
				if desc == "" {
					desc = a.Cause
				}
				b.WriteString(fmt.Sprintf("    %s  %s  %s\n", a.Time, a.Status, desc))
			}
			b.WriteByte('\n')
		}
	} else {
		b.WriteString(tui.DimStyle.Render("  No scaling activities during test window"))
		b.WriteByte('\n')
		b.WriteByte('\n')
	}

	// Alarm History
	if len(m.activities.Alarms) > 0 {
		b.WriteString(tui.HeaderStyle.Render("Alarm History"))
		b.WriteByte('\n')
		b.WriteByte('\n')
		for _, a := range m.activities.Alarms {
			b.WriteString(fmt.Sprintf("  %s  %s  %s -> %s\n", a.Time, a.AlarmName, a.OldState, a.NewState))
		}
		b.WriteByte('\n')
	}

	if m.outputFile != "" && m.err == nil {
		b.WriteString(tui.SuccessStyle.Render(fmt.Sprintf("JSON report written to: %s", m.outputFile)))
		b.WriteByte('\n')
		b.WriteByte('\n')
	}

	b.WriteString(tui.DimStyle.Render("Press q to quit"))
	b.WriteByte('\n')

	return b.String()
}

// --- View helpers ---

func renderMetricTableHeader() string {
	return fmt.Sprintf("  %-35s %10s %10s %8s\n  %-35s %10s %10s %8s\n",
		"Metric", "Peak", "Avg", "Points",
		strings.Repeat("-", 35), strings.Repeat("-", 10), strings.Repeat("-", 10), strings.Repeat("-", 8))
}

func renderMetricRow(mr provider.MetricResult) string {
	if mr.Peak == nil || mr.Avg == nil {
		return fmt.Sprintf("  %-35s %10s %10s %8d\n", mr.ID, "-", "-", len(mr.Values))
	}
	return fmt.Sprintf("  %-35s %10.2f %10.2f %8d\n", mr.ID, *mr.Peak, *mr.Avg, len(mr.Values))
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
