// Package runtui implements the Bubble Tea model for k6delta run.
package runtui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	k6runner "github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/stepper"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
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
	app         config.ResolvedApp
	provider    provider.InfraProvider
	baseURL     string
	skipAnalyze bool

	ctx *tuictx.ProgramContext

	currentPhase  runPhase
	stepper       stepper.Model
	headerComp    header.Model
	footerComp    footer.Model
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

// ProgressMsg is sent by the provider's OnProgress callback via tea.Program.Send.
type ProgressMsg struct {
	ID      string
	Current int
	Total   int
}

// NewModel creates a new run Model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, baseURL string, skipAnalyze bool) Model {
	ctx := tuictx.New(80, 24)

	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	s.Style = lipgloss.NewStyle().Foreground(ctx.Theme.HeaderText)

	st := stepper.NewModel(ctx,
		"AWS credentials",
		"Pre-snapshot",
		"Running k6",
		"Post-snapshot",
		"Analysis",
		"Report",
	)

	hdr := header.NewModel(ctx, app.Name, app.Env, app.Phase)
	ftr := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
	})

	prefix := k6runner.GenerateResultsPrefix(app.Name, app.Phase, app.Env)

	return Model{
		app:           app,
		provider:      prov,
		baseURL:       baseURL,
		skipAnalyze:   skipAnalyze,
		ctx:           ctx,
		currentPhase:  phaseInit,
		stepper:       st,
		headerComp:    hdr,
		footerComp:    ftr,
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
		m.ctx.Resize(msg.Width, msg.Height)
		m.headerComp.UpdateContext(m.ctx)
		m.stepper.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case ProgressMsg:
		detail := fmt.Sprintf("%s [%d/%d]", msg.ID, msg.Current, msg.Total)
		m.stepper.SetDetail(m.currentStepIndex(), detail)
		return m, nil

	case authOKMsg:
		m.stepper.MarkDone(stepAuth, "verified")
		m.stepper.MarkRunning(stepPreSnapshot)
		m.currentPhase = phasePreSnapshot
		return m, m.fetchSnapshot("pre")

	case snapshotMsg:
		switch msg.label {
		case "pre":
			m.preSnapshot = msg.snapshot
			detail := fmt.Sprintf("tasks=%d/%d asg=%d/%d",
				msg.snapshot.TaskRunning, msg.snapshot.TaskDesired,
				msg.snapshot.ASGInstances, msg.snapshot.ASGDesired)
			m.stepper.MarkDone(stepPreSnapshot, detail)
			m.stepper.MarkRunning(stepK6)
			m.currentPhase = phaseK6Run
			m.startTime = time.Now().UTC()
			return m, m.runK6()
		case "post":
			m.postSnapshot = msg.snapshot
			detail := fmt.Sprintf("tasks=%d/%d asg=%d/%d",
				msg.snapshot.TaskRunning, msg.snapshot.TaskDesired,
				msg.snapshot.ASGInstances, msg.snapshot.ASGDesired)
			m.stepper.MarkDone(stepPostSnapshot, detail)
			if m.skipAnalyze {
				m.stepper.MarkDone(stepAnalysis, "skipped")
				m.stepper.MarkRunning(stepReport)
				m.currentPhase = phaseReport
				return m, m.buildReport()
			}
			m.stepper.MarkRunning(stepAnalysis)
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
		m.stepper.MarkDone(stepK6, exitDetail)
		m.stepper.MarkRunning(stepPostSnapshot)
		m.currentPhase = phasePostSnapshot
		return m, m.fetchSnapshot("post")

	case analysisMsg:
		m.metrics = msg.metrics
		m.activities = msg.activities
		detail := fmt.Sprintf("%d metrics", len(msg.metrics))
		m.stepper.MarkDone(stepAnalysis, detail)
		m.stepper.MarkRunning(stepReport)
		m.currentPhase = phaseReport
		return m, m.buildReport()

	case reportMsg:
		m.report = msg.report
		m.reportPath = msg.path
		m.stepper.MarkDone(stepReport, "written")
		m.currentPhase = phaseDone
		return m, nil

	case errMsg:
		m.err = msg.err
		stepIdx := m.currentStepIndex()
		m.stepper.MarkFailed(stepIdx, msg.err.Error())
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	cs := m.ctx.Styles.Common
	var sections []string

	sections = append(sections, m.headerComp.View(), "")
	sections = append(sections, m.stepper.View())

	if m.err != nil {
		sections = append(sections, cs.ErrorStyle.Render("Error: "+m.err.Error()), "")
		sections = append(sections, m.footerComp.View())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	if m.currentPhase != phaseDone {
		sections = append(sections, m.spinner.View()+" "+m.phaseDescription())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	sections = append(sections, m.renderReport(), "")
	sections = append(sections, m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
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
	k6Exit := 0
	if m.k6Result != nil {
		k6Exit = m.k6Result.ExitCode
	}
	alb5xx := 0
	var ecsCPUPeak *float64
	for _, mr := range m.metrics {
		switch mr.ID {
		case "alb_5xx":
			if mr.Peak != nil {
				alb5xx = int(*mr.Peak)
			}
		case "ecs_cpu":
			ecsCPUPeak = mr.Peak
		}
	}

	v := computeVerdict(verdictInput{
		k6Exit:      k6Exit,
		alb5xx:      alb5xx,
		ecsCPUPeak:  ecsCPUPeak,
		tasksBefore: m.preSnapshot.TaskRunning,
		tasksAfter:  m.postSnapshot.TaskRunning,
		activities:  m.activities,
	})

	var verdictStyle lipgloss.Style
	switch v.level {
	case verdictFail:
		verdictStyle = s.Verdict.Fail
	case verdictWarn:
		verdictStyle = s.Verdict.Warn
	default:
		verdictStyle = s.Verdict.Pass
	}
	sections = append(sections, "", "  "+s.Common.BoldStyle.Render("Verdict: ")+verdictStyle.Render(v.level.String()))
	for _, reason := range v.reasons {
		icon := "\u2713"
		switch v.level {
		case verdictFail:
			icon = "\u2717"
		case verdictWarn:
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

func fmtDelta(before, after int) string {
	diff := after - before
	if diff == 0 {
		return "  -"
	}
	if diff > 0 {
		return fmt.Sprintf("+%d \u2191", diff)
	}
	return fmt.Sprintf("%d \u2193", diff)
}

func metricLabel(id string) string {
	switch id {
	case "ecs_cpu":
		return "ECS CPU"
	case "ecs_memory":
		return "ECS Memory"
	case "cluster_cpu_reservation":
		return "Cluster CPU Reservation"
	case "cluster_memory_reservation":
		return "Cluster Mem Reservation"
	case "capacity_provider_reservation":
		return "Capacity Provider Res."
	case "alb_response_time":
		return "ALB Response Time (p95)"
	case "alb_5xx":
		return "ALB 5xx"
	case "alb_requests_per_target":
		return "ALB Req/Target"
	case "alb_healthy_hosts":
		return "ALB Healthy Hosts"
	case "asg_desired":
		return "ASG Desired"
	case "asg_in_service":
		return "ASG In-Service"
	default:
		return ""
	}
}

func fmtMetricValue(id string, v float64) string {
	switch id {
	case "alb_response_time":
		return fmt.Sprintf("%.0fms", v*1000)
	case "alb_5xx", "alb_requests_per_target":
		return fmt.Sprintf("%.0f", v)
	case "alb_healthy_hosts", "asg_desired", "asg_in_service":
		return fmt.Sprintf("%.0f", v)
	default:
		return fmt.Sprintf("%.1f%%", v)
	}
}
