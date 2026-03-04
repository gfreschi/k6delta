package runtui

import (
	"fmt"

	"github.com/gfreschi/k6delta/internal/provider"
)

type verdictLevel int

const (
	verdictPass verdictLevel = iota
	verdictWarn
	verdictFail
)

func (v verdictLevel) String() string {
	switch v {
	case verdictPass:
		return "PASS"
	case verdictWarn:
		return "WARN"
	case verdictFail:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

type verdict struct {
	level   verdictLevel
	reasons []string
}

type verdictInput struct {
	k6Exit      int
	alb5xx      int
	ecsCPUPeak  *float64
	tasksBefore int
	tasksAfter  int
	activities  provider.Activities
}

func computeVerdict(in verdictInput) verdict {
	var v verdict
	v.level = verdictPass

	// FAIL conditions
	if in.k6Exit != 0 {
		v.level = verdictFail
		v.reasons = append(v.reasons, fmt.Sprintf("k6 threshold failures (exit code %d)", in.k6Exit))
	}

	// WARN conditions
	if in.alb5xx > 0 {
		if v.level < verdictWarn {
			v.level = verdictWarn
		}
		v.reasons = append(v.reasons, fmt.Sprintf("%d 5xx errors detected", in.alb5xx))
	}

	if in.ecsCPUPeak != nil && *in.ecsCPUPeak > 90.0 {
		if v.level < verdictWarn {
			v.level = verdictWarn
		}
		v.reasons = append(v.reasons, fmt.Sprintf("ECS CPU peaked at %.1f%% (>90%%)", *in.ecsCPUPeak))
	}

	// INFO notes
	tasksChanged := in.tasksAfter != in.tasksBefore
	hasScalingEvents := len(in.activities.ECSScaling) > 0 || len(in.activities.ASGScaling) > 0
	if tasksChanged && !hasScalingEvents {
		v.reasons = append(v.reasons, fmt.Sprintf("tasks changed (%d→%d) but no scaling events recorded", in.tasksBefore, in.tasksAfter))
	}

	// Positive reasons for PASS
	if v.level == verdictPass {
		if in.k6Exit == 0 {
			v.reasons = append(v.reasons, "All k6 thresholds passed")
		}
		if in.alb5xx == 0 {
			v.reasons = append(v.reasons, "Zero 5xx errors")
		}
		if tasksChanged && hasScalingEvents {
			v.reasons = append(v.reasons, fmt.Sprintf("Autoscaling responded (%d→%d tasks)", in.tasksBefore, in.tasksAfter))
		}
	}

	return v
}
