package comparetui

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/report"
)

func TestParsePctChange(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"+5.2%", 5.2},
		{"-3.0%", -3.0},
		{"+0.0%", 0.0},
		{"N/A", 0},
		{"", 0},
		{"-", 0},
		{"+100.5%", 100.5},
		{"-0.1%", -0.1},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parsePctChange(tt.input)
			if got != tt.want {
				t.Errorf("parsePctChange(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsLowerBetter(t *testing.T) {
	tests := []struct {
		metric string
		want   bool
	}{
		{"p95", true},
		{"p90", true},
		{"error_rate", true},
		{"service_cpu_peak", true},
		{"service_memory_peak", true},
		{"alb_5xx", true},
		{"throughput", false},
		{"total_requests", false},
		{"checks_rate", false},
		{"unknown_metric", false},
	}
	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			got := isLowerBetter(tt.metric)
			if got != tt.want {
				t.Errorf("isLowerBetter(%q) = %v, want %v", tt.metric, got, tt.want)
			}
		})
	}
}

func TestComputeSummary(t *testing.T) {
	result := &report.ComparisonResult{
		K6Rows: []report.ComparisonRow{
			{Metric: "p95", Delta: "+12.5%", Direction: "worse"},
			{Metric: "throughput", Delta: "+5.0%", Direction: "better"},
			{Metric: "error_rate", Delta: "+0.1%", Direction: ""},
		},
		InfraRows: []report.ComparisonRow{
			{Metric: "service_cpu_peak", Delta: "+20.0%", Direction: "worse"},
			{Metric: "service_memory_peak", Delta: "-3.0%", Direction: "better"},
		},
	}

	sum := computeSummary(result)

	if sum.improved != 2 {
		t.Errorf("improved = %d, want 2", sum.improved)
	}
	if sum.regressed != 2 {
		t.Errorf("regressed = %d, want 2", sum.regressed)
	}
	if sum.unchanged != 1 {
		t.Errorf("unchanged = %d, want 1", sum.unchanged)
	}
	if sum.worstMetric != "service_cpu_peak" {
		t.Errorf("worstMetric = %q, want %q", sum.worstMetric, "service_cpu_peak")
	}
	if sum.worstPct != 20.0 {
		t.Errorf("worstPct = %v, want 20.0", sum.worstPct)
	}
}

func TestSortedRows(t *testing.T) {
	rows := []report.ComparisonRow{
		{Metric: "p95", Delta: "+5.0%"},
		{Metric: "throughput", Delta: "+20.0%"},
		{Metric: "error_rate", Delta: "-1.0%"},
	}

	m := Model{sort: sortDefault}
	defaultResult := m.sortedRows(rows)
	if defaultResult[0].Metric != "p95" {
		t.Errorf("default sort: first = %q, want %q", defaultResult[0].Metric, "p95")
	}

	m.sort = sortWorstFirst
	worstResult := m.sortedRows(rows)
	if worstResult[0].Metric != "throughput" {
		t.Errorf("worst first: first = %q, want %q", worstResult[0].Metric, "throughput")
	}
	if worstResult[2].Metric != "error_rate" {
		t.Errorf("worst first: last = %q, want %q", worstResult[2].Metric, "error_rate")
	}

	m.sort = sortBestFirst
	bestResult := m.sortedRows(rows)
	if bestResult[0].Metric != "error_rate" {
		t.Errorf("best first: first = %q, want %q", bestResult[0].Metric, "error_rate")
	}
	if bestResult[2].Metric != "throughput" {
		t.Errorf("best first: last = %q, want %q", bestResult[2].Metric, "throughput")
	}

	// Verify original slice not mutated
	if rows[0].Metric != "p95" {
		t.Error("original slice was mutated by sortedRows")
	}
}
