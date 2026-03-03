// Package runtui implements the Bubble Tea model for k6delta run.
package runtui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	k6runner "github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui"
)

type runPhase int

const (
	phaseInit runPhase = iota
	phaseAuth
	phasePreSnapshot
	phaseK6Run
	phasePostSnapshot
	phaseAnalysis
	phaseReport
	phaseDone
)

const (
	stepAuth         = 0
	stepPreSnapshot  = 1
	stepK6           = 2
	stepPostSnapshot = 3
	stepAnalysis     = 4
	stepReport       = 5
)

// Model is the Bubble Tea model for k6delta run.
type Model struct {
	app      config.ResolvedApp
	provider provider.InfraProvider
	baseURL  string
	skipAnalyze bool

	currentPhase  runPhase
	steps         *tui.StepTracker
	resultsPrefix string

	preSnapshot  provider.Snapshot
	postSnapshot provider.Snapshot

	k6Result   *k6runner.RunResult
	metrics    []provider.MetricResult
	activities provider.Activities

	report     *report.UnifiedReport
	reportPath string

	startTime time.Time
	endTime   time.Time

	spinner  spinner.Model
	width    int
	height   int
	err      error
	quitting bool
}

type authOKMsg struct{}
type snapshotMsg struct {
	snapshot provider.Snapshot
	label    string
}
type k6DoneMsg struct{ result k6runner.RunResult }
type analysisMsg struct {
	metrics    []provider.MetricResult
	activities provider.Activities
}
type reportMsg struct {
	report *report.UnifiedReport
	path   string
}
type errMsg struct{ err error }

// NewModel creates a new run Model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, baseURL string, skipAnalyze bool) Model {
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("12"))

	steps := tui.NewStepTracker(
		"AWS credentials",
		"Pre-snapshot",
		"Running k6",
		"Post-snapshot",
		"Analysis",
		"Report",
	)

	prefix := k6runner.GenerateResultsPrefix(app.Name, app.Phase, app.Env)

	return Model{
		app:           app,
		provider:      prov,
		baseURL:       baseURL,
		skipAnalyze:   skipAnalyze,
		currentPhase:  phaseInit,
		steps:         steps,
		resultsPrefix: prefix,
		spinner:       s,
	}
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
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case authOKMsg:
		m.steps.MarkDone(stepAuth, "verified")
		m.steps.MarkRunning(stepPreSnapshot)
		m.currentPhase = phasePreSnapshot
		return m, m.fetchSnapshot("pre")

	case snapshotMsg:
		switch msg.label {
		case "pre":
			m.preSnapshot = msg.snapshot
			detail := fmt.Sprintf("tasks=%d/%d asg=%d/%d",
				msg.snapshot.TaskRunning, msg.snapshot.TaskDesired,
				msg.snapshot.ASGInstances, msg.snapshot.ASGDesired)
			m.steps.MarkDone(stepPreSnapshot, detail)
			m.steps.MarkRunning(stepK6)
			m.currentPhase = phaseK6Run
			m.startTime = time.Now().UTC()
			return m, m.runK6()
		case "post":
			m.postSnapshot = msg.snapshot
			detail := fmt.Sprintf("tasks=%d/%d asg=%d/%d",
				msg.snapshot.TaskRunning, msg.snapshot.TaskDesired,
				msg.snapshot.ASGInstances, msg.snapshot.ASGDesired)
			m.steps.MarkDone(stepPostSnapshot, detail)
			if m.skipAnalyze {
				m.steps.MarkDone(stepAnalysis, "skipped")
				m.steps.MarkRunning(stepReport)
				m.currentPhase = phaseReport
				return m, m.buildReport()
			}
			m.steps.MarkRunning(stepAnalysis)
			m.currentPhase = phaseAnalysis
			return m, m.fetchAnalysis()
		}

	case k6DoneMsg:
		m.k6Result = &msg.result
		m.endTime = time.Now().UTC()
		exitDetail := fmt.Sprintf("exit %d", msg.result.ExitCode)
		if msg.result.ExitCode == 0 {
			exitDetail += " (all thresholds passed)"
		}
		m.steps.MarkDone(stepK6, exitDetail)
		m.steps.MarkRunning(stepPostSnapshot)
		m.currentPhase = phasePostSnapshot
		return m, m.fetchSnapshot("post")

	case analysisMsg:
		m.metrics = msg.metrics
		m.activities = msg.activities
		detail := fmt.Sprintf("%d metrics", len(msg.metrics))
		m.steps.MarkDone(stepAnalysis, detail)
		m.steps.MarkRunning(stepReport)
		m.currentPhase = phaseReport
		return m, m.buildReport()

	case reportMsg:
		m.report = msg.report
		m.reportPath = msg.path
		m.steps.MarkDone(stepReport, "written")
		m.currentPhase = phaseDone
		return m, nil

	case errMsg:
		m.err = msg.err
		stepIdx := m.currentStepIndex()
		m.steps.MarkFailed(stepIdx, msg.err.Error())
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	header := fmt.Sprintf("k6delta: %s (%s) -- %s", m.app.Name, m.app.Env, m.app.Phase)
	b.WriteString(tui.HeaderStyle.Render(header))
	b.WriteByte('\n')
	b.WriteByte('\n')

	b.WriteString(m.steps.View())

	if m.err != nil {
		b.WriteByte('\n')
		b.WriteString(tui.ErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteByte('\n')
		b.WriteByte('\n')
		b.WriteString(tui.DimStyle.Render("Press q to quit"))
		b.WriteByte('\n')
		return b.String()
	}

	if m.currentPhase != phaseDone {
		b.WriteByte('\n')
		b.WriteString(m.spinner.View() + " " + m.phaseDescription())
		b.WriteByte('\n')
		return b.String()
	}

	b.WriteByte('\n')
	b.WriteString(m.renderReport())
	b.WriteByte('\n')
	b.WriteString(tui.DimStyle.Render("Press q to quit"))
	b.WriteByte('\n')

	return b.String()
}

// --- Commands ---

func (m Model) checkAuth() tea.Cmd {
	return func() tea.Msg {
		if err := m.provider.CheckCredentials(context.Background()); err != nil {
			return errMsg{err: err}
		}
		return authOKMsg{}
	}
}

func (m Model) fetchSnapshot(label string) tea.Cmd {
	return func() tea.Msg {
		snap, err := m.provider.TakeSnapshot(context.Background())
		if err != nil {
			return errMsg{err: fmt.Errorf("snapshot (%s): %w", label, err)}
		}
		return snapshotMsg{snapshot: snap, label: label}
	}
}

func (m Model) runK6() tea.Cmd {
	cfg := k6runner.RunConfig{
		TestFile:      m.app.TestFile,
		Env:           m.app.Env,
		ResultsPrefix: m.resultsPrefix,
		ResultsDir:    m.app.ResultsDir,
	}
	if m.baseURL != "" {
		cfg.BaseURL = m.baseURL
	}

	args := k6runner.BuildArgs(cfg)
	env := k6runner.BuildEnv(cfg)

	c := exec.Command("k6", args...)
	c.Env = env

	startTime := m.startTime

	return tea.ExecProcess(c, func(err error) tea.Msg {
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				return errMsg{err: fmt.Errorf("k6 exec: %w", err)}
			}
		}
		return k6DoneMsg{result: k6runner.RunResult{
			ExitCode:  exitCode,
			StartTime: startTime,
			EndTime:   time.Now().UTC(),
		}}
	})
}

func (m Model) fetchAnalysis() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		metrics, err := m.provider.FetchMetrics(ctx, m.startTime, m.endTime, 60)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch metrics: %w", err)}
		}

		activities, err := m.provider.FetchActivities(ctx, m.startTime, m.endTime)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch activities: %w", err)}
		}

		return analysisMsg{metrics: metrics, activities: activities}
	}
}

func buildInfraMetrics(metrics []provider.MetricResult) *report.InfraMetrics {
	infra := &report.InfraMetrics{}
	for _, m := range metrics {
		pa := &report.PeakAvg{Peak: m.Peak, Avg: m.Avg}
		switch m.ID {
		case "ecs_cpu":
			infra.ECSCPU = pa
		case "ecs_memory":
			infra.ECSMemory = pa
		case "cluster_cpu_reservation":
			infra.ClusterCPUReservation = pa
		case "cluster_memory_reservation":
			infra.ClusterMemoryReservation = pa
		case "capacity_provider_reservation":
			infra.CapacityProviderReservation = pa
		case "alb_response_time":
			infra.ALBResponseTimeP95 = m.Peak
		case "alb_5xx":
			if m.Peak != nil {
				infra.ALB5xx = int(*m.Peak)
			}
		}
	}
	return infra
}

func (m Model) buildReport() tea.Cmd {
	return func() tea.Msg {
		info := report.RunInfo{
			App:             m.app.Name,
			Env:             m.app.Env,
			Phase:           m.app.Phase,
			Start:           m.startTime.Format(time.RFC3339),
			End:             m.endTime.Format(time.RFC3339),
			DurationSeconds: int(m.endTime.Sub(m.startTime).Seconds()),
		}
		if m.k6Result != nil {
			info.K6Exit = m.k6Result.ExitCode
		}

		k6SummaryPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+"-summary.json")
		if _, err := os.Stat(k6SummaryPath); err != nil {
			k6SummaryPath = ""
		}

		infra := buildInfraMetrics(m.metrics)

		r, err := report.BuildUnifiedReportFromInfra(
			info, k6SummaryPath, infra,
			m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning,
			m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances,
		)
		if err != nil {
			return errMsg{err: fmt.Errorf("build report: %w", err)}
		}

		r.ScalingActivities = m.activities
		r.AlarmHistory = m.activities.Alarms

		reportPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+"-report.json")
		if err := report.WriteReport(r, reportPath); err != nil {
			return errMsg{err: fmt.Errorf("write report: %w", err)}
		}

		return reportMsg{report: r, path: reportPath}
	}
}

// --- Helpers ---

func (m Model) currentStepIndex() int {
	switch m.currentPhase {
	case phaseInit, phaseAuth:
		return stepAuth
	case phasePreSnapshot:
		return stepPreSnapshot
	case phaseK6Run:
		return stepK6
	case phasePostSnapshot:
		return stepPostSnapshot
	case phaseAnalysis:
		return stepAnalysis
	case phaseReport:
		return stepReport
	default:
		return stepReport
	}
}

func (m Model) phaseDescription() string {
	switch m.currentPhase {
	case phaseInit, phaseAuth:
		return "Checking AWS credentials..."
	case phasePreSnapshot:
		return "Capturing pre-test snapshot..."
	case phaseK6Run:
		return "Running k6 load test..."
	case phasePostSnapshot:
		return "Capturing post-test snapshot..."
	case phaseAnalysis:
		return "Fetching CloudWatch metrics..."
	case phaseReport:
		return "Building unified report..."
	default:
		return ""
	}
}

func (m Model) renderReport() string {
	var b strings.Builder

	separatorLine := strings.Repeat("-", 50)

	b.WriteString(tui.TitleStyle.Render("Load Test Report"))
	b.WriteByte('\n')
	b.WriteByte('\n')

	duration := m.endTime.Sub(m.startTime)
	minutes := int(duration.Minutes())
	seconds := int(duration.Seconds()) % 60
	b.WriteString(fmt.Sprintf("  %s%s -> %s  (%dm %ds)\n",
		tui.LabelStyle.Render("Duration"),
		m.startTime.Format("15:04:05"),
		m.endTime.Format("15:04:05"),
		minutes, seconds))

	exitStr := "0 (all thresholds passed)"
	exitStyle := tui.SuccessStyle
	if m.k6Result != nil && m.k6Result.ExitCode != 0 {
		exitStr = fmt.Sprintf("%d (threshold failures)", m.k6Result.ExitCode)
		exitStyle = tui.WarnStyle
	}
	b.WriteString(fmt.Sprintf("  %s%s\n",
		tui.LabelStyle.Render("k6 exit"),
		exitStyle.Render(exitStr)))

	if m.report != nil && m.report.K6 != nil {
		k6 := m.report.K6
		b.WriteByte('\n')
		b.WriteString(fmt.Sprintf("  %-26s %s\n", "Metric", "Value"))
		b.WriteString("  " + separatorLine)
		b.WriteByte('\n')
		b.WriteString(fmt.Sprintf("  %-26s %s\n", "p95 latency", fmtFloatMs(k6.P95ms)))
		b.WriteString(fmt.Sprintf("  %-26s %s\n", "Error rate", fmtPct(k6.ErrorRate)))
		b.WriteString(fmt.Sprintf("  %-26s %s\n", "Throughput", fmtFloatRate(k6.RPSAvg)))
		b.WriteString(fmt.Sprintf("  %-26s %s\n", "Checks", fmtPctRate(k6.ChecksRate)))
		b.WriteString(fmt.Sprintf("  %-26s %s\n", "Thresholds",
			fmt.Sprintf("%d passed, %d failed", k6.Thresholds.Passed, k6.Thresholds.Failed)))
	}

	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  %-26s %-12s %s\n", "Infrastructure", "Before", "After"))
	b.WriteString("  " + separatorLine)
	b.WriteByte('\n')
	b.WriteString(fmt.Sprintf("  %-26s %-12d %d\n", "ECS tasks",
		m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning))
	b.WriteString(fmt.Sprintf("  %-26s %-12d %d\n", "EC2 instances",
		m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances))

	b.WriteByte('\n')
	b.WriteString("  Output Files\n")
	k6SummaryPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+"-summary.json")
	if _, err := os.Stat(k6SummaryPath); err == nil {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", "k6 summary:", k6SummaryPath))
	}
	htmlPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+".html")
	if _, err := os.Stat(htmlPath); err == nil {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", "HTML report:", htmlPath))
	}
	if m.reportPath != "" {
		b.WriteString(fmt.Sprintf("  %-18s %s\n", "Unified report:", m.reportPath))
	}

	b.WriteByte('\n')
	return b.String()
}

func fmtFloatMs(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.1fms", *v)
}

func fmtPct(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.2f%%", *v*100)
}

func fmtFloatRate(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f/s", *v)
}

func fmtPctRate(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", *v*100)
}
