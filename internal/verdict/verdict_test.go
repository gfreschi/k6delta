package verdict_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/verdict"
)

func ptrFloat(f float64) *float64 {
	return &f
}

func defaults() config.VerdictConfig {
	return config.DefaultVerdictConfig()
}

func TestCompute_failOnK6Exit(t *testing.T) {
	in := verdict.Input{K6Exit: 1}
	v := verdict.Compute(in, defaults())
	if v.Level != verdict.Fail {
		t.Errorf("level = %v, want Fail", v.Level)
	}
}

func TestCompute_warnOnCPU(t *testing.T) {
	in := verdict.Input{
		K6Exit:     0,
		ECSCPUPeak: ptrFloat(92.0),
	}
	v := verdict.Compute(in, defaults())
	if v.Level != verdict.Warn {
		t.Errorf("level = %v, want Warn", v.Level)
	}
}

func TestCompute_passClean(t *testing.T) {
	in := verdict.Input{K6Exit: 0}
	v := verdict.Compute(in, defaults())
	if v.Level != verdict.Pass {
		t.Errorf("level = %v, want Pass", v.Level)
	}
}

func TestCompute_configurableCPUThreshold(t *testing.T) {
	in := verdict.Input{
		K6Exit:     0,
		ECSCPUPeak: ptrFloat(85.0),
	}
	// Custom threshold: warn at 80%
	cfg := defaults()
	cfg.CPUPeakWarn = 80.0
	v := verdict.Compute(in, cfg)
	if v.Level != verdict.Warn {
		t.Errorf("level = %v, want Warn (CPU 85%% > threshold 80%%)", v.Level)
	}

	// Default threshold: 90%, so 85% passes
	v2 := verdict.Compute(in, defaults())
	if v2.Level != verdict.Pass {
		t.Errorf("level = %v, want Pass (CPU 85%% < default 90%%)", v2.Level)
	}
}

func TestCompute_configurable5xxThreshold(t *testing.T) {
	in := verdict.Input{
		K6Exit: 0,
		ALB5xx: 3,
	}
	// Custom: warn at 5 → 3 < 5 → pass
	cfg := defaults()
	cfg.Error5xxWarn = 5
	v := verdict.Compute(in, cfg)
	if v.Level != verdict.Pass {
		t.Errorf("level = %v, want Pass (3 < threshold 5)", v.Level)
	}

	// Default: warn at 1 → 3 >= 1 → warn
	v2 := verdict.Compute(in, defaults())
	if v2.Level != verdict.Warn {
		t.Errorf("level = %v, want Warn (3 >= default 1)", v2.Level)
	}
}

func TestCompute_allPassWithScaling(t *testing.T) {
	in := verdict.Input{
		K6Exit:      0,
		ALB5xx:      0,
		ECSCPUPeak:  ptrFloat(45.0),
		TasksBefore: 4,
		TasksAfter:  8,
		Activities:  provider.Activities{ServiceScaling: []provider.ScalingActivity{{}}},
	}
	v := verdict.Compute(in, defaults())
	if v.Level != verdict.Pass {
		t.Errorf("level = %v, want Pass", v.Level)
	}
	if len(v.Reasons) == 0 {
		t.Error("expected positive reasons for PASS verdict")
	}
}

func TestCompute_scalingWithoutEvents(t *testing.T) {
	in := verdict.Input{
		TasksBefore: 4,
		TasksAfter:  8,
		Activities:  provider.Activities{},
	}
	v := verdict.Compute(in, defaults())
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
