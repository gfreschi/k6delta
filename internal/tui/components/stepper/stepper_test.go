package stepper_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/components/stepper"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func testContext() *tuictx.ProgramContext {
	return tuictx.New(120, 40)
}

func TestStepper_rendersSteps(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Step A", "Step B", "Step C")
	view := s.View()
	if !strings.Contains(view, "Step A") {
		t.Error("expected 'Step A' in view")
	}
	if !strings.Contains(view, "Step B") {
		t.Error("expected 'Step B' in view")
	}
}

func TestStepper_markRunning(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Auth", "Snapshot")
	before := time.Now()
	s.MarkRunning(0)
	after := time.Now()

	steps := s.Steps()
	if steps[0].Status != stepper.StepRunning {
		t.Fatalf("Status = %v, want StepRunning", steps[0].Status)
	}
	if steps[0].StartedAt.Before(before) || steps[0].StartedAt.After(after) {
		t.Errorf("StartedAt not within expected range")
	}
}

func TestStepper_markDoneComputesDuration(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Auth")
	s.MarkRunning(0)
	time.Sleep(10 * time.Millisecond)
	s.MarkDone(0, "verified")

	steps := s.Steps()
	if steps[0].Status != stepper.StepDone {
		t.Fatalf("Status = %v, want StepDone", steps[0].Status)
	}
	if steps[0].Duration < 10*time.Millisecond {
		t.Errorf("Duration = %v, want >= 10ms", steps[0].Duration)
	}
}

func TestStepper_markFailedComputesDuration(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Auth")
	s.MarkRunning(0)
	time.Sleep(10 * time.Millisecond)
	s.MarkFailed(0, "timeout")

	steps := s.Steps()
	if steps[0].Status != stepper.StepFailed {
		t.Fatalf("Status = %v, want StepFailed", steps[0].Status)
	}
	if steps[0].Duration < 10*time.Millisecond {
		t.Errorf("Duration = %v, want >= 10ms", steps[0].Duration)
	}
}

func TestStepper_viewShowsElapsed(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Auth")
	s.MarkRunning(0)
	// Manually set start time in the past to get predictable output
	steps := s.Steps()
	steps[0].StartedAt = time.Now().Add(-5 * time.Second)

	view := s.View()
	if !strings.Contains(view, "(") || !strings.Contains(view, "s)") {
		t.Errorf("View() = %q, expected elapsed time in parentheses", view)
	}
}

func TestStepper_viewShowsDuration(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Auth")
	steps := s.Steps()
	steps[0].Status = stepper.StepDone
	steps[0].Detail = "verified"
	steps[0].Duration = 3 * time.Second

	view := s.View()
	if !strings.Contains(view, "(3s)") {
		t.Errorf("View() = %q, expected (3s)", view)
	}
}

func TestStepper_flashClearedByMethod(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Auth")
	s.MarkRunning(0)
	cmd := s.MarkDone(0, "verified")

	// MarkDone should return a non-nil command (the auto-clear tick)
	if cmd == nil {
		t.Error("expected MarkDone to return a tea.Cmd for flash auto-clear")
	}

	// View should render without error while flash is active
	view1 := s.View()
	if !strings.Contains(view1, "Auth") {
		t.Error("expected step name in view")
	}

	// Manually clear flash (simulating ClearFlashMsg handler)
	s.ClearFlash(0)

	// Verify flash state is cleared via View still working
	view2 := s.View()
	if !strings.Contains(view2, "Auth") {
		t.Error("expected step name in view after clearing flash")
	}
}

func TestStepper_viewShowsMinutesForLongDuration(t *testing.T) {
	ctx := testContext()
	s := stepper.NewModel(ctx, "Analysis")
	steps := s.Steps()
	steps[0].Status = stepper.StepDone
	steps[0].Detail = "11 metrics"
	steps[0].Duration = 2*time.Minute + 15*time.Second

	view := s.View()
	if !strings.Contains(view, "(2m 15s)") {
		t.Errorf("View() = %q, expected (2m 15s)", view)
	}
}
