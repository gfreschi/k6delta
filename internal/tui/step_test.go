package tui

import (
	"strings"
	"testing"
	"time"
)

func TestMarkRunning_SetsStartedAt(t *testing.T) {
	st := NewStepTracker("Auth", "Snapshot")
	before := time.Now()
	st.MarkRunning(0)
	after := time.Now()

	if st.Steps[0].Status != StepRunning {
		t.Fatalf("Status = %v, want StepRunning", st.Steps[0].Status)
	}
	if st.Steps[0].StartedAt.Before(before) || st.Steps[0].StartedAt.After(after) {
		t.Errorf("StartedAt not within expected range")
	}
}

func TestMarkDone_ComputesDuration(t *testing.T) {
	st := NewStepTracker("Auth")
	st.MarkRunning(0)
	time.Sleep(10 * time.Millisecond)
	st.MarkDone(0, "verified")

	if st.Steps[0].Status != StepDone {
		t.Fatalf("Status = %v, want StepDone", st.Steps[0].Status)
	}
	if st.Steps[0].Duration < 10*time.Millisecond {
		t.Errorf("Duration = %v, want >= 10ms", st.Steps[0].Duration)
	}
}

func TestMarkFailed_ComputesDuration(t *testing.T) {
	st := NewStepTracker("Auth")
	st.MarkRunning(0)
	time.Sleep(10 * time.Millisecond)
	st.MarkFailed(0, "timeout")

	if st.Steps[0].Status != StepFailed {
		t.Fatalf("Status = %v, want StepFailed", st.Steps[0].Status)
	}
	if st.Steps[0].Duration < 10*time.Millisecond {
		t.Errorf("Duration = %v, want >= 10ms", st.Steps[0].Duration)
	}
}

func TestViewRunning_ShowsElapsed(t *testing.T) {
	st := NewStepTracker("Auth")
	st.Steps[0].Status = StepRunning
	st.Steps[0].StartedAt = time.Now().Add(-5 * time.Second)

	view := st.View()
	if !strings.Contains(view, "(") || !strings.Contains(view, "s)") {
		t.Errorf("View() = %q, expected elapsed time in parentheses", view)
	}
}

func TestViewDone_ShowsDuration(t *testing.T) {
	st := NewStepTracker("Auth")
	st.Steps[0].Status = StepDone
	st.Steps[0].Detail = "verified"
	st.Steps[0].Duration = 3 * time.Second

	view := st.View()
	if !strings.Contains(view, "(3s)") {
		t.Errorf("View() = %q, expected (3s)", view)
	}
}

func TestViewDone_ShowsMinutesForLongDuration(t *testing.T) {
	st := NewStepTracker("Analysis")
	st.Steps[0].Status = StepDone
	st.Steps[0].Detail = "11 metrics"
	st.Steps[0].Duration = 2*time.Minute + 15*time.Second

	view := st.View()
	if !strings.Contains(view, "(2m 15s)") {
		t.Errorf("View() = %q, expected (2m 15s)", view)
	}
}

func TestViewPending_NoTimer(t *testing.T) {
	st := NewStepTracker("Auth")
	view := st.View()
	if strings.Contains(view, "(") && strings.Contains(view, "s)") {
		t.Errorf("View() = %q, pending steps should not show timer", view)
	}
}
