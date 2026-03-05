// Package testutil provides deterministic test data for TUI model tests.
package testutil

import (
	"time"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
)

// ReferenceTime is the fixed time used in all golden file tests.
var ReferenceTime = time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

// ResolvedApp returns a deterministic ResolvedApp for tests.
func ResolvedApp() config.ResolvedApp {
	return config.ResolvedApp{
		Name:       "test-app",
		Env:        "staging",
		Phase:      "smoke",
		Region:     "us-east-1",
		ResultsDir: "",
		TestFile:   "tests/smoke.js",
		Cluster:    "test-cluster",
		Service:    "test-service",
	}
}

// VerdictConfig returns default verdict config.
func VerdictConfig() config.VerdictConfig {
	return config.DefaultVerdictConfig()
}

// Float64Ptr returns a pointer to a float64.
func Float64Ptr(v float64) *float64 {
	return &v
}

// SampleMetrics returns deterministic metrics for golden tests.
func SampleMetrics() []provider.MetricResult {
	return []provider.MetricResult{
		{
			ID:     "service_cpu",
			Values: []float64{35.0, 42.0, 55.0, 48.0, 38.0},
			Peak:   Float64Ptr(55.0),
			Avg:    Float64Ptr(43.6),
			Timestamps: []time.Time{
				ReferenceTime,
				ReferenceTime.Add(10 * time.Second),
				ReferenceTime.Add(20 * time.Second),
				ReferenceTime.Add(30 * time.Second),
				ReferenceTime.Add(40 * time.Second),
			},
		},
		{
			ID:     "service_memory",
			Values: []float64{512.0, 520.0, 530.0, 525.0, 518.0},
			Peak:   Float64Ptr(530.0),
			Avg:    Float64Ptr(521.0),
			Timestamps: []time.Time{
				ReferenceTime,
				ReferenceTime.Add(10 * time.Second),
				ReferenceTime.Add(20 * time.Second),
				ReferenceTime.Add(30 * time.Second),
				ReferenceTime.Add(40 * time.Second),
			},
		},
	}
}

// SampleActivities returns deterministic activities for golden tests.
func SampleActivities() provider.Activities {
	return provider.Activities{
		ServiceScaling: []provider.ScalingActivity{
			{
				Time:        ReferenceTime.Add(15 * time.Second).Format(time.RFC3339),
				Status:      "Successful",
				Description: "ECS service scaled from 2 to 4 tasks",
			},
		},
	}
}

// SampleSnapshot returns a deterministic snapshot.
func SampleSnapshot() provider.Snapshot {
	return provider.Snapshot{
		TaskRunning:  4,
		TaskDesired:  4,
		ASGName:      "test-asg",
		ASGInstances: 2,
		ASGDesired:   2,
	}
}

// SampleReportA returns a deterministic report for compare tests.
func SampleReportA() *report.UnifiedReport {
	return &report.UnifiedReport{
		Run: report.RunInfo{
			App:   "test-app",
			Env:   "staging",
			Phase: "smoke",
			Start: ReferenceTime.Format(time.RFC3339),
			End:   ReferenceTime.Add(60 * time.Second).Format(time.RFC3339),
		},
	}
}
