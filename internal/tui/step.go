package tui

import (
	"fmt"
	"strings"
	"time"
)

// StepStatus represents the current state of a step.
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepDone
	StepFailed
)

// Step represents a single item in the step tracker.
type Step struct {
	Name      string
	Status    StepStatus
	Detail    string
	StartedAt time.Time
	Duration  time.Duration
}

// StepTracker renders a progress list of steps.
type StepTracker struct {
	Steps []Step
}

// NewStepTracker creates a StepTracker with the given step names, all pending.
func NewStepTracker(names ...string) *StepTracker {
	steps := make([]Step, len(names))
	for i, n := range names {
		steps[i] = Step{Name: n, Status: StepPending}
	}
	return &StepTracker{Steps: steps}
}

// SetStatus sets the status of the step at the given index.
func (st *StepTracker) SetStatus(index int, status StepStatus) {
	if index >= 0 && index < len(st.Steps) {
		st.Steps[index].Status = status
	}
}

// SetDetail sets the detail text for the step at the given index.
func (st *StepTracker) SetDetail(index int, detail string) {
	if index >= 0 && index < len(st.Steps) {
		st.Steps[index].Detail = detail
	}
}

// MarkDone sets the step to StepDone, assigns the detail text, and computes duration.
func (st *StepTracker) MarkDone(index int, detail string) {
	if index >= 0 && index < len(st.Steps) {
		if !st.Steps[index].StartedAt.IsZero() {
			st.Steps[index].Duration = time.Since(st.Steps[index].StartedAt)
		}
		st.Steps[index].Status = StepDone
		st.Steps[index].Detail = detail
	}
}

// MarkFailed sets the step to StepFailed, assigns the detail text, and computes duration.
func (st *StepTracker) MarkFailed(index int, detail string) {
	if index >= 0 && index < len(st.Steps) {
		if !st.Steps[index].StartedAt.IsZero() {
			st.Steps[index].Duration = time.Since(st.Steps[index].StartedAt)
		}
		st.Steps[index].Status = StepFailed
		st.Steps[index].Detail = detail
	}
}

// MarkRunning sets the step to StepRunning and records the start time.
func (st *StepTracker) MarkRunning(index int) {
	if index >= 0 && index < len(st.Steps) {
		st.Steps[index].Status = StepRunning
		st.Steps[index].StartedAt = time.Now()
	}
}

// fmtDuration formats a duration as "Xs" for durations under 60s,
// or "Xm Ys" for durations of 60s or more.
func fmtDuration(d time.Duration) string {
	d = d.Truncate(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	return fmt.Sprintf("%dm %ds", m, s)
}

// View renders the step list as a string.
func (st *StepTracker) View() string {
	var b strings.Builder
	for _, s := range st.Steps {
		switch s.Status {
		case StepPending:
			b.WriteString("  \u25cb " + DimStyle.Render(s.Name))
		case StepRunning:
			line := "  \u25b8 " + BoldStyle.Render(s.Name) + "..."
			if s.Detail != "" {
				line += " " + s.Detail
			}
			if !s.StartedAt.IsZero() {
				elapsed := time.Since(s.StartedAt).Truncate(time.Second)
				line += " " + DimStyle.Render("("+fmtDuration(elapsed)+")")
			}
			b.WriteString(line)
		case StepDone:
			line := "  \u2713 " + SuccessStyle.Render(s.Name)
			if s.Detail != "" {
				line += ": " + s.Detail
			}
			if s.Duration > 0 {
				line += " " + DimStyle.Render("("+fmtDuration(s.Duration)+")")
			}
			b.WriteString(line)
		case StepFailed:
			line := "  \u2717 " + ErrorStyle.Render(s.Name)
			if s.Detail != "" {
				line += ": " + s.Detail
			}
			if s.Duration > 0 {
				line += " " + DimStyle.Render("("+fmtDuration(s.Duration)+")")
			}
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
