// Package stepper provides a themed step-tracker component.
// This is a migration of internal/tui/step.go to use ctx.Styles.
package stepper

import (
	"fmt"
	"strings"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
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
	flash     bool // true for one render cycle after MarkDone
}

// Model is the stepper Bubble Tea model.
type Model struct {
	ctx   *tuictx.ProgramContext
	steps []Step
}

// NewModel creates a stepper with the given step names, all pending.
func NewModel(ctx *tuictx.ProgramContext, names ...string) Model {
	steps := make([]Step, len(names))
	for i, n := range names {
		steps[i] = Step{Name: n, Status: StepPending}
	}
	return Model{ctx: ctx, steps: steps}
}

// Steps returns the underlying step slice for direct access.
func (m *Model) Steps() []Step {
	return m.steps
}

// SetStatus sets the status of the step at the given index.
func (m *Model) SetStatus(index int, status StepStatus) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = status
	}
}

// SetDetail sets the detail text for the step at the given index.
func (m *Model) SetDetail(index int, detail string) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Detail = detail
	}
}

// MarkRunning sets the step to StepRunning and records the start time.
func (m *Model) MarkRunning(index int) {
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = StepRunning
		m.steps[index].StartedAt = time.Now()
	}
}

// MarkDone sets the step to StepDone, assigns the detail text, and computes duration.
// Sets flash flag for a bright completion indicator on the next render.
func (m *Model) MarkDone(index int, detail string) {
	if index >= 0 && index < len(m.steps) {
		if !m.steps[index].StartedAt.IsZero() {
			m.steps[index].Duration = time.Since(m.steps[index].StartedAt)
		}
		m.steps[index].Status = StepDone
		m.steps[index].Detail = detail
		m.steps[index].flash = true
	}
}

// MarkFailed sets the step to StepFailed, assigns the detail text, and computes duration.
func (m *Model) MarkFailed(index int, detail string) {
	if index >= 0 && index < len(m.steps) {
		if !m.steps[index].StartedAt.IsZero() {
			m.steps[index].Duration = time.Since(m.steps[index].StartedAt)
		}
		m.steps[index].Status = StepFailed
		m.steps[index].Detail = detail
	}
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the step list as a string.
func (m Model) View() string {
	st := m.ctx.Styles.Stepper
	cs := m.ctx.Styles.Common
	ds := m.ctx.Styles.Delta
	var b strings.Builder

	for i, s := range m.steps {
		switch s.Status {
		case StepPending:
			b.WriteString("  " + constants.IconPending + " " + st.Pending.Render(s.Name))
		case StepRunning:
			line := "  " + constants.IconRunning + " " + st.Running.Render(s.Name) + "..."
			if s.Detail != "" {
				line += " " + st.Detail.Render(s.Detail)
			}
			if !s.StartedAt.IsZero() {
				elapsed := time.Since(s.StartedAt).Truncate(time.Second)
				line += " " + st.Elapsed.Render("("+fmtDuration(elapsed)+")")
			}
			b.WriteString(line)
		case StepDone:
			icon := cs.CheckMark
			if s.flash {
				icon = ds.BetterStrong.Render(constants.IconDone)
				m.steps[i].flash = false
			}
			line := "  " + icon + " " + st.Done.Render(s.Name)
			if s.Detail != "" {
				line += ": " + st.Detail.Render(s.Detail)
			}
			if s.Duration > 0 {
				line += " " + st.Elapsed.Render("("+fmtDuration(s.Duration)+")")
			}
			b.WriteString(line)
		case StepFailed:
			line := "  " + cs.XMark + " " + st.Failed.Render(s.Name)
			if s.Detail != "" {
				line += ": " + st.Detail.Render(s.Detail)
			}
			if s.Duration > 0 {
				line += " " + st.Elapsed.Render("("+fmtDuration(s.Duration)+")")
			}
			b.WriteString(line)
		}
		b.WriteByte('\n')
	}
	return b.String()
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
