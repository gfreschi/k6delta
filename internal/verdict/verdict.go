// Package verdict computes pass/warn/fail verdicts from load test results.
// Shared by both TUI and CI codepaths.
package verdict

import (
	"fmt"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
)

// Level represents the severity of a verdict.
type Level int

const (
	Pass Level = iota
	Warn
	Fail
)

// String returns the verdict level as a string.
func (l Level) String() string {
	switch l {
	case Pass:
		return "PASS"
	case Warn:
		return "WARN"
	case Fail:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

// Result holds the computed verdict and its reasons.
type Result struct {
	Level   Level
	Reasons []string
}

// ExitCode returns the Unix-convention exit code: 0 for PASS/WARN, 1 for FAIL.
func (r Result) ExitCode() int {
	if r.Level == Fail {
		return 1
	}
	return 0
}

// Input holds the data needed to compute a verdict.
type Input struct {
	K6Exit      int
	ALB5xx      int
	ECSCPUPeak  *float64
	TasksBefore int
	TasksAfter  int
	Activities  provider.Activities
}

// Compute calculates a verdict from the input and config thresholds.
func Compute(in Input, cfg config.VerdictConfig) Result {
	v := Result{Level: Pass}

	// FAIL: k6 exit code != 0
	if in.K6Exit != 0 {
		v.Level = Fail
		v.Reasons = append(v.Reasons, fmt.Sprintf("k6 threshold failures (exit code %d)", in.K6Exit))
	}

	// WARN/FAIL: 5xx errors above threshold
	if in.ALB5xx >= cfg.Error5xxFail {
		v.Level = Fail
		v.Reasons = append(v.Reasons, fmt.Sprintf("%d 5xx errors detected (threshold: %d)", in.ALB5xx, cfg.Error5xxFail))
	} else if in.ALB5xx >= cfg.Error5xxWarn {
		if v.Level < Warn {
			v.Level = Warn
		}
		v.Reasons = append(v.Reasons, fmt.Sprintf("%d 5xx errors detected (threshold: %d)", in.ALB5xx, cfg.Error5xxWarn))
	}

	// WARN/FAIL: CPU peak above threshold
	if in.ECSCPUPeak != nil {
		if *in.ECSCPUPeak > cfg.CPUPeakFail {
			v.Level = Fail
			v.Reasons = append(v.Reasons, fmt.Sprintf("ECS CPU peaked at %.1f%% (threshold: %.0f%%)", *in.ECSCPUPeak, cfg.CPUPeakFail))
		} else if *in.ECSCPUPeak > cfg.CPUPeakWarn {
			if v.Level < Warn {
				v.Level = Warn
			}
			v.Reasons = append(v.Reasons, fmt.Sprintf("ECS CPU peaked at %.1f%% (threshold: %.0f%%)", *in.ECSCPUPeak, cfg.CPUPeakWarn))
		}
	}

	// INFO: tasks changed without scaling events
	tasksChanged := in.TasksBefore != in.TasksAfter
	hasScalingEvents := len(in.Activities.ServiceScaling) > 0 || len(in.Activities.NodeScaling) > 0
	if tasksChanged && !hasScalingEvents {
		v.Reasons = append(v.Reasons, fmt.Sprintf("tasks changed (%d→%d) but no scaling events recorded", in.TasksBefore, in.TasksAfter))
	}

	// Positive reasons for PASS
	if v.Level == Pass {
		if in.K6Exit == 0 {
			v.Reasons = append(v.Reasons, "All k6 thresholds passed")
		}
		if in.ALB5xx == 0 {
			v.Reasons = append(v.Reasons, "Zero 5xx errors")
		}
		if tasksChanged && hasScalingEvents {
			v.Reasons = append(v.Reasons, fmt.Sprintf("Autoscaling responded (%d→%d tasks)", in.TasksBefore, in.TasksAfter))
		}
	}

	return v
}
