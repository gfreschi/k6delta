package runtui

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/verdict"
)

func f64(v float64) *float64 { return &v }

func defaults() config.VerdictConfig {
	return config.DefaultVerdictConfig()
}

func TestComputeVerdict_AllPass(t *testing.T) {
	v := computeVerdict(verdict.Input{
		K6Exit:      0,
		ALB5xx:      0,
		ECSCPUPeak:  nil,
		TasksBefore: 4,
		TasksAfter:  8,
		Activities:  provider.Activities{ServiceScaling: []provider.ScalingActivity{{}}},
	}, defaults())
	if v.Level != verdict.Pass {
		t.Errorf("level = %v, want Pass", v.Level)
	}
}

func TestComputeVerdict_K6Fail(t *testing.T) {
	v := computeVerdict(verdict.Input{K6Exit: 1}, defaults())
	if v.Level != verdict.Fail {
		t.Errorf("level = %v, want Fail", v.Level)
	}
	found := false
	for _, r := range v.Reasons {
		if r == "k6 threshold failures (exit code 1)" {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons = %v, expected k6 threshold failure reason", v.Reasons)
	}
}

func TestComputeVerdict_5xxWarn(t *testing.T) {
	v := computeVerdict(verdict.Input{ALB5xx: 12}, defaults())
	if v.Level != verdict.Fail {
		t.Errorf("level = %v, want Fail (12 >= default fail threshold 10)", v.Level)
	}
}

func TestComputeVerdict_HighCPUWarn(t *testing.T) {
	v := computeVerdict(verdict.Input{ECSCPUPeak: f64(95.0)}, defaults())
	if v.Level != verdict.Warn {
		t.Errorf("level = %v, want Warn", v.Level)
	}
}

func TestComputeVerdict_ScalingWithoutEvents(t *testing.T) {
	v := computeVerdict(verdict.Input{
		TasksBefore: 4,
		TasksAfter:  8,
		Activities:  provider.Activities{},
	}, defaults())
	found := false
	for _, r := range v.Reasons {
		if len(r) > 0 && r != "All k6 thresholds passed" && r != "Zero 5xx errors" {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons = %v, expected info note about tasks changed without scaling events", v.Reasons)
	}
}

func TestComputeVerdict_PassReasons(t *testing.T) {
	v := computeVerdict(verdict.Input{
		K6Exit:      0,
		ALB5xx:      0,
		ECSCPUPeak:  f64(45.0),
		TasksBefore: 4,
		TasksAfter:  8,
		Activities:  provider.Activities{ServiceScaling: []provider.ScalingActivity{{}}},
	}, defaults())
	if v.Level != verdict.Pass {
		t.Errorf("level = %v, want Pass", v.Level)
	}
	if len(v.Reasons) == 0 {
		t.Error("expected positive reasons for PASS verdict")
	}
}
