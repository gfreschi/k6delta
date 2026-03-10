// Package runtui implements the Bubble Tea model for k6delta run.
package runtui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/verdict"
	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/metriccard"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/statusbar"
	"github.com/gfreschi/k6delta/internal/tui/components/streamchart"
	"github.com/gfreschi/k6delta/internal/tui/components/stepper"
	"github.com/gfreschi/k6delta/internal/tui/components/timechart"
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
	verdictCfg  config.VerdictConfig

	computedVerdict *verdict.Result

	ctx *tuictx.ProgramContext

	currentPhase  runPhase
	stepper       stepper.Model
	headerComp    header.Model
	footerComp    footer.Model
	resultsPrefix string

	preSnapshot  provider.Snapshot
	postSnapshot provider.Snapshot

	k6Result   *k6.RunResult
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
	hasRPSData         bool
	hasLatencyData     bool

	// Cached rendered strings (immutable after initDashboard)
	cachedVitalSigns string
	cachedVerdictTile string
	hasSummaryFile    bool
	hasHTMLFile       bool

	// Live dashboard state
	streamingSupported bool
	liveMode           bool
	graphMode          bool
	k6PointChan        chan k6.K6Point
	k6Cancel           context.CancelFunc
	rpsChart           streamchart.Model
	latencyChart       streamchart.Model
	prevLiveSnapshot   provider.Snapshot
	liveMetrics        []provider.MetricResult
	infraError         error
	liveRPSCount       int
	liveRPSTime        time.Time

	// Live dashboard panels + focus
	liveFocusMgr   *focus.Manager
	liveGraphPanel panel.Model
	liveInfraPanel panel.Model

	// Live infra tiles (replacing gauge + trendline)
	cpuTile    metriccard.Model
	memTile    metriccard.Model
	reservTile metriccard.Model
	tasksTile  metriccard.Model
	asgTile    metriccard.Model

	// Status bar (live mode)
	statusBar statusbar.Model

	// Infrastructure polling backoff
	infraStale       bool
	infraPollBackoff time.Duration

	// Idle detection for live charts
	lastK6DataTime time.Time

	// Phase transition animation (0 = none, 1 = fading, 2 = report shown)
	transitionFrame int

	// Adaptive panel visibility (indices of panels with data)
	visiblePanels []int

	// Health micro-tiles (post-k6 summary)
	healthTiles      []metriccard.Model
	healthTilesReady bool

	// Demo mode
	demoMode     bool
	demoSpeed    float64
	demoScenario string

	// Help overlay
	showHelp bool
}

// NewModel creates a new run Model.
func NewModel(app config.ResolvedApp, prov provider.InfraProvider, baseURL string, skipAnalyze bool, verdictCfg config.VerdictConfig) Model {
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

	prefix := k6.GenerateResultsPrefix(app.Name, app.Phase, app.Env)

	canStream, _ := k6.SupportsJSONStreaming()

	return Model{
		app:                app,
		provider:           prov,
		baseURL:            baseURL,
		skipAnalyze:        skipAnalyze,
		verdictCfg:         verdictCfg,
		ctx:                ctx,
		currentPhase:       phaseInit,
		stepper:            st,
		headerComp:         hdr,
		footerComp:         ftr,
		resultsPrefix:      prefix,
		spinner:            s,
		streamingSupported: canStream,
		graphMode:          true,
		rpsChart:     streamchart.NewModel(ctx, "Throughput", "req/s", ctx.ContentWidth*constants.PanelSplitPct/100, constants.LiveChartHeight),
		latencyChart: streamchart.NewModel(ctx, "Latency", "ms", ctx.ContentWidth*constants.PanelSplitPct/100, constants.LiveChartHeight),
		cpuTile:      metriccard.NewModel(ctx, "CPU", "%", constants.TileWidthWide),
		memTile:      metriccard.NewModel(ctx, "Memory", "%", constants.TileWidthWide),
		reservTile:   metriccard.NewModel(ctx, "Reserv", "%", constants.TileWidthWide),
		tasksTile:    metriccard.NewModel(ctx, "Tasks", "count", constants.TileWidthWide),
		asgTile:      metriccard.NewModel(ctx, "ASG", "count", constants.TileWidthWide),
		statusBar:    statusbar.NewModel(ctx),
	}
}

// NewDemoModel creates a Model for demo mode with fake k6 streaming.
// It uses the mock provider and generates synthetic k6 data points.
func NewDemoModel(app config.ResolvedApp, prov provider.InfraProvider, speed float64, scenario string, verdictCfg config.VerdictConfig) Model {
	m := NewModel(app, prov, "", false, verdictCfg)
	m.demoMode = true
	m.demoSpeed = speed
	m.demoScenario = scenario
	m.streamingSupported = true
	m.stepper = stepper.NewModel(m.ctx,
		"Mock credentials",
		"Pre-snapshot",
		"Running demo",
		"Post-snapshot",
		"Analysis",
		"Report",
	)
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.checkAuth())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.liveMode {
			switch {
			case key.Matches(msg, keys.Keys.NextPanel):
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
			case key.Matches(msg, keys.Keys.PrevPanel):
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
			case key.Matches(msg, keys.LiveKeys.ToggleGraphs):
				m.graphMode = !m.graphMode
				return m, nil
			case key.Matches(msg, keys.LiveKeys.Abort):
				if m.k6Cancel != nil {
					m.k6Cancel()
				}
				return m, nil
			}
		}
		// Help overlay toggle (available in any phase)
		if key.Matches(msg, keys.Keys.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.footerComp.SetState(footer.StateHelp)
			} else {
				m.footerComp.SetState(footer.StateNormal)
			}
			return m, nil
		}
		if m.showHelp {
			if key.Matches(msg, keys.Keys.Escape) {
				m.showHelp = false
				m.footerComp.SetState(footer.StateNormal)
				return m, nil
			}
			return m, nil
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
			case key.Matches(msg, keys.Keys.Jump1):
				if fi := m.visibleFocusIndex(0); fi >= 0 {
					m.focusMgr.SetFocus(fi)
					return m, m.updateDashboardFocus()
				}
			case key.Matches(msg, keys.Keys.Jump2):
				if fi := m.visibleFocusIndex(1); fi >= 0 {
					m.focusMgr.SetFocus(fi)
					return m, m.updateDashboardFocus()
				}
			case key.Matches(msg, keys.Keys.Jump3):
				if fi := m.visibleFocusIndex(2); fi >= 0 {
					m.focusMgr.SetFocus(fi)
					return m, m.updateDashboardFocus()
				}
			case key.Matches(msg, keys.Keys.Jump4):
				if fi := m.visibleFocusIndex(3); fi >= 0 {
					m.focusMgr.SetFocus(fi)
					return m, m.updateDashboardFocus()
				}
			case key.Matches(msg, keys.Keys.Expand):
				cmd := m.cycleExpandFocusedPanel()
				m.footerComp.SetState(footer.StateExpanded)
				return m, cmd
			case key.Matches(msg, keys.Keys.Enter):
				m.expandFocusedPanelFull()
				m.footerComp.SetState(footer.StateExpanded)
				return m, nil
			case key.Matches(msg, keys.Keys.Escape):
				if m.anyPanelExpanded() {
					m.resetAllPanelExpand()
					m.footerComp.SetState(footer.StateNormal)
					return m, nil
				}
			case key.Matches(msg, keys.RunKeys.Export):
				return m, m.exportReport()
			case key.Matches(msg, keys.RunKeys.OpenHTML):
				return m, m.openHTML()
			case key.Matches(msg, keys.RunKeys.RawView):
				m.rawMode = !m.rawMode
				return m, nil
			}
		}
		if key.Matches(msg, keys.Keys.Quit) {
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
			chartW := width*constants.PanelSplitPct/100 - constants.PanelBorderWidth
			chartH := constants.CalcChartHeight(m.ctx.ContentHeight - constants.LayoutOverhead)
			m.rpsChart.Resize(chartW, chartH)
			m.latencyChart.Resize(chartW, chartH)
		case width >= constants.BreakpointStacked:
			chartH := constants.CalcChartHeight(m.ctx.ContentHeight/3 - constants.PanelBorderWidth)
			m.rpsChart.Resize(width-constants.PanelBorderWidth, chartH)
			m.latencyChart.Resize(width-constants.PanelBorderWidth, chartH)
		}
		m.cpuTile.UpdateContext(m.ctx)
		m.memTile.UpdateContext(m.ctx)
		m.reservTile.UpdateContext(m.ctx)
		m.tasksTile.UpdateContext(m.ctx)
		m.asgTile.UpdateContext(m.ctx)
		m.statusBar.UpdateContext(m.ctx)
		if m.liveFocusMgr != nil {
			m.liveGraphPanel.UpdateContext(m.ctx)
			m.liveInfraPanel.UpdateContext(m.ctx)
			m.resizeLivePanels()
			m.updateLivePanelContent()
		}
		if m.focusMgr != nil {
			m.k6Panel.UpdateContext(m.ctx)
			m.graphsPanel.UpdateContext(m.ctx)
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
			m.k6PointChan = make(chan k6.K6Point, 256)
			m.liveMode = true

			m.liveFocusMgr = focus.New(2)
			leftW := m.ctx.ContentWidth * constants.PanelSplitPct / 100
			rightW := m.ctx.ContentWidth - leftW
			panelH := m.ctx.ContentHeight - constants.LayoutOverhead
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
			var k6Cmd tea.Cmd
			if m.demoMode {
				k6Cmd = m.runK6Demo()
			} else {
				k6Cmd = m.runK6Streaming(k6Ctx)
			}
			return m, tea.Batch(
				flashCmd,
				k6Cmd,
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
		m.lastK6DataTime = time.Now()
		m.rpsChart.SetIdle(false)
		m.latencyChart.SetIdle(false)
		m.handleK6Point(msg.point)
		if m.liveFocusMgr != nil {
			m.updateLivePanelContent()
		}
		return m, m.waitForK6Point()

	case infraTickMsg:
		var pulseCmds tea.Cmd
		if msg.err != nil {
			m.infraError = msg.err
			m.infraStale = true
			// Exponential backoff: 20s -> 40s -> 60s (capped)
			if m.infraPollBackoff == 0 {
				m.infraPollBackoff = 20 * time.Second
			} else if m.infraPollBackoff < 60*time.Second {
				m.infraPollBackoff = min(m.infraPollBackoff*2, 60*time.Second)
			}
			m.markInfraTilesStale(true)
		} else {
			m.infraError = nil
			m.infraStale = false
			m.infraPollBackoff = 0
			m.markInfraTilesStale(false)
			// Detect scaling events for chart annotations
			if m.prevLiveSnapshot.TaskRunning != 0 && msg.snapshot.TaskRunning != m.prevLiveSnapshot.TaskRunning {
				label := fmt.Sprintf("scale %d\u2192%d", m.prevLiveSnapshot.TaskRunning, msg.snapshot.TaskRunning)
				ann := streamchart.Annotation{
					Time:  time.Now(),
					Label: label,
					Style: m.ctx.Styles.Timeline.Scaling,
				}
				m.rpsChart.AddAnnotation(ann)
				m.latencyChart.AddAnnotation(ann)
			}
			m.prevLiveSnapshot = msg.snapshot
			m.liveMetrics = msg.metrics
			pulseCmd1 := m.updateTilesFromMetrics(msg.metrics)
			pulseCmd2 := m.updateTilesFromSnapshot(msg.snapshot)
			pulseCmds = tea.Batch(pulseCmd1, pulseCmd2)
		}
		if m.liveMode {
			// Detect idle charts when no k6 data flows for >3s
			if !m.lastK6DataTime.IsZero() && time.Since(m.lastK6DataTime) > 3*time.Second {
				m.rpsChart.SetIdle(true)
				m.latencyChart.SetIdle(true)
			}
			if m.liveFocusMgr != nil {
				m.updateLivePanelContent()
			}
			interval := 15 * time.Second
			if m.infraPollBackoff > 0 {
				interval = m.infraPollBackoff
			}
			return m, tea.Batch(pulseCmds, infraPollCmd(context.Background(), m.provider, interval))
		}
		return m, pulseCmds

	case k6DoneMsg:
		if m.liveRPSCount > 0 && !m.liveRPSTime.IsZero() {
			m.rpsChart.Push(m.liveRPSTime, float64(m.liveRPSCount))
			m.liveRPSCount = 0
		}
		m.liveMode = false
		m.k6Cancel = nil
		m.footerComp.SetHints([]footer.KeyHint{
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
			case "service_cpu":
				ecsCPUPeak = mr.Peak
			}
		}
		v := computeVerdict(verdict.Input{
			K6Exit:      k6Exit,
			ALB5xx:      alb5xx,
			ECSCPUPeak:  ecsCPUPeak,
			TasksBefore: m.preSnapshot.TaskRunning,
			TasksAfter:  m.postSnapshot.TaskRunning,
			Activities:  m.activities,
		}, m.verdictCfg)
		m.computedVerdict = &v
		flashCmd := m.stepper.MarkDone(stepReport, "written")
		// Start transition: frame 1 = fading stepper
		m.transitionFrame = 1
		return m, tea.Batch(flashCmd, tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
			return transitionTickMsg{frame: 2}
		}))

	case transitionTickMsg:
		m.transitionFrame = msg.frame
		if msg.frame == 2 {
			// Switch to report dashboard
			m.currentPhase = phaseDone
			m.initDashboard()
			m.initHealthTiles()
			return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg {
				return transitionTickMsg{frame: 0}
			})
		}
		// frame 0: transition complete
		m.transitionFrame = 0
		return m, nil

	case exportDoneMsg:
		m.footerComp.SetHints([]footer.KeyHint{
			{Key: "q", Action: "quit"},
			{Key: "tab", Action: "panel"},
			{Key: "1-4", Action: "jump"},
			{Key: "+", Action: "expand"},
			{Key: "↑↓", Action: "scroll"},
			{Key: "e", Action: "export"},
			{Key: "o", Action: "open"},
			{Key: "r", Action: "raw"},
			{Key: "?", Action: "help"},
		})
		return m, nil

	case openDoneMsg:
		return m, nil

	case stepper.ClearFlashMsg:
		m.stepper.ClearFlash(msg.Index)
		return m, nil

	case panel.ExpandTransitionTickMsg:
		allPanels := []*panel.Model{&m.k6Panel, &m.graphsPanel, &m.infraPanel, &m.eventsPanel}
		var cmds []tea.Cmd
		for _, p := range allPanels {
			if cmd := p.AdvanceExpandTransition(); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case metriccard.ClearPulseMsg:
		tiles := []*metriccard.Model{&m.cpuTile, &m.memTile, &m.reservTile, &m.tasksTile, &m.asgTile}
		for _, t := range tiles {
			if t.Label() == msg.ID {
				t.ClearPulse()
			}
		}
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
				m.graphsPanel.AdvanceTransition(),
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

	if m.showHelp {
		return m.renderHelpOverlay()
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

	if m.transitionFrame == 1 {
		// Fading: show stepper with "Building report..." in faint
		sections = append(sections, m.ctx.Styles.Common.FaintTextStyle.Render("Building report..."))
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	if m.currentPhase != phaseDone {
		sections = append(sections, m.spinner.View()+" "+m.phaseDescription())
		return lipgloss.JoinVertical(lipgloss.Left, sections...)
	}

	if m.healthTilesReady {
		sections = append(sections, "", m.renderHealthMicroTiles())
	}
	sections = append(sections, m.viewReportDashboard(), "")
	sections = append(sections, m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

