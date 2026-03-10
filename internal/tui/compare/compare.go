// Package comparetui implements the Bubble Tea model for k6delta compare.
package comparetui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/report"
	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/components/focus"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/constants"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/keys"
)

// Model is the Bubble Tea model for the compare TUI.
type Model struct {
	pathA  string
	pathB  string
	result *report.ComparisonResult

	ctx        *tuictx.ProgramContext
	headerComp header.Model
	footerComp footer.Model

	k6Panel    panel.Model
	infraPanel panel.Model
	focusMgr   *focus.Manager

	sort         sortMode
	exportStatus string
	err          error
	quitting     bool

	// Help overlay
	showHelp bool

	// Side-by-side diff mode (only at >=140 width)
	diffMode bool

	// Drill-down active (Enter toggles, Escape exits)
	drillActive bool
}

// NewModel creates a new compare TUI model for the two given report paths.
func NewModel(pathA, pathB string) Model {
	ctx := tuictx.New(80, 24)
	hdr := header.NewModel(ctx, "", "", "compare")
	ftr := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
	})
	return Model{pathA: pathA, pathB: pathB, ctx: ctx, headerComp: hdr, footerComp: ftr}
}

func (m Model) Init() tea.Cmd {
	return func() tea.Msg {
		result, err := report.CompareReports(m.pathA, m.pathB)
		if err != nil {
			return errMsg{err: err}
		}
		return resultMsg{result: result}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case resultMsg:
		m.result = msg.result
		m.initDashboard()
		return m, nil
	case errMsg:
		m.err = msg.err
		return m, nil
	case exportDoneMsg:
		m.exportStatus = fmt.Sprintf("Exported to %s", msg.path)
		return m, nil
	case panel.TransitionTickMsg:
		if m.focusMgr != nil {
			cmd := tea.Batch(m.k6Panel.AdvanceTransition(), m.infraPanel.AdvanceTransition())
			return m, cmd
		}
	case panel.ExpandTransitionTickMsg:
		if m.focusMgr != nil {
			cmd := tea.Batch(m.k6Panel.AdvanceExpandTransition(), m.infraPanel.AdvanceExpandTransition())
			return m, cmd
		}
	case tea.KeyMsg:
		if key.Matches(msg, keys.Keys.Help) {
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.footerComp.SetState(footer.StateHelp)
			} else {
				m.footerComp.SetState(footer.StateNormal)
			}
			return m, nil
		}
		if m.showHelp {
			if key.Matches(msg, keys.Keys.Escape) {
				m.showHelp = false
				m.footerComp.SetState(footer.StateNormal)
				return m, nil
			}
			return m, nil
		}
		switch {
		case key.Matches(msg, keys.Keys.Quit):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, keys.CompareKeys.Export):
			if m.result != nil {
				return m, m.exportComparison()
			}
			return m, nil
		case key.Matches(msg, keys.CompareKeys.Sort):
			m.sort = (m.sort + 1) % 3
			m.refreshPanels()
			return m, nil
		case key.Matches(msg, keys.CompareKeys.Diff):
			if m.ctx.ContentWidth >= constants.BreakpointWide {
				m.diffMode = !m.diffMode
			}
			return m, nil
		}
		if m.focusMgr != nil {
			switch {
			case key.Matches(msg, keys.Keys.NextPanel):
				m.focusMgr.Next()
				return m, m.updatePanelFocus()
			case key.Matches(msg, keys.Keys.PrevPanel):
				m.focusMgr.Prev()
				return m, m.updatePanelFocus()
			case key.Matches(msg, keys.Keys.Down):
				m.scrollFocusedPanel(1)
				return m, nil
			case key.Matches(msg, keys.Keys.Up):
				m.scrollFocusedPanel(-1)
				return m, nil
			case key.Matches(msg, keys.Keys.Jump1):
				m.focusMgr.SetFocus(0)
				return m, m.updatePanelFocus()
			case key.Matches(msg, keys.Keys.Jump2):
				m.focusMgr.SetFocus(1)
				return m, m.updatePanelFocus()
			case key.Matches(msg, keys.Keys.Expand):
				cmd := m.cycleExpandFocusedPanel()
				m.footerComp.SetState(footer.StateExpanded)
				return m, cmd
			case key.Matches(msg, keys.Keys.Enter):
				if !m.drillActive {
					// Drill-down requires full-expand to show A/B detail
					m.drillActive = true
					m.expandFocusedPanelFull()
					m.footerComp.SetState(footer.StateExpanded)
					m.refreshDrillPanel()
				}
				return m, nil
			case key.Matches(msg, keys.Keys.Escape):
				if m.drillActive {
					m.drillActive = false
					m.resetAllPanelExpand()
					m.footerComp.SetState(footer.StateNormal)
					m.refreshPanels()
					return m, nil
				}
				if m.anyPanelExpanded() {
					m.resetAllPanelExpand()
					m.footerComp.SetState(footer.StateNormal)
					return m, nil
				}
			}
		}
	case tea.WindowSizeMsg:
		m.ctx.Resize(msg.Width, msg.Height)
		m.headerComp.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
		if m.focusMgr != nil {
			m.k6Panel.UpdateContext(m.ctx)
			m.infraPanel.UpdateContext(m.ctx)
			m.resizePanels()
		}
		// Disable diff mode if terminal shrinks below threshold
		if m.ctx.ContentWidth < constants.BreakpointWide {
			m.diffMode = false
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.showHelp {
		return m.renderHelpOverlay()
	}

	s := m.ctx.Styles

	if m.err != nil {
		return s.Common.ErrorStyle.Render("Error: "+m.err.Error()) + "\n"
	}
	if m.result == nil {
		return common.RenderEmptyState(m.ctx.Styles.Common, common.EmptyPending, "Loading comparison...", "")
	}

	var sections []string

	// Metadata header
	title := fmt.Sprintf("Compare: %s %s (%s)", m.result.RunA.App, m.result.RunA.Phase, m.result.RunA.Env)
	sections = append(sections, "",
		s.Header.Root.Render(title),
		fmt.Sprintf("  Run A: %s   Run B: %s",
			s.Common.FaintTextStyle.Render(m.result.RunA.Start),
			s.Common.FaintTextStyle.Render(m.result.RunB.Start)),
		"")

	// Delta KPI strip
	deltaStrip := m.renderDeltaStrip()
	if deltaStrip != "" {
		sections = append(sections, deltaStrip)
	}

	// Panels or side-by-side
	width := m.ctx.ContentWidth
	if m.focusMgr != nil {
		if m.diffMode && width >= constants.BreakpointWide {
			sections = append(sections, m.renderSideBySide())
		} else {
			focused := m.focusMgr.Current()
			panels := [2]panel.Model{m.k6Panel, m.infraPanel}

			if panels[focused].ExpandMode() == constants.ExpandFull {
				sections = append(sections, panels[focused].View())
			} else if width >= constants.BreakpointStacked {
				sections = append(sections, m.k6Panel.View())
				sections = append(sections, m.infraPanel.View())
			} else {
				sections = append(sections, m.renderK6Table(), "", m.renderInfraTable())
			}
		}
	}

	// Regression verdict tile + footer
	verdictTile := m.renderRegressionVerdict()
	if verdictTile != "" {
		sections = append(sections, "", verdictTile)
	}
	if m.exportStatus != "" {
		sections = append(sections, m.ctx.Styles.Common.SuccessStyle.Render("  "+m.exportStatus))
	}
	sections = append(sections, "", m.footerComp.View())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// --- Dashboard Lifecycle ---

func (m *Model) initDashboard() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	m.k6Panel = panel.NewModel(m.ctx, m.k6PanelTitle(), w, panelH)
	m.k6Panel.SetContent(m.renderK6Table())

	m.infraPanel = panel.NewModel(m.ctx, "Infrastructure [2]", w, panelH)
	m.infraPanel.SetContent(m.renderInfraTable())

	m.k6Panel.SetDrillable(true)
	m.infraPanel.SetDrillable(true)

	m.focusMgr = focus.New(2)
	m.k6Panel.SetFocused(true)

	hints := []footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "tab", Action: "panel"},
		{Key: "1-2", Action: "jump"},
		{Key: "+", Action: "expand"},
		{Key: "enter", Action: "drill"},
		{Key: "↑↓", Action: "scroll"},
		{Key: "s", Action: "sort"},
		{Key: "e", Action: "export"},
	}
	if m.ctx.ContentWidth >= constants.BreakpointWide {
		hints = append(hints, footer.KeyHint{Key: "d", Action: "diff"})
	}
	hints = append(hints, footer.KeyHint{Key: "?", Action: "help"})
	m.footerComp.SetHints(hints)
}

func (m *Model) resizePanels() {
	w := m.ctx.ContentWidth
	panelH := m.ctx.ContentHeight / 2

	m.k6Panel.SetDimensions(w, panelH)
	m.infraPanel.SetDimensions(w, panelH)
}

func (m *Model) refreshPanels() {
	if m.focusMgr == nil {
		return
	}
	m.k6Panel.SetTitle(m.k6PanelTitle())
	m.k6Panel.SetContent(m.renderK6Table())
	m.infraPanel.SetContent(m.renderInfraTable())
}

func (m *Model) refreshDrillPanel() {
	if m.focusMgr == nil || !m.drillActive {
		return
	}
	focused := m.focusMgr.Current()
	panels := []*panel.Model{&m.k6Panel, &m.infraPanel}
	panels[focused].SetContent(m.renderDrillDown())
}

func (m *Model) updatePanelFocus() tea.Cmd {
	m.k6Panel.SetFocused(m.focusMgr.IsFocused(0))
	m.infraPanel.SetFocused(m.focusMgr.IsFocused(1))
	return tea.Batch(m.k6Panel.TransitionCmd(), m.infraPanel.TransitionCmd())
}

func (m *Model) cycleExpandFocusedPanel() tea.Cmd {
	panels := []*panel.Model{&m.k6Panel, &m.infraPanel}
	idx := m.focusMgr.Current()
	if idx >= 0 && idx < len(panels) {
		return panels[idx].CycleExpand()
	}
	return nil
}

func (m *Model) expandFocusedPanelFull() {
	panels := []*panel.Model{&m.k6Panel, &m.infraPanel}
	idx := m.focusMgr.Current()
	if idx >= 0 && idx < len(panels) {
		panels[idx].SetExpandFull()
	}
}

func (m Model) anyPanelExpanded() bool {
	return m.k6Panel.ExpandMode() != constants.ExpandNormal ||
		m.infraPanel.ExpandMode() != constants.ExpandNormal
}

func (m *Model) resetAllPanelExpand() {
	m.k6Panel.ResetExpand()
	m.infraPanel.ResetExpand()
}

func (m *Model) scrollFocusedPanel(dir int) {
	var p *panel.Model
	switch m.focusMgr.Current() {
	case 0:
		p = &m.k6Panel
	case 1:
		p = &m.infraPanel
	}
	if p == nil {
		return
	}
	if dir > 0 {
		p.ScrollDown()
	} else {
		p.ScrollUp()
	}
}

func (m Model) k6PanelTitle() string {
	if m.sort != sortDefault {
		return fmt.Sprintf("k6 Metrics [1] (sorted: %s)", m.sort)
	}
	return "k6 Metrics [1]"
}

// RunJSON performs the comparison and prints JSON, bypassing the TUI.
func RunJSON(pathA, pathB string) error {
	data, err := report.CompareReportsJSON(pathA, pathB)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}
