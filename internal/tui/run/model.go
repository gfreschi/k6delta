// Package runtui implements the Bubble Tea model for k6delta run.
package runtui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	k6runner "github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/gauge"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/streamchart"
	"github.com/gfreschi/k6delta/internal/tui/components/timechart"
	"github.com/gfreschi/k6delta/internal/tui/components/trendline"
	"github.com/gfreschi/k6delta/internal/tui/components/stepper"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/keys"
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

	// Report dashboard state (active in phaseDone)
	focusMgr    *focus.Manager
	k6Panel     panel.Model
	graphsPanel panel.Model
	infraPanel  panel.Model
	eventsPanel panel.Model
	rawMode     bool

	// Report time-series charts
	reportRPSChart     timechart.Model
	reportLatencyChart timechart.Model

	// Live dashboard state
	streamingSupported bool
	liveMode           bool
	graphMode          bool
	k6PointChan        chan k6runner.K6Point
	k6Cancel           context.CancelFunc
	rpsChart           streamchart.Model
	latencyChart       streamchart.Model
	cpuGauge           gauge.Model
	memGauge           gauge.Model
	reservGauge        gauge.Model
	liveSnapshot       provider.Snapshot
	liveMetrics        []provider.MetricResult
	liveRPSCount       int
	liveRPSTime        time.Time

	// Live dashboard panels + focus
	liveFocusMgr   *focus.Manager
	liveGraphPanel panel.Model
	liveInfraPanel panel.Model

	// Infra trend sparklines
	cpuTrend    trendline.Model
	memTrend    trendline.Model
	reservTrend trendline.Model
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
type k6PointMsg struct{ point k6runner.K6Point }
type exportDoneMsg struct{ path string }
type openDoneMsg struct{ path string }

// ProgressMsg is sent by the provider's OnProgress callback via tea.Program.Send.
type ProgressMsg struct {
	ID      string
	Current int
	Total   int
}

// NewModel creates a new run Model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, baseURL string, skipAnalyze bool) Model {
	ctx := tuictx.New(80, 24)

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
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

	canStream, _ := k6runner.SupportsJSONStreaming()

	return Model{
		app:                app,
		provider:           prov,
		baseURL:            baseURL,
		skipAnalyze:        skipAnalyze,
		ctx:                ctx,
		currentPhase:       phaseInit,
		stepper:            st,
		headerComp:         hdr,
		footerComp:         ftr,
		resultsPrefix:      prefix,
		spinner:            s,
		streamingSupported: canStream,
		graphMode:          true,
		rpsChart:           streamchart.NewModel(ctx, "Throughput", "req/s", ctx.ContentWidth*55/100, 12),
		latencyChart:       streamchart.NewModel(ctx, "Latency", "ms", ctx.ContentWidth*55/100, 12),
		cpuGauge:           gauge.NewModel(ctx, "CPU", 30),
		memGauge:           gauge.NewModel(ctx, "Memory", 30),
		reservGauge:        gauge.NewModel(ctx, "Reserv", 30),
		cpuTrend:           trendline.NewModel(ctx, 30, 1),
		memTrend:           trendline.NewModel(ctx, 30, 1),
		reservTrend:        trendline.NewModel(ctx, 30, 1),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.checkAuth())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.liveMode {
			switch msg.String() {
			case "tab":
				if m.liveFocusMgr != nil {
					m.liveFocusMgr.Next()
					m.liveGraphPanel.SetFocused(m.liveFocusMgr.IsFocused(0))
					m.liveInfraPanel.SetFocused(m.liveFocusMgr.IsFocused(1))
					return m, tea.Batch(
						m.liveGraphPanel.TransitionCmd(),
						m.liveInfraPanel.TransitionCmd(),
					)
				}
				return m, nil
			case "shift+tab":
				if m.liveFocusMgr != nil {
					m.liveFocusMgr.Prev()
					m.liveGraphPanel.SetFocused(m.liveFocusMgr.IsFocused(0))
					m.liveInfraPanel.SetFocused(m.liveFocusMgr.IsFocused(1))
					return m, tea.Batch(
						m.liveGraphPanel.TransitionCmd(),
						m.liveInfraPanel.TransitionCmd(),
					)
				}
				return m, nil
			case "g":
				m.graphMode = !m.graphMode
				return m, nil
			case "a":
				if m.k6Cancel != nil {
					m.k6Cancel()
				}
				return m, nil
			}
		}
		if m.currentPhase == phaseDone {
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
			case key.Matches(msg, keys.RunKeys.Export):
				return m, m.exportReport()
			case key.Matches(msg, keys.RunKeys.OpenHTML):
				return m, m.openHTML()
			case key.Matches(msg, keys.RunKeys.RawView):
				m.rawMode = !m.rawMode
				return m, nil
			}
		}
		switch msg.String() {
		case "q", "ctrl+c":
			if m.k6Cancel != nil {
				m.k6Cancel()
			}
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.ctx.Resize(msg.Width, msg.Height)
		m.headerComp.UpdateContext(m.ctx)
		m.stepper.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
		m.rpsChart.UpdateContext(m.ctx)
		m.latencyChart.UpdateContext(m.ctx)
		width := m.ctx.ContentWidth
		switch {
		case width >= constants.BreakpointSplit:
			chartW := width*55/100 - 2
			m.rpsChart.Resize(chartW, 12)
			m.latencyChart.Resize(chartW, 12)
		case width >= constants.BreakpointStacked:
			m.rpsChart.Resize(width-2, 10)
			m.latencyChart.Resize(width-2, 10)
		}
		m.cpuGauge.UpdateContext(m.ctx)
		m.reservGauge.UpdateContext(m.ctx)
		m.memGauge.UpdateContext(m.ctx)
		m.cpuTrend.UpdateContext(m.ctx)
		m.memTrend.UpdateContext(m.ctx)
		m.reservTrend.UpdateContext(m.ctx)
		if m.liveFocusMgr != nil {
			leftW := m.ctx.ContentWidth * 55 / 100
			rightW := m.ctx.ContentWidth - leftW
			panelH := m.ctx.ContentHeight - 8
			m.liveGraphPanel.SetDimensions(leftW, panelH)
			m.liveInfraPanel.SetDimensions(rightW, panelH)
			m.liveGraphPanel.UpdateContext(m.ctx)
			m.liveInfraPanel.UpdateContext(m.ctx)
		}
		if m.focusMgr != nil {
			m.k6Panel.UpdateContext(m.ctx)
			m.infraPanel.UpdateContext(m.ctx)
			m.eventsPanel.UpdateContext(m.ctx)
			m.resizeDashboardPanels()
		}
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
		flashCmd := m.stepper.MarkDone(stepAuth, "verified")
		m.stepper.MarkRunning(stepPreSnapshot)
		m.currentPhase = phasePreSnapshot
		return m, tea.Batch(flashCmd, m.fetchSnapshot("pre"))

	case snapshotMsg:
		switch msg.label {
		case "pre":
			m.preSnapshot = msg.snapshot
			detail := fmt.Sprintf("tasks=%d/%d asg=%d/%d",
				msg.snapshot.TaskRunning, msg.snapshot.TaskDesired,
				msg.snapshot.ASGInstances, msg.snapshot.ASGDesired)
			flashCmd := m.stepper.MarkDone(stepPreSnapshot, detail)
			m.stepper.MarkRunning(stepK6)
			m.currentPhase = phaseK6Run
			m.startTime = time.Now().UTC()

			if !m.streamingSupported {
				return m, tea.Batch(flashCmd, m.runK6Fallback())
			}

			// Start streaming k6 with live dashboard
			m.k6PointChan = make(chan k6runner.K6Point, 256)
			m.liveMode = true

			m.liveFocusMgr = focus.New(2)
			leftW := m.ctx.ContentWidth * 55 / 100
			rightW := m.ctx.ContentWidth - leftW
			panelH := m.ctx.ContentHeight - 8
			m.liveGraphPanel = panel.NewModel(m.ctx, "Graphs [1]", leftW, panelH)
			m.liveGraphPanel.SetFocused(true)
			m.liveInfraPanel = panel.NewModel(m.ctx, "Infrastructure [2]", rightW, panelH)
			k6Ctx, cancel := context.WithCancel(context.Background())
			m.k6Cancel = cancel
			m.footerComp.SetHints([]footer.KeyHint{
				{Key: "q", Action: "quit"},
				{Key: "tab", Action: "next panel"},
				{Key: "g", Action: "graphs"},
				{Key: "a", Action: "abort"},
			})
			return m, tea.Batch(
				flashCmd,
				m.runK6Streaming(k6Ctx),
				m.waitForK6Point(),
				infraPollCmd(context.Background(), m.provider, 15*time.Second),
			)
		case "post":
			m.postSnapshot = msg.snapshot
			detail := fmt.Sprintf("tasks=%d/%d asg=%d/%d",
				msg.snapshot.TaskRunning, msg.snapshot.TaskDesired,
				msg.snapshot.ASGInstances, msg.snapshot.ASGDesired)
			flashCmd := m.stepper.MarkDone(stepPostSnapshot, detail)
			if m.skipAnalyze {
				flashCmd2 := m.stepper.MarkDone(stepAnalysis, "skipped")
				m.stepper.MarkRunning(stepReport)
				m.currentPhase = phaseReport
				return m, tea.Batch(flashCmd, flashCmd2, m.buildReport())
			}
			m.stepper.MarkRunning(stepAnalysis)
			m.currentPhase = phaseAnalysis
			return m, tea.Batch(flashCmd, m.fetchAnalysis())
		}

	case k6PointMsg:
		m.handleK6Point(msg.point)
		return m, m.waitForK6Point()

	case infraTickMsg:
		m.liveSnapshot = msg.snapshot
		m.liveMetrics = msg.metrics
		m.updateGaugesFromMetrics(msg.metrics)
		if m.liveMode {
			return m, infraPollCmd(context.Background(), m.provider, 15*time.Second)
		}
		return m, nil

	case k6DoneMsg:
		m.liveMode = false
		m.k6Cancel = nil
		m.footerComp = footer.NewModel(m.ctx, []footer.KeyHint{
			{Key: "q", Action: "quit"},
		})
		m.k6Result = &msg.result
		m.endTime = time.Now().UTC()
		exitDetail := fmt.Sprintf("exit %d", msg.result.ExitCode)
		if msg.result.ExitCode == 0 {
			exitDetail += " (all thresholds passed)"
		}
		flashCmd := m.stepper.MarkDone(stepK6, exitDetail)
		m.stepper.MarkRunning(stepPostSnapshot)
		m.currentPhase = phasePostSnapshot
		return m, tea.Batch(flashCmd, m.fetchSnapshot("post"))

	case analysisMsg:
		m.metrics = msg.metrics
		m.activities = msg.activities
		detail := fmt.Sprintf("%d metrics", len(msg.metrics))
		flashCmd := m.stepper.MarkDone(stepAnalysis, detail)
		m.stepper.MarkRunning(stepReport)
		m.currentPhase = phaseReport
		return m, tea.Batch(flashCmd, m.buildReport())

	case reportMsg:
		m.report = msg.report
		m.reportPath = msg.path
		flashCmd := m.stepper.MarkDone(stepReport, "written")
		m.currentPhase = phaseDone
		m.initDashboard()
		return m, flashCmd

	case exportDoneMsg:
		m.footerComp.SetHints([]footer.KeyHint{
			{Key: "q", Action: "quit"},
			{Key: "tab", Action: "next panel"},
			{Key: "\u2191\u2193", Action: "scroll"},
			{Key: "e", Action: "export JSON"},
			{Key: "o", Action: "open HTML"},
			{Key: "r", Action: "raw view"},
		})
		return m, nil

	case openDoneMsg:
		return m, nil

	case stepper.ClearFlashMsg:
		m.stepper.ClearFlash(msg.Index)
		return m, nil

	case panel.TransitionTickMsg:
		if m.liveMode && m.liveFocusMgr != nil {
			var cmds []tea.Cmd
			if cmd := m.liveGraphPanel.AdvanceTransition(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			if cmd := m.liveInfraPanel.AdvanceTransition(); cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}
		if m.focusMgr != nil {
			cmd := tea.Batch(
				m.k6Panel.AdvanceTransition(),
				m.infraPanel.AdvanceTransition(),
				m.eventsPanel.AdvanceTransition(),
			)
			return m, cmd
		}

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

	if m.liveMode {
		return m.viewLiveDashboard()
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

	sections = append(sections, m.viewReportDashboard(), "")
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

// --- Report Dashboard ---

func (m *Model) initDashboard() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	halfW := w / 2

	// Top row: k6 (left) + graphs (right)
	m.k6Panel = panel.NewModel(m.ctx, "k6 Summary [1]", halfW, panelH)
	m.k6Panel.SetContent(m.renderK6SummaryGrid())

	m.graphsPanel = panel.NewModel(m.ctx, "Graphs [2]", w-halfW, panelH)
	m.reportRPSChart = timechart.NewModel(m.ctx, "Throughput", "req/s", w-halfW-4, panelH/2-2)
	m.reportLatencyChart = timechart.NewModel(m.ctx, "Latency", "ms", w-halfW-4, panelH/2-2)
	m.populateReportCharts()
	m.graphsPanel.SetContent(m.renderGraphsPanelContent())

	// Bottom row: infra (left) + events (right)
	infraW := w * 55 / 100
	eventsW := w - infraW
	m.infraPanel = panel.NewModel(m.ctx, "Infrastructure [3]", infraW, panelH)
	m.infraPanel.SetContent(m.renderInfraTable())

	m.eventsPanel = panel.NewModel(m.ctx, "Scaling Events [4]", eventsW, panelH)
	m.eventsPanel.SetContent(m.renderEventsList())

	m.focusMgr = focus.New(4)
	m.k6Panel.SetFocused(true)

	m.footerComp.SetHints([]footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "tab", Action: "next panel"},
		{Key: "\u2191\u2193", Action: "scroll"},
		{Key: "e", Action: "export JSON"},
		{Key: "o", Action: "open HTML"},
		{Key: "r", Action: "raw view"},
	})
}

func (m *Model) resizeDashboardPanels() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	if w >= constants.BreakpointSplit {
		halfW := w / 2
		m.k6Panel.SetDimensions(halfW, panelH)
		m.graphsPanel.SetDimensions(w-halfW, panelH)

		infraW := w * 55 / 100
		eventsW := w - infraW
		m.infraPanel.SetDimensions(infraW, panelH)
		m.eventsPanel.SetDimensions(eventsW, panelH)

		m.reportRPSChart.Resize(w-halfW-4, panelH/2-2)
		m.reportLatencyChart.Resize(w-halfW-4, panelH/2-2)
		m.graphsPanel.SetContent(m.renderGraphsPanelContent())
	} else {
		m.k6Panel.SetDimensions(w, panelH)
		m.graphsPanel.SetDimensions(w, panelH)
		m.infraPanel.SetDimensions(w, panelH)
		m.eventsPanel.SetDimensions(w, panelH)

		m.reportRPSChart.Resize(w-4, panelH/2-2)
		m.reportLatencyChart.Resize(w-4, panelH/2-2)
		m.graphsPanel.SetContent(m.renderGraphsPanelContent())
	}
}

func (m Model) viewReportDashboard() string {
	if m.rawMode {
		return m.renderReport()
	}

	width := m.ctx.ContentWidth
	verdict := m.renderVerdictBar()

	switch {
	case width >= constants.BreakpointSplit:
		topRow := lipgloss.JoinHorizontal(lipgloss.Top,
			m.k6Panel.View(),
			m.graphsPanel.View(),
		)
		bottomRow := lipgloss.JoinHorizontal(lipgloss.Top,
			m.infraPanel.View(),
			m.eventsPanel.View(),
		)
		return lipgloss.JoinVertical(lipgloss.Left, topRow, bottomRow, verdict)
	case width >= constants.BreakpointStacked:
		return lipgloss.JoinVertical(lipgloss.Left,
			m.k6Panel.View(),
			m.graphsPanel.View(),
			m.infraPanel.View(),
			m.eventsPanel.View(),
			verdict,
		)
	default:
		return m.renderReport()
	}
}

func (m *Model) updateDashboardFocus() tea.Cmd {
	m.k6Panel.SetFocused(m.focusMgr.IsFocused(0))
	m.graphsPanel.SetFocused(m.focusMgr.IsFocused(1))
	m.infraPanel.SetFocused(m.focusMgr.IsFocused(2))
	m.eventsPanel.SetFocused(m.focusMgr.IsFocused(3))
	return tea.Batch(
		m.k6Panel.TransitionCmd(),
		m.graphsPanel.TransitionCmd(),
		m.infraPanel.TransitionCmd(),
		m.eventsPanel.TransitionCmd(),
	)
}

func (m *Model) scrollFocusedPanel(dir int) {
	var p *panel.Model
	switch m.focusMgr.Current() {
	case 0:
		p = &m.k6Panel
	case 1:
		p = &m.graphsPanel
	case 2:
		p = &m.infraPanel
	case 3:
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

func (m Model) renderK6SummaryGrid() string {
	if m.report == nil || m.report.K6 == nil {
		if m.k6Result != nil && m.k6Result.ExitCode != 0 {
			return m.ctx.Styles.Common.FaintTextStyle.Render(
				fmt.Sprintf("k6 exited with code %d \u2014 no summary data available", m.k6Result.ExitCode))
		}
		return m.ctx.Styles.Common.FaintTextStyle.Render("No k6 data available")
	}
	k := m.report.K6
	s := m.ctx.Styles

	type pair struct{ left, right string }
	rows := []pair{
		{
			s.Table.Label.Render("p95 latency") + "  " + fmtFloatMs(k.P95ms),
			s.Table.Label.Render("Throughput") + "  " + fmtFloatRate(k.RPSAvg),
		},
		{
			s.Table.Label.Render("p90 latency") + "  " + fmtFloatMs(k.P90ms),
			s.Table.Label.Render("Error rate") + "  " + fmtPct(k.ErrorRate),
		},
		{
			s.Table.Label.Render("Checks") + "  " + fmtPctRate(k.ChecksRate),
			s.Table.Label.Render("VUs max") + "  " + fmtIntPtr(k.VUsMax),
		},
		{
			s.Table.Label.Render("Total reqs") + "  " + fmtIntPtr(k.TotalRequests),
			s.Table.Label.Render("Thresholds") + "  " + fmt.Sprintf("%d passed, %d failed", k.Thresholds.Passed, k.Thresholds.Failed),
		},
	}

	var b strings.Builder
	colWidth := m.ctx.ContentWidth / 2
	for _, r := range rows {
		left := lipgloss.NewStyle().Width(colWidth).Render(r.left)
		right := lipgloss.NewStyle().Width(colWidth).Render(r.right)
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, right))
		b.WriteString("\n")
	}
	return b.String()
}

func (m Model) renderInfraTable() string {
	s := m.ctx.Styles

	if len(m.metrics) == 0 && m.preSnapshot.TaskRunning == 0 && m.postSnapshot.TaskRunning == 0 {
		return s.Common.FaintTextStyle.Render("Infrastructure metrics pending")
	}

	var lines []string

	// Snapshot deltas
	lines = append(lines,
		fmt.Sprintf("%s  %d -> %d  %s",
			s.Table.Label.Render("ECS tasks"),
			m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning,
			fmtDelta(m.preSnapshot.TaskRunning, m.postSnapshot.TaskRunning)),
		fmt.Sprintf("%s  %d -> %d  %s",
			s.Table.Label.Render("EC2 instances"),
			m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances,
			fmtDelta(m.preSnapshot.ASGInstances, m.postSnapshot.ASGInstances)),
		"",
	)

	// CloudWatch peaks
	for _, mr := range m.metrics {
		label := metricLabel(mr.ID)
		if label == "" {
			continue
		}
		if mr.Peak != nil && mr.Avg != nil {
			lines = append(lines, fmt.Sprintf("%s  peak=%s  avg=%s",
				s.Table.Label.Render(label),
				fmtMetricValue(mr.ID, *mr.Peak),
				fmtMetricValue(mr.ID, *mr.Avg)))
		}
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderEventsList() string {
	var lines []string

	if len(m.activities.ECSScaling) > 0 {
		for _, ev := range m.activities.ECSScaling {
			lines = append(lines, fmt.Sprintf("[%s] %s", ev.Time, ev.Description))
		}
	}
	if len(m.activities.ASGScaling) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		for _, ev := range m.activities.ASGScaling {
			lines = append(lines, fmt.Sprintf("[%s] %s %s", ev.Time, ev.Status, ev.Cause))
		}
	}
	if len(m.activities.Alarms) > 0 {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		for _, a := range m.activities.Alarms {
			lines = append(lines, fmt.Sprintf("[%s] %s: %s -> %s", a.Time, a.AlarmName, a.OldState, a.NewState))
		}
	}

	if len(lines) == 0 {
		return "No scaling events recorded"
	}
	return strings.Join(lines, "\n")
}

func (m *Model) populateReportCharts() {
	for _, mr := range m.metrics {
		switch mr.ID {
		case "alb_requests_per_target":
			if len(mr.Timestamps) > 0 && len(mr.Values) > 0 {
				m.reportRPSChart.SetData(mr.Timestamps, mr.Values)
			}
		case "alb_response_time":
			if len(mr.Timestamps) > 0 && len(mr.Values) > 0 {
				m.reportLatencyChart.SetData(mr.Timestamps, mr.Values)
			}
		}
	}
}

func (m Model) renderGraphsPanelContent() string {
	hasRPS := false
	hasLatency := false
	for _, mr := range m.metrics {
		if mr.ID == "alb_requests_per_target" && len(mr.Values) > 0 {
			hasRPS = true
		}
		if mr.ID == "alb_response_time" && len(mr.Values) > 0 {
			hasLatency = true
		}
	}

	if !hasRPS && !hasLatency {
		return m.ctx.Styles.Common.FaintTextStyle.Render("No metric data for graphs")
	}

	var sections []string
	if hasRPS {
		sections = append(sections, m.reportRPSChart.View())
	}
	if hasLatency {
		if len(sections) > 0 {
			sections = append(sections, "")
		}
		sections = append(sections, m.reportLatencyChart.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderVerdictBar() string {
	s := m.ctx.Styles

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

	var b strings.Builder
	b.WriteString("  " + s.Common.BoldStyle.Render("Verdict: ") + verdictStyle.Render(v.level.String()))
	for _, reason := range v.reasons {
		icon := "\u2713"
		switch v.level {
		case verdictFail:
			icon = "\u2717"
		case verdictWarn:
			icon = "\u26a0"
		}
		b.WriteString("\n  " + icon + " " + reason)
	}
	return b.String()
}

func (m Model) exportReport() tea.Cmd {
	return func() tea.Msg {
		if m.report == nil {
			return errMsg{err: fmt.Errorf("no report to export")}
		}
		path := m.reportPath
		if err := report.WriteReport(m.report, path); err != nil {
			return errMsg{err: err}
		}
		return exportDoneMsg{path: path}
	}
}

func (m Model) openHTML() tea.Cmd {
	return func() tea.Msg {
		htmlPath := filepath.Join(m.app.ResultsDir, m.resultsPrefix+".html")
		if _, err := os.Stat(htmlPath); err != nil {
			return errMsg{err: fmt.Errorf("HTML report not found: %s", htmlPath)}
		}
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", htmlPath)
		default:
			cmd = exec.Command("xdg-open", htmlPath)
		}
		_ = cmd.Start() // fire and forget
		return openDoneMsg{path: htmlPath}
	}
}

func fmtIntPtr(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}

// --- Live Dashboard ---

func (m Model) runK6Streaming(k6Ctx context.Context) tea.Cmd {
	cfg := k6runner.RunConfig{
		TestFile:      m.app.TestFile,
		Env:           m.app.Env,
		ResultsPrefix: m.resultsPrefix,
		ResultsDir:    m.app.ResultsDir,
	}
	if m.baseURL != "" {
		cfg.BaseURL = m.baseURL
	}
	ch := m.k6PointChan
	startTime := m.startTime
	return func() tea.Msg {
		result, err := k6runner.RunStreaming(k6Ctx, cfg, ch)
		if err != nil {
			return errMsg{err: fmt.Errorf("k6 streaming: %w", err)}
		}
		result.StartTime = startTime
		return k6DoneMsg{result: result}
	}
}

func (m Model) waitForK6Point() tea.Cmd {
	ch := m.k6PointChan
	return func() tea.Msg {
		point, ok := <-ch
		if !ok {
			return nil
		}
		return k6PointMsg{point: point}
	}
}

func (m *Model) handleK6Point(point k6runner.K6Point) {
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
		case "ecs_cpu":
			m.cpuGauge.SetValue(latest, 100.0)
			m.cpuTrend.Push(latest)
		case "ecs_memory":
			m.memGauge.SetValue(latest, 100.0)
			m.memTrend.Push(latest)
		case "capacity_provider_reservation":
			m.reservGauge.SetValue(latest, 100.0)
			m.reservTrend.Push(latest)
		}
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

	m.liveGraphPanel.SetContent(m.viewLiveGraphs())
	m.liveInfraPanel.SetContent(m.renderInfraLivePanel())

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

	m.liveGraphPanel.SetContent(m.viewLiveGraphs())
	m.liveGraphPanel.SetDimensions(m.ctx.ContentWidth, m.ctx.ContentHeight/2)

	m.liveInfraPanel.SetContent(m.renderInfraLivePanel())
	m.liveInfraPanel.SetDimensions(m.ctx.ContentWidth, m.ctx.ContentHeight/3)

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
		if mr.ID == "ecs_cpu" && mr.Peak != nil && *mr.Peak >= 90 {
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

// runK6Fallback hands the terminal to k6 via tea.ExecProcess (no live dashboard).
// Used when k6 JSON streaming is not available.
func (m Model) runK6Fallback() tea.Cmd {
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
