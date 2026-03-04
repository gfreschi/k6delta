package runtui

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/provider"
)

func f64(v float64) *float64 { return &v }

func TestComputeVerdict_AllPass(t *testing.T) {
	v := computeVerdict(verdictInput{
		k6Exit:      0,
		alb5xx:      0,
		ecsCPUPeak:  nil,
		tasksBefore: 4,
		tasksAfter:  8,
		activities:  provider.Activities{ECSScaling: []provider.ScalingActivity{{}}},
	})
	if v.level != verdictPass {
		t.Errorf("level = %v, want verdictPass", v.level)
	}
}

func TestComputeVerdict_K6Fail(t *testing.T) {
	v := computeVerdict(verdictInput{k6Exit: 1})
	if v.level != verdictFail {
		t.Errorf("level = %v, want verdictFail", v.level)
	}
	found := false
	for _, r := range v.reasons {
		if r == "k6 threshold failures (exit code 1)" {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons = %v, expected k6 threshold failure reason", v.reasons)
	}
}

func TestComputeVerdict_5xxWarn(t *testing.T) {
	v := computeVerdict(verdictInput{alb5xx: 12})
	if v.level != verdictWarn {
		t.Errorf("level = %v, want verdictWarn", v.level)
	}
}

func TestComputeVerdict_HighCPUWarn(t *testing.T) {
	v := computeVerdict(verdictInput{ecsCPUPeak: f64(95.0)})
	if v.level != verdictWarn {
		t.Errorf("level = %v, want verdictWarn", v.level)
	}
}

func TestComputeVerdict_ScalingWithoutEvents(t *testing.T) {
	v := computeVerdict(verdictInput{
		tasksBefore: 4,
		tasksAfter:  8,
		activities:  provider.Activities{},
	})
	// Should have an info note about tasks changing without scaling events
	found := false
	for _, r := range v.reasons {
		if len(r) > 0 && r != "All k6 thresholds passed" && r != "Zero 5xx errors" {
			found = true
		}
	}
	if !found {
		t.Errorf("reasons = %v, expected info note about tasks changed without scaling events", v.reasons)
	}
}

func TestComputeVerdict_PassReasons(t *testing.T) {
	v := computeVerdict(verdictInput{
		k6Exit:      0,
		alb5xx:      0,
		ecsCPUPeak:  f64(45.0),
		tasksBefore: 4,
		tasksAfter:  8,
		activities:  provider.Activities{ECSScaling: []provider.ScalingActivity{{}}},
	})
	if v.level != verdictPass {
		t.Errorf("level = %v, want verdictPass", v.level)
	}
	if len(v.reasons) == 0 {
		t.Error("expected positive reasons for PASS verdict")
	}
}
