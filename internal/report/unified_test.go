package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseK6Summary(t *testing.T) {
	fixture := filepath.Join("testdata", "k6-summary.json")
	k6, err := ParseK6Summary(fixture)
	if err != nil {
		t.Fatalf("ParseK6Summary(%q) returned unexpected error: %v", fixture, err)
	}

	assertFloatPtr(t, "P95ms", k6.P95ms, 300.0)
	assertFloatPtr(t, "P90ms", k6.P90ms, 200.0)
	assertFloatPtr(t, "ErrorRate", k6.ErrorRate, 0.01)
	assertFloatPtr(t, "ChecksRate", k6.ChecksRate, 0.99)
	assertIntPtr(t, "TotalRequests", k6.TotalRequests, 1000)
	assertFloatPtr(t, "RPSAvg", k6.RPSAvg, 50.5)
	assertIntPtr(t, "VUsMax", k6.VUsMax, 10)
}

func TestParseK6SummaryThresholds(t *testing.T) {
	fixture := filepath.Join("testdata", "k6-summary.json")
	k6, err := ParseK6Summary(fixture)
	if err != nil {
		t.Fatalf("ParseK6Summary(%q) returned unexpected error: %v", fixture, err)
	}

	if k6.Thresholds.Passed != 1 {
		t.Errorf("Thresholds.Passed = %d, want 1", k6.Thresholds.Passed)
	}
	if k6.Thresholds.Failed != 1 {
		t.Errorf("Thresholds.Failed = %d, want 1", k6.Thresholds.Failed)
	}
}

func TestParseK6SummaryExportFormat(t *testing.T) {
	fixture := filepath.Join("testdata", "k6-summary-export.json")
	k6, err := ParseK6Summary(fixture)
	if err != nil {
		t.Fatalf("ParseK6Summary(%q) returned unexpected error: %v", fixture, err)
	}

	assertFloatPtr(t, "P95ms", k6.P95ms, 350.0)
	assertFloatPtr(t, "P90ms", k6.P90ms, 250.0)
	assertFloatPtr(t, "ErrorRate", k6.ErrorRate, 0.005)
	assertIntPtr(t, "TotalRequests", k6.TotalRequests, 2000)

	if k6.Thresholds.Passed != 2 {
		t.Errorf("Thresholds.Passed = %d, want 2", k6.Thresholds.Passed)
	}
	if k6.Thresholds.Failed != 0 {
		t.Errorf("Thresholds.Failed = %d, want 0", k6.Thresholds.Failed)
	}
}

func TestBuildUnifiedReport(t *testing.T) {
	fixture := filepath.Join("testdata", "k6-summary.json")

	info := RunInfo{
		App:             "app",
		Env:             "staging",
		Phase:           "smoke",
		Start:           "2026-03-03T15:00:00Z",
		End:             "2026-03-03T15:05:00Z",
		K6Exit:          0,
		DurationSeconds: 300,
	}

	report, err := BuildUnifiedReport(info, fixture, nil, 2, 4, 2, 3)
	if err != nil {
		t.Fatalf("BuildUnifiedReport returned unexpected error: %v", err)
	}

	if report.Run.App != "app" {
		t.Errorf("Run.App = %q, want %q", report.Run.App, "app")
	}
	if report.Run.DurationSeconds != 300 {
		t.Errorf("Run.DurationSeconds = %d, want 300", report.Run.DurationSeconds)
	}
	if report.K6 == nil {
		t.Fatal("K6 is nil, want non-nil")
	}
	assertFloatPtr(t, "K6.P95ms", report.K6.P95ms, 300.0)

	if report.Infrastructure == nil {
		t.Fatal("Infrastructure is nil, want non-nil")
	}
	if report.Infrastructure.Tasks.Before != 2 {
		t.Errorf("Infrastructure.Tasks.Before = %d, want 2", report.Infrastructure.Tasks.Before)
	}
	if report.Infrastructure.Tasks.After != 4 {
		t.Errorf("Infrastructure.Tasks.After = %d, want 4", report.Infrastructure.Tasks.After)
	}
	if report.Infrastructure.ASG.Before != 2 {
		t.Errorf("Infrastructure.ASG.Before = %d, want 2", report.Infrastructure.ASG.Before)
	}
	if report.Infrastructure.ASG.After != 3 {
		t.Errorf("Infrastructure.ASG.After = %d, want 3", report.Infrastructure.ASG.After)
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal returned unexpected error: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}
	for _, key := range []string{"run", "k6", "infrastructure", "scaling_activities", "alarm_history"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("serialized report missing top-level key %q", key)
		}
	}
}

func TestWriteReport(t *testing.T) {
	fixture := filepath.Join("testdata", "k6-summary.json")

	info := RunInfo{
		App:             "api-app",
		Env:             "production",
		Phase:           "load",
		Start:           "2026-03-03T16:00:00Z",
		End:             "2026-03-03T16:30:00Z",
		K6Exit:          0,
		DurationSeconds: 1800,
	}

	report, err := BuildUnifiedReport(info, fixture, nil, 1, 2, 1, 1)
	if err != nil {
		t.Fatalf("BuildUnifiedReport returned unexpected error: %v", err)
	}

	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "report.json")

	if err := WriteReport(report, outPath); err != nil {
		t.Fatalf("WriteReport returned unexpected error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) returned unexpected error: %v", outPath, err)
	}

	var loaded UnifiedReport
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("json.Unmarshal returned unexpected error: %v", err)
	}

	if loaded.Run.App != "api-app" {
		t.Errorf("loaded Run.App = %q, want %q", loaded.Run.App, "api-app")
	}
	if loaded.Run.DurationSeconds != 1800 {
		t.Errorf("loaded Run.DurationSeconds = %d, want 1800", loaded.Run.DurationSeconds)
	}
	if loaded.K6 == nil {
		t.Fatal("loaded K6 is nil, want non-nil")
	}
	assertFloatPtr(t, "loaded K6.P95ms", loaded.K6.P95ms, 300.0)
	assertIntPtr(t, "loaded K6.TotalRequests", loaded.K6.TotalRequests, 1000)
}

func assertFloatPtr(t *testing.T, name string, got *float64, want float64) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %v", name, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", name, *got, want)
	}
}

func assertIntPtr(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil {
		t.Errorf("%s is nil, want %v", name, want)
		return
	}
	if *got != want {
		t.Errorf("%s = %v, want %v", name, *got, want)
	}
}
