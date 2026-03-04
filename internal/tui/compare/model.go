// Package comparetui implements the Bubble Tea model for k6delta compare.
package comparetui

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/keys"
)

type errMsg struct{ err error }
type resultMsg struct{ result *report.ComparisonResult }

type sortMode int

const (
	sortDefault sortMode = iota
	sortWorstFirst
	sortBestFirst
)

func (s sortMode) String() string {
	switch s {
	case sortWorstFirst:
		return "worst first"
	case sortBestFirst:
		return "best first"
	default:
		return "default"
	}
}

// Model is the Bubble Tea model for the compare TUI.
type Model struct {
	pathA  string
	pathB  string
	result *report.ComparisonResult

	ctx        *tuictx.ProgramContext
	headerComp header.Model
	footerComp footer.Model

	k6Panel    panel.Model
	infraPanel panel.Model
	focusMgr   *focus.Manager

	sort     sortMode
	err      error
	quitting bool
}

// NewModel creates a new compare TUI model for the two given report paths.
func NewModel(pathA, pathB string) Model {
	ctx := tuictx.New(80, 24)
	hdr := header.NewModel(ctx, "", "", "compare")
	ftr := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
	})
	return Model{pathA: pathA, pathB: pathB, ctx: ctx, headerComp: hdr, footerComp: ftr}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		result, err := report.CompareReports(m.pathA, m.pathB)
		if err != nil {
			return errMsg{err: err}
		}
		return resultMsg{result: result}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultMsg:
		m.result = msg.result
		m.initDashboard()
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, nil
	case panel.TransitionTickMsg:
		if m.focusMgr != nil {
			cmd := tea.Batch(m.k6Panel.AdvanceTransition(), m.infraPanel.AdvanceTransition())
			return m, cmd
		}
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.CompareKeys.Sort):
			m.sort = (m.sort + 1) % 3
			m.refreshPanels()
			return m, nil
		case m.focusMgr != nil && key.Matches(msg, keys.Keys.NextPanel):
			m.focusMgr.Next()
			return m, m.updatePanelFocus()
		case m.focusMgr != nil && key.Matches(msg, keys.Keys.PrevPanel):
			m.focusMgr.Prev()
			return m, m.updatePanelFocus()
		case m.focusMgr != nil && key.Matches(msg, keys.Keys.Down):
			m.scrollFocusedPanel(1)
			return m, nil
		case m.focusMgr != nil && key.Matches(msg, keys.Keys.Up):
			m.scrollFocusedPanel(-1)
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.ctx.Resize(msg.Width, msg.Height)
		m.headerComp.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
		if m.focusMgr != nil {
			m.k6Panel.UpdateContext(m.ctx)
			m.infraPanel.UpdateContext(m.ctx)
			m.resizePanels()
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	s := m.ctx.Styles

	if m.err != nil {
		return s.Common.ErrorStyle.Render("Error: "+m.err.Error()) + "\n"
	}
	if m.result == nil {
		return "  Loading...\n"
	}

	var sections []string

	// Metadata header
	title := fmt.Sprintf("Compare: %s %s (%s)", m.result.RunA.App, m.result.RunA.Phase, m.result.RunA.Env)
	sections = append(sections, "",
		s.Header.Root.Render(title),
		fmt.Sprintf("  Run A: %s   Run B: %s",
			s.Common.FaintTextStyle.Render(m.result.RunA.Start),
			s.Common.FaintTextStyle.Render(m.result.RunB.Start)),
		"")

	// Panels (responsive: panels at >=80, fallback text at <80)
	width := m.ctx.ContentWidth
	if m.focusMgr != nil && width >= constants.BreakpointStacked {
		sections = append(sections, m.k6Panel.View())
		sections = append(sections, m.infraPanel.View())
	} else if m.focusMgr != nil {
		// Compact fallback: render table content without panel borders
		sections = append(sections, m.renderK6Table(), "", m.renderInfraTable())
	}

	// Summary + footer
	sections = append(sections, "", m.renderSummary(), "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- Dashboard Lifecycle ---

func (m *Model) initDashboard() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	m.k6Panel = panel.NewModel(m.ctx, m.k6PanelTitle(), w, panelH)
	m.k6Panel.SetContent(m.renderK6Table())

	m.infraPanel = panel.NewModel(m.ctx, "Infrastructure [2]", w, panelH)
	m.infraPanel.SetContent(m.renderInfraTable())

	m.focusMgr = focus.New(2)
	m.k6Panel.SetFocused(true)

	m.footerComp.SetHints([]footer.KeyHint{
		{Key: "tab", Action: "next panel"},
		{Key: "↑↓", Action: "scroll"},
		{Key: "s", Action: "sort"},
		{Key: "q", Action: "quit"},
	})
}

func (m *Model) resizePanels() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	m.k6Panel.SetDimensions(w, panelH)
	m.infraPanel.SetDimensions(w, panelH)
}

func (m *Model) refreshPanels() {
	if m.focusMgr == nil {
		return
	}
	m.k6Panel.SetTitle(m.k6PanelTitle())
	m.k6Panel.SetContent(m.renderK6Table())
	m.infraPanel.SetContent(m.renderInfraTable())
}

func (m *Model) updatePanelFocus() tea.Cmd {
	m.k6Panel.SetFocused(m.focusMgr.IsFocused(0))
	m.infraPanel.SetFocused(m.focusMgr.IsFocused(1))
	return tea.Batch(m.k6Panel.TransitionCmd(), m.infraPanel.TransitionCmd())
}

func (m *Model) scrollFocusedPanel(dir int) {
	var p *panel.Model
	switch m.focusMgr.Current() {
	case 0:
		p = &m.k6Panel
	case 1:
		p = &m.infraPanel
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

func (m Model) k6PanelTitle() string {
	if m.sort != sortDefault {
		return fmt.Sprintf("k6 Metrics [1] (sorted: %s)", m.sort)
	}
	return "k6 Metrics [1]"
}

// --- Table Rendering ---

func (m Model) renderK6Table() string {
	k6Sorted := m.sortedRows(m.result.K6Rows)
	var k6Rows []table.Row
	for _, row := range k6Sorted {
		label, valA, valB := formatK6Row(row)
		delta := m.formatDelta(row.Delta, row.Direction, row.Metric)
		k6Rows = append(k6Rows, table.Row{label, valA, valB, delta})
	}
	tbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Metric", Width: 22},
		{Title: "Run A", Width: 12, Align: lipgloss.Right},
		{Title: "Run B", Width: 12, Align: lipgloss.Right},
		{Title: "Delta", Width: 20, Align: lipgloss.Right},
	})
	tbl.SetRows(k6Rows)
	return tbl.View()
}

func (m Model) renderInfraTable() string {
	var cpuRow, memRow, alb5xxRow *report.ComparisonRow
	var tasksBefore, tasksAfter, asgBefore, asgAfter *report.ComparisonRow

	for i := range m.result.InfraRows {
		row := &m.result.InfraRows[i]
		switch row.Metric {
		case "ecs_cpu_peak":
			cpuRow = row
		case "ecs_memory_peak":
			memRow = row
		case "tasks_before":
			tasksBefore = row
		case "tasks_after":
			tasksAfter = row
		case "asg_before":
			asgBefore = row
		case "asg_after":
			asgAfter = row
		case "alb_5xx":
			alb5xxRow = row
		}
	}

	var infraRows []table.Row
	if cpuRow != nil && (cpuRow.ValueA != "-" || cpuRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS CPU peak", cpuRow.ValueA + "%", cpuRow.ValueB + "%", m.formatDelta(cpuRow.Delta, cpuRow.Direction, cpuRow.Metric)})
	}
	if memRow != nil && (memRow.ValueA != "-" || memRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS memory peak", memRow.ValueA + "%", memRow.ValueB + "%", m.formatDelta(memRow.Delta, memRow.Direction, memRow.Metric)})
	}
	if tasksBefore != nil && tasksAfter != nil {
		infraRows = append(infraRows, table.Row{"Tasks (before/after)",
			tasksBefore.ValueA + "/" + tasksAfter.ValueA,
			tasksBefore.ValueB + "/" + tasksAfter.ValueB, ""})
	}
	if asgBefore != nil && asgAfter != nil {
		infraRows = append(infraRows, table.Row{"ASG (before/after)",
			asgBefore.ValueA + "/" + asgAfter.ValueA,
			asgBefore.ValueB + "/" + asgAfter.ValueB, ""})
	}
	if alb5xxRow != nil && (alb5xxRow.ValueA != "-" || alb5xxRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ALB 5xx total", alb5xxRow.ValueA, alb5xxRow.ValueB, m.formatDelta(alb5xxRow.Delta, alb5xxRow.Direction, alb5xxRow.Metric)})
	}

	if len(infraRows) == 0 {
		return "  No infrastructure data"
	}

	tbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Infrastructure", Width: 22},
		{Title: "Run A", Width: 12, Align: lipgloss.Right},
		{Title: "Run B", Width: 12, Align: lipgloss.Right},
		{Title: "Delta", Width: 20, Align: lipgloss.Right},
	})
	tbl.SetRows(infraRows)
	return tbl.View()
}

// --- Sort ---

func (m Model) sortedRows(rows []report.ComparisonRow) []report.ComparisonRow {
	if m.sort == sortDefault {
		return rows
	}
	sorted := make([]report.ComparisonRow, len(rows))
	copy(sorted, rows)
	sort.Slice(sorted, func(i, j int) bool {
		absI := math.Abs(parsePctChange(sorted[i].Delta))
		absJ := math.Abs(parsePctChange(sorted[j].Delta))
		if m.sort == sortWorstFirst {
			return absI > absJ
		}
		return absI < absJ
	})
	return sorted
}

// --- Helpers ---

func formatK6Row(row report.ComparisonRow) (label, valA, valB string) {
	switch row.Metric {
	case "p95":
		return "p95 latency", row.ValueA + "ms", row.ValueB + "ms"
	case "p90":
		return "p90 latency", row.ValueA + "ms", row.ValueB + "ms"
	case "error_rate":
		return "Error rate", row.ValueA, row.ValueB
	case "throughput":
		return "Throughput", row.ValueA + "/s", row.ValueB + "/s"
	case "checks_rate":
		return "Checks rate", row.ValueA, row.ValueB
	case "total_requests":
		return "Total requests", row.ValueA, row.ValueB
	default:
		return row.Metric, row.ValueA, row.ValueB
	}
}

func parsePctChange(delta string) float64 {
	s := strings.TrimSuffix(delta, "%")
	s = strings.TrimPrefix(s, "+")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func isLowerBetter(metric string) bool {
	switch metric {
	case "p95", "p90", "error_rate", "ecs_cpu_peak", "ecs_memory_peak", "alb_5xx":
		return true
	default:
		return false
	}
}

func (m Model) formatDelta(delta, direction, metric string) string {
	if direction == "" {
		return delta
	}

	pct := parsePctChange(delta)
	tiers := m.ctx.Styles.Delta.Tiers()
	style := common.DeltaStyle(tiers, pct, isLowerBetter(metric))

	arrow := ""
	switch {
	case strings.HasPrefix(delta, "+"):
		arrow = " " + constants.IconUp
	case strings.HasPrefix(delta, "-"):
		arrow = " " + constants.IconDown
	}

	// Only show direction word when terminal is wide enough
	dirWord := ""
	if m.ctx.ContentWidth >= constants.BreakpointNarrow {
		switch direction {
		case "better":
			dirWord = " better"
		case "worse":
			dirWord = " worse"
		}
	}

	return style.Render(delta + arrow + dirWord)
}

// --- Summary ---

type comparisonSummary struct {
	improved    int
	regressed   int
	unchanged   int
	worstMetric string
	worstPct    float64
}

func computeSummary(result *report.ComparisonResult) comparisonSummary {
	var sum comparisonSummary
	allRows := make([]report.ComparisonRow, 0, len(result.K6Rows)+len(result.InfraRows))
	allRows = append(allRows, result.K6Rows...)
	allRows = append(allRows, result.InfraRows...)
	for _, row := range allRows {
		switch row.Direction {
		case "better":
			sum.improved++
		case "worse":
			sum.regressed++
			pct := math.Abs(parsePctChange(row.Delta))
			if pct > math.Abs(sum.worstPct) {
				sum.worstPct = parsePctChange(row.Delta)
				sum.worstMetric = row.Metric
			}
		default:
			sum.unchanged++
		}
	}
	return sum
}

func (m Model) renderSummary() string {
	sum := computeSummary(m.result)
	s := m.ctx.Styles

	improved := s.Common.SuccessStyle.Render(fmt.Sprintf("%d improved", sum.improved))
	regressed := s.Common.ErrorStyle.Render(fmt.Sprintf("%d regressed", sum.regressed))
	unchanged := s.Common.FaintTextStyle.Render(fmt.Sprintf("%d unchanged", sum.unchanged))

	line1 := fmt.Sprintf("  %s %s %s %s %s", improved, constants.IconBullet, regressed, constants.IconBullet, unchanged)

	if sum.worstMetric != "" {
		worst := s.Delta.WorseSevere.Render(fmt.Sprintf("%s %+.1f%%", sum.worstMetric, sum.worstPct))
		return line1 + "\n" + fmt.Sprintf("  Worst regression: %s", worst)
	}

	return line1
}

// RunJSON performs the comparison and prints JSON, bypassing the TUI.
func RunJSON(pathA, pathB string) error {
	data, err := report.CompareReportsJSON(pathA, pathB)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
