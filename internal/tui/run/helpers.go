package runtui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	case "service_cpu":
		return "ECS CPU"
	case "service_memory":
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

func fmtIntPtr(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}

func (m Model) renderHelpOverlay() string {
	s := m.ctx.Styles
	w := m.ctx.ContentWidth
	h := m.ctx.ContentHeight

	groups := []struct {
		title string
		keys  [][2]string
	}{
		{"Navigation", [][2]string{
			{"q", "Quit"},
			{"?", "Toggle help"},
			{"esc", "Close help / collapse panel"},
		}},
		{"Panels", [][2]string{
			{"tab / shift+tab", "Next / previous panel"},
			{"1-4", "Jump to panel"},
			{"+", "Cycle expand (normal → expanded → full)"},
			{"↑↓ / j k", "Scroll focused panel"},
		}},
		{"Actions", [][2]string{
			{"e", "Export JSON report"},
			{"o", "Open HTML report"},
			{"r", "Toggle raw view"},
		}},
	}
	if m.liveMode {
		groups = append(groups, struct {
			title string
			keys  [][2]string
		}{"Live Mode", [][2]string{
			{"g", "Toggle graphs"},
			{"a", "Abort k6"},
		}})
	}

	var lines []string
	lines = append(lines, s.Header.Root.Render("Keyboard Shortcuts"), "")
	for _, g := range groups {
		lines = append(lines, s.Common.BoldStyle.Render("  "+g.title))
		for _, kv := range g.keys {
			lines = append(lines, fmt.Sprintf("    %-22s %s", s.Footer.Key.Render(kv[0]), kv[1]))
		}
		lines = append(lines, "")
	}
	lines = append(lines, s.Common.FaintTextStyle.Render("  Press ? or esc to close"))

	content := strings.Join(lines, "\n")
	overlay := lipgloss.NewStyle().
		Width(min(w-4, 60)).
		Height(min(h-2, len(lines)+2)).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Panel.Focused.GetBorderTopForeground()).
		Padding(1, 2).
		Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, overlay)
}

