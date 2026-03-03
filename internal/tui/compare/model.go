// Package comparetui implements the Bubble Tea model for k6delta compare.
package comparetui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui"
)

type errMsg struct{ err error }
type resultMsg struct{ result *report.ComparisonResult }

// Model is the Bubble Tea model for the compare TUI.
type Model struct {
	pathA    string
	pathB    string
	result   *report.ComparisonResult
	err      error
	quitting bool
	width    int
	height   int
}

// NewModel creates a new compare TUI model for the two given report paths.
func NewModel(pathA, pathB string) Model {
	return Model{pathA: pathA, pathB: pathB}
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
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}
	if m.err != nil {
		return tui.ErrorStyle.Render("Error: "+m.err.Error()) + "\n"
	}
	if m.result == nil {
		return "  Loading...\n"
	}

	var b strings.Builder

	title := fmt.Sprintf("Compare: %s %s (%s)", m.result.RunA.App, m.result.RunA.Phase, m.result.RunA.Env)
	b.WriteString("\n")
	b.WriteString(tui.HeaderStyle.Render(title))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  Run A: %s   Run B: %s",
		tui.DimStyle.Render(m.result.RunA.Start),
		tui.DimStyle.Render(m.result.RunB.Start)))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  %-22s %12s %12s %12s\n", "Metric", "Run A", "Run B", "Delta"))
	b.WriteString(fmt.Sprintf("  %-22s %12s %12s %12s\n",
		strings.Repeat("\u2500", 22),
		strings.Repeat("\u2500", 12),
		strings.Repeat("\u2500", 12),
		strings.Repeat("\u2500", 12)))

	for _, row := range m.result.K6Rows {
		label, valA, valB := formatK6Row(row)
		delta := formatDelta(row.Delta, row.Direction)
		b.WriteString(fmt.Sprintf("  %-22s %12s %12s %s\n", label, valA, valB, delta))
	}

	b.WriteString("\n")

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

	if cpuRow != nil && (cpuRow.ValueA != "-" || cpuRow.ValueB != "-") {
		delta := formatDelta(cpuRow.Delta, cpuRow.Direction)
		b.WriteString(fmt.Sprintf("  %-22s %11s%% %11s%% %s\n", "ECS CPU peak", cpuRow.ValueA, cpuRow.ValueB, delta))
	}
	if memRow != nil && (memRow.ValueA != "-" || memRow.ValueB != "-") {
		delta := formatDelta(memRow.Delta, memRow.Direction)
		b.WriteString(fmt.Sprintf("  %-22s %11s%% %11s%% %s\n", "ECS memory peak", memRow.ValueA, memRow.ValueB, delta))
	}
	if tasksBefore != nil && tasksAfter != nil {
		b.WriteString(fmt.Sprintf("  %-22s %8s/%-3s %8s/%-3s\n", "Tasks (before/after)",
			tasksBefore.ValueA, tasksAfter.ValueA,
			tasksBefore.ValueB, tasksAfter.ValueB))
	}
	if asgBefore != nil && asgAfter != nil {
		b.WriteString(fmt.Sprintf("  %-22s %8s/%-3s %8s/%-3s\n", "ASG (before/after)",
			asgBefore.ValueA, asgAfter.ValueA,
			asgBefore.ValueB, asgAfter.ValueB))
	}
	if alb5xxRow != nil && (alb5xxRow.ValueA != "-" || alb5xxRow.ValueB != "-") {
		delta := formatDelta(alb5xxRow.Delta, alb5xxRow.Direction)
		b.WriteString(fmt.Sprintf("  %-22s %12s %12s %s\n", "ALB 5xx total", alb5xxRow.ValueA, alb5xxRow.ValueB, delta))
	}

	b.WriteString("\n")
	b.WriteString(tui.DimStyle.Render("  Press q to quit"))
	b.WriteString("\n")

	return b.String()
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

func formatDelta(delta, direction string) string {
	if direction == "" {
		return fmt.Sprintf("%12s", delta)
	}

	var arrow string
	var style func(strs ...string) string

	switch {
	case direction == "better" && strings.HasPrefix(delta, "-"):
		arrow = " \u2193"
		style = tui.SuccessStyle.Render
	case direction == "better" && strings.HasPrefix(delta, "+"):
		arrow = " \u2191"
		style = tui.SuccessStyle.Render
	case direction == "worse" && strings.HasPrefix(delta, "+"):
		arrow = " \u2191"
		style = tui.WarnStyle.Render
	case direction == "worse" && strings.HasPrefix(delta, "-"):
		arrow = " \u2193"
		style = tui.WarnStyle.Render
	default:
		return fmt.Sprintf("%12s", delta)
	}

	return fmt.Sprintf("%12s %s", delta, style(arrow))
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
