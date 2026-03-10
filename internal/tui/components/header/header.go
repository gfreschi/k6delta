// Package header provides the app/env/phase context bar with elapsed timer and spinner.
package header

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model is the header Bubble Tea model.
type Model struct {
	ctx         *tuictx.ProgramContext
	app         string
	env         string
	phase       string
	status      string
	elapsed     time.Duration
	spinner     spinner.Model
	active      bool
	breadcrumbs []string
}

// NewModel creates a header.
func NewModel(ctx *tuictx.ProgramContext, app, env, phase string) Model {
	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	s.Style = ctx.Styles.Header.Context
	return Model{ctx: ctx, app: app, env: env, phase: phase, spinner: s}
}

// SetStatus sets the right-side status text.
func (m *Model) SetStatus(s string) {
	m.status = s
}

// SetElapsed sets the elapsed duration displayed in the header.
func (m *Model) SetElapsed(d time.Duration) {
	m.elapsed = d
}

// SetActive enables or disables the spinner animation.
func (m *Model) SetActive(active bool) {
	m.active = active
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// SetBreadcrumbs sets breadcrumb path segments. If non-nil, View renders
// breadcrumbs instead of the classic "app (env) -- phase" format.
func (m *Model) SetBreadcrumbs(parts []string) {
	m.breadcrumbs = parts
}

// Init returns the spinner tick command if active.
func (m Model) Init() tea.Cmd {
	if m.active {
		return m.spinner.Tick
	}
	return nil
}

// Update handles spinner tick messages.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if m.active {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View renders the header bar.
func (m Model) View() string {
	s := m.ctx.Styles.Header

	var left string
	if len(m.breadcrumbs) > 0 {
		sep := s.Context.Render(" > ")
		parts := make([]string, len(m.breadcrumbs))
		for i, p := range m.breadcrumbs {
			if i == 0 {
				parts[i] = s.Root.Render(p)
			} else {
				parts[i] = s.Title.Render(p)
			}
		}
		left = strings.Join(parts, sep)
	} else {
		left = s.Root.Render(fmt.Sprintf("k6delta: %s (%s) -- %s", m.app, m.env, m.phase))
	}

	var right string
	if m.status != "" {
		right += s.Context.Render(m.status)
	}
	if m.active {
		right += " " + m.spinner.View()
	}
	if m.elapsed > 0 {
		mins := int(m.elapsed.Minutes())
		secs := int(m.elapsed.Seconds()) % 60
		right += " " + s.Context.Render(fmt.Sprintf("%dm %ds", mins, secs))
	}

	if right != "" {
		return left + "  " + right
	}
	return left
}
