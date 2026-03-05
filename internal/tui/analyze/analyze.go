// Package analyzetui implements the Bubble Tea model for k6delta analyze.
package analyzetui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

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

	// Help overlay
	showHelp bool
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

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.checkAuth())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, keys.Keys.Help) {
			m.showHelp = !m.showHelp
			return m, nil
		}
		if m.showHelp {
			if key.Matches(msg, keys.Keys.Escape) {
				m.showHelp = false
				return m, nil
			}
			return m, nil
		}
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
			case key.Matches(msg, keys.Keys.Jump1):
				m.focusMgr.SetFocus(0)
				return m, m.updateDashboardFocus()
			case key.Matches(msg, keys.Keys.Jump2):
				m.focusMgr.SetFocus(1)
				return m, m.updateDashboardFocus()
			case key.Matches(msg, keys.Keys.Jump3):
				m.focusMgr.SetFocus(2)
				return m, m.updateDashboardFocus()
			case key.Matches(msg, keys.Keys.Expand):
				m.cycleExpandFocusedPanel()
				return m, nil
			case key.Matches(msg, keys.Keys.Escape):
				if m.anyPanelExpanded() {
					m.resetAllPanelExpand()
					return m, nil
				}
			case key.Matches(msg, keys.AnalyzeKeys.Export):
				if m.outputFile != "" {
					if err := m.writeOutputFile(); err != nil {
						m.err = err
					}
				}
				return m, nil
			}
		}
		if key.Matches(msg, keys.Keys.Quit) {
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

	if m.showHelp {
		return m.renderHelpOverlay()
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
