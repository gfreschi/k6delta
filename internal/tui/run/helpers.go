package runtui

import (
	"fmt"
	"strings"
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

func (m Model) renderHealthBar() string {
	s := m.ctx.Styles
	var checks []string

	// CPU check
	cpuOK := true
	for _, mr := range m.liveMetrics {
		if mr.ID == "service_cpu" && mr.Peak != nil && *mr.Peak >= 90 {
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
