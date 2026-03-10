// Package appbrowser provides the dashboard app list with phase picker.
package appbrowser

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/common"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

var phases = []string{"smoke", "load", "stress", "soak"}

// AppEntry represents one app row in the browser.
type AppEntry struct {
	Name    string
	Service string
	LastRun string // formatted summary, e.g. "2m ago  PASS  p95:142ms" (empty = never)
}

// Model is the app browser component.
type Model struct {
	ctx      *tuictx.ProgramContext
	apps     []AppEntry
	cursor   int
	phaseIdx int
	width    int
}

// NewModel creates an app browser.
func NewModel(ctx *tuictx.ProgramContext, apps []AppEntry) Model {
	return Model{
		ctx:   ctx,
		apps:  apps,
		width: ctx.ContentWidth,
	}
}

// SelectedApp returns the name of the currently selected app (empty if no apps).
func (m *Model) SelectedApp() string {
	if len(m.apps) == 0 {
		return ""
	}
	return m.apps[m.cursor].Name
}

// SelectedPhase returns the currently selected phase.
func (m *Model) SelectedPhase() string {
	return phases[m.phaseIdx]
}

// MoveDown moves the cursor to the next app.
func (m *Model) MoveDown() {
	if m.cursor < len(m.apps)-1 {
		m.cursor++
	}
}

// MoveUp moves the cursor to the previous app.
func (m *Model) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// NextPhase advances the phase picker (wraps around).
func (m *Model) NextPhase() {
	m.phaseIdx = (m.phaseIdx + 1) % len(phases)
}

// PrevPhase moves the phase picker backward (wraps around).
func (m *Model) PrevPhase() {
	m.phaseIdx = (m.phaseIdx - 1 + len(phases)) % len(phases)
}

// SetWidth sets the available render width.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// UpdateContext updates the shared program context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the app browser.
func (m Model) View() string {
	if len(m.apps) == 0 {
		return common.RenderEmptyState(m.ctx.Styles.Common, common.EmptyNoData,
			"No apps configured",
			"Run k6delta init to create a config")
	}

	s := m.ctx.Styles

	var b strings.Builder

	// Column widths
	nameW := 16
	serviceW := 32
	lastRunW := m.width - nameW - serviceW - 4
	if lastRunW < 10 {
		lastRunW = 10
	}

	// Header row
	hdr := fmt.Sprintf("  %-*s %-*s %s",
		nameW, "App", serviceW, "Service", "Last Run")
	b.WriteString(s.Table.Header.Render(hdr))
	b.WriteString("\n")

	// Separator
	b.WriteString(s.Table.Separator.Render(strings.Repeat("-", m.width)))
	b.WriteString("\n")

	// Rows
	for i, app := range m.apps {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}

		lastRun := app.LastRun
		if lastRun == "" {
			lastRun = "--"
		}

		row := fmt.Sprintf("%s%-*s %-*s %-*s",
			prefix, nameW, app.Name, serviceW, truncate(app.Service, serviceW), lastRunW, lastRun)

		if i == m.cursor {
			b.WriteString(s.Common.BoldStyle.Render(row))
		} else {
			b.WriteString(s.Common.MainTextStyle.Render(row))
		}
		b.WriteString("\n")
	}

	// Phase picker
	b.WriteString("\n")
	phaseParts := make([]string, 0, len(phases))
	for i, p := range phases {
		if i == m.phaseIdx {
			phaseParts = append(phaseParts, s.Common.BoldStyle.Render(p))
		} else {
			phaseParts = append(phaseParts, s.Common.FaintTextStyle.Render(p))
		}
	}
	phaseLabel := s.Common.FaintTextStyle.Render("Phase: ")
	sep := s.Common.FaintTextStyle.Render(" | ")
	b.WriteString("  " + phaseLabel + strings.Join(phaseParts, sep))

	return b.String()
}

func truncate(s string, maxW int) string {
	if lipgloss.Width(s) <= maxW {
		return s
	}
	runes := []rune(s)
	if maxW <= 1 {
		if maxW <= 0 {
			return ""
		}
		return string(runes[:1])
	}
	return string(runes[:maxW-1]) + "~"
}
