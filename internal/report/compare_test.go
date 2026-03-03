package report

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestComputeDelta(t *testing.T) {
	tests := []struct {
		name string
		a, b float64
		want string
	}{
		{"increase 50%", 100, 150, "+50.0%"},
		{"same values", 100, 100, "same"},
		{"both zero", 0, 0, "same"},
		{"a zero b nonzero", 0, 50, "new"},
		{"b is zero", 100, 0, "-100.0%"},
		{"decrease 50%", 200, 100, "-50.0%"},
		{"small decrease", 500, 350, "-30.0%"},
		{"small increase", 50, 60, "+20.0%"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeDelta(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("ComputeDelta(%v, %v) = %q, want %q", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestDirection(t *testing.T) {
	tests := []struct {
		name, delta, metricType, want string
	}{
		{"positive lower_better is worse", "+50.0%", "lower_better", "worse"},
		{"negative lower_better is better", "-30.0%", "lower_better", "better"},
		{"positive higher_better is better", "+20.0%", "higher_better", "better"},
		{"negative higher_better is worse", "-10.0%", "higher_better", "worse"},
		{"same returns empty", "same", "lower_better", ""},
		{"N/A returns empty", "N/A", "lower_better", ""},
		{"new returns empty", "new", "lower_better", ""},
		{"empty delta returns empty", "", "lower_better", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Direction(tt.delta, tt.metricType)
			if got != tt.want {
				t.Errorf("Direction(%q, %q) = %q, want %q", tt.delta, tt.metricType, got, tt.want)
			}
		})
	}
}

func TestCompareReports(t *testing.T) {
	pathA := filepath.Join("testdata", "report-a.json")
	pathB := filepath.Join("testdata", "report-b.json")

	result, err := CompareReports(pathA, pathB)
	if err != nil {
		t.Fatalf("CompareReports: %v", err)
	}

	if result.RunA.Start != "2026-03-02T10:00:00Z" {
		t.Errorf("RunA.Start = %q, want %q", result.RunA.Start, "2026-03-02T10:00:00Z")
	}

	findK6 := func(metric string) *ComparisonRow {
		for i := range result.K6Rows {
			if result.K6Rows[i].Metric == metric {
				return &result.K6Rows[i]
			}
		}
		return nil
	}
	findInfra := func(metric string) *ComparisonRow {
		for i := range result.InfraRows {
			if result.InfraRows[i].Metric == metric {
				return &result.InfraRows[i]
			}
		}
		return nil
	}

	p95 := findK6("p95")
	if p95 == nil {
		t.Fatal("p95 row not found")
	}
	if p95.Delta != "-30.0%" {
		t.Errorf("p95 delta = %q, want %q", p95.Delta, "-30.0%")
	}
	if p95.Direction != "better" {
		t.Errorf("p95 direction = %q, want %q", p95.Direction, "better")
	}

	throughput := findK6("throughput")
	if throughput == nil {
		t.Fatal("throughput row not found")
	}
	if throughput.Delta != "+20.0%" {
		t.Errorf("throughput delta = %q, want %q", throughput.Delta, "+20.0%")
	}

	alb5xx := findInfra("alb_5xx")
	if alb5xx == nil {
		t.Fatal("alb_5xx row not found")
	}
	if alb5xx.Delta != "new" {
		t.Errorf("alb_5xx delta = %q, want %q", alb5xx.Delta, "new")
	}

	if len(result.K6Rows) != 6 {
		t.Errorf("K6Rows count = %d, want 6", len(result.K6Rows))
	}
	if len(result.InfraRows) != 7 {
		t.Errorf("InfraRows count = %d, want 7", len(result.InfraRows))
	}
}

func TestCompareReportsJSON(t *testing.T) {
	pathA := filepath.Join("testdata", "report-a.json")
	pathB := filepath.Join("testdata", "report-b.json")

	data, err := CompareReportsJSON(pathA, pathB)
	if err != nil {
		t.Fatalf("CompareReportsJSON: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("JSON unmarshal: %v", err)
	}
	for _, key := range []string{"run_a", "run_b", "comparison"} {
		if _, ok := out[key]; !ok {
			t.Errorf("missing top-level key %q", key)
		}
	}
}

func TestCompareReports_FileNotFound(t *testing.T) {
	_, err := CompareReports("/nonexistent/a.json", "/nonexistent/b.json")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}
