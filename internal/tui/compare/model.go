// Package comparetui implements the Bubble Tea model for k6delta compare.
package comparetui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/table"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

type errMsg struct{ err error }
type resultMsg struct{ result *report.ComparisonResult }

// Model is the Bubble Tea model for the compare TUI.
type Model struct {
	pathA    string
	pathB    string
	result   *report.ComparisonResult

	ctx        *tuictx.ProgramContext
	headerComp header.Model
	footerComp footer.Model

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
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.quitting = true
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.ctx.Resize(msg.Width, msg.Height)
		m.headerComp.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
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

	// Update header with actual data now available
	var sections []string

	title := fmt.Sprintf("Compare: %s %s (%s)", m.result.RunA.App, m.result.RunA.Phase, m.result.RunA.Env)
	sections = append(sections, "",
		s.Header.Root.Render(title),
		fmt.Sprintf("  Run A: %s   Run B: %s",
			s.Common.FaintTextStyle.Render(m.result.RunA.Start),
			s.Common.FaintTextStyle.Render(m.result.RunB.Start)),
		"")

	// k6 metrics table
	var k6Rows []table.Row
	for _, row := range m.result.K6Rows {
		label, valA, valB := formatK6Row(row)
		delta := m.formatDelta(row.Delta, row.Direction)
		k6Rows = append(k6Rows, table.Row{label, valA, valB, delta})
	}
	k6Tbl := table.NewModel(m.ctx, []table.Column{
		{Title: "Metric", Width: 22},
		{Title: "Run A", Width: 12, Align: lipgloss.Right},
		{Title: "Run B", Width: 12, Align: lipgloss.Right},
		{Title: "Delta", Width: 14, Align: lipgloss.Right},
	})
	k6Tbl.SetRows(k6Rows)
	sections = append(sections, k6Tbl.View(), "")

	// Infrastructure rows
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
		infraRows = append(infraRows, table.Row{"ECS CPU peak", cpuRow.ValueA + "%", cpuRow.ValueB + "%", m.formatDelta(cpuRow.Delta, cpuRow.Direction)})
	}
	if memRow != nil && (memRow.ValueA != "-" || memRow.ValueB != "-") {
		infraRows = append(infraRows, table.Row{"ECS memory peak", memRow.ValueA + "%", memRow.ValueB + "%", m.formatDelta(memRow.Delta, memRow.Direction)})
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
		infraRows = append(infraRows, table.Row{"ALB 5xx total", alb5xxRow.ValueA, alb5xxRow.ValueB, m.formatDelta(alb5xxRow.Delta, alb5xxRow.Direction)})
	}

	if len(infraRows) > 0 {
		infraTbl := table.NewModel(m.ctx, []table.Column{
			{Title: "Infrastructure", Width: 22},
			{Title: "Run A", Width: 12, Align: lipgloss.Right},
			{Title: "Run B", Width: 12, Align: lipgloss.Right},
			{Title: "Delta", Width: 14, Align: lipgloss.Right},
		})
		infraTbl.SetRows(infraRows)
		sections = append(sections, infraTbl.View())
	}

	sections = append(sections, "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

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

func (m Model) formatDelta(delta, direction string) string {
	if direction == "" {
		return delta
	}

	ds := m.ctx.Styles.Delta
	var arrow string
	var style lipgloss.Style

	switch {
	case direction == "better" && strings.HasPrefix(delta, "-"):
		arrow = " \u2193"
		style = ds.Better
	case direction == "better" && strings.HasPrefix(delta, "+"):
		arrow = " \u2191"
		style = ds.Better
	case direction == "worse" && strings.HasPrefix(delta, "+"):
		arrow = " \u2191"
		style = ds.Worse
	case direction == "worse" && strings.HasPrefix(delta, "-"):
		arrow = " \u2193"
		style = ds.Worse
	default:
		return delta
	}

	return delta + style.Render(arrow)
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
