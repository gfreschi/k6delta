package runtui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider/mock"
	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/golden"
	"github.com/gfreschi/k6delta/internal/tui/testutil"
)

func TestRunModel_reportDashboard(t *testing.T) {
	prov, err := mock.New("happy-path")
	if err != nil {
		t.Fatal(err)
	}

	app := testutil.ResolvedApp()
	vcfg := testutil.VerdictConfig()
	start := testutil.ReferenceTime
	end := start.Add(60 * time.Second)

	m := NewModel(app, prov, "", false, vcfg)

	// Set terminal size
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Populate the model with deterministic data to reach phaseDone
	// without running through the full state machine.
	rm := model.(Model)
	rm.currentPhase = phaseDone
	rm.startTime = start
	rm.endTime = end
	rm.preSnapshot = testutil.SampleSnapshot()
	rm.postSnapshot = testutil.SampleSnapshot()
	rm.metrics = testutil.SampleMetrics()
	rm.activities = testutil.SampleActivities()
	rm.k6Result = &k6.RunResult{ExitCode: 0, StartTime: start, EndTime: end}

	p95 := 150.0
	p90 := 120.0
	errRate := 0.005
	checksRate := 0.998
	totalReqs := 5000
	rpsAvg := 83.3
	vusMax := 50
	rm.report = &report.UnifiedReport{
		Run: report.RunInfo{
			App:             "test-app",
			Env:             "staging",
			Phase:           "smoke",
			Start:           start.Format(time.RFC3339),
			End:             end.Format(time.RFC3339),
			K6Exit:          0,
			DurationSeconds: 60,
		},
		K6: &report.K6Metrics{
			P95ms:         &p95,
			P90ms:         &p90,
			ErrorRate:     &errRate,
			ChecksRate:    &checksRate,
			TotalRequests: &totalReqs,
			RPSAvg:        &rpsAvg,
			VUsMax:        &vusMax,
			Thresholds:    report.ThresholdSummary{Passed: 3, Failed: 0},
		},
		Infrastructure: &report.InfraMetrics{
			ServiceCPU:    &report.PeakAvg{Peak: testutil.Float64Ptr(55.0), Avg: testutil.Float64Ptr(43.6)},
			ServiceMemory: &report.PeakAvg{Peak: testutil.Float64Ptr(530.0), Avg: testutil.Float64Ptr(521.0)},
			Tasks:         report.BeforeAfter{Before: 4, After: 4},
			ASG:           report.BeforeAfter{Before: 2, After: 2},
		},
	}

	rm.initDashboard()

	out := rm.View()
	golden.RequireEqual(t, []byte(out))
}
