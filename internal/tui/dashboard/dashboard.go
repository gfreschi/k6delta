// Package dashboard provides the root TUI model for the k6delta dashboard.
package dashboard

import (
	"sort"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/tui/components/appbrowser"
	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	"github.com/gfreschi/k6delta/internal/tui/components/header"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/keys"
)

// Model is the root dashboard Bubble Tea model. It composes the header,
// footer, and app browser, and dispatches key events to the active view.
type Model struct {
	cfg *config.Config
	env string

	ctx        *tuictx.ProgramContext
	headerComp header.Model
	footerComp footer.Model
	browser    appbrowser.Model

	ready bool
}

// NewModel creates a dashboard model from a loaded config. The store
// parameter is reserved for future report storage (pass nil for now).
func NewModel(cfg *config.Config, env string, _ interface{}) Model {
	ctx := tuictx.New(80, 24)
	ctx.ActiveView = tuictx.ViewBrowsing

	apps := buildAppEntries(cfg, env)

	h := header.NewModel(ctx, "", "", "")
	h.SetBreadcrumbs([]string{"k6delta"})

	viewLabel := env
	if cfg.Provider != "" {
		viewLabel += " | " + cfg.Provider
	}

	f := footer.NewModelWithHints(ctx, dashboardHints())
	f.SetViewLabel(viewLabel)

	return Model{
		cfg:        cfg,
		env:        env,
		ctx:        ctx,
		headerComp: h,
		footerComp: f,
		browser:    appbrowser.NewModel(ctx, apps),
	}
}

// ViewType returns the current active view type.
func (m Model) ViewType() tuictx.ViewType {
	return m.ctx.ActiveView
}

// SelectedApp returns the name of the currently highlighted app.
func (m Model) SelectedApp() string {
	return m.browser.SelectedApp()
}

// SelectedPhase returns the currently selected test phase.
func (m Model) SelectedPhase() string {
	return m.browser.SelectedPhase()
}

// Init enters the alternate screen buffer.
func (m Model) Init() tea.Cmd {
	return tea.EnterAltScreen
}

// Update handles messages and dispatches to sub-components.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.ctx.Resize(msg.Width, msg.Height)
		m.browser.SetWidth(m.ctx.ContentWidth)
		m.browser.UpdateContext(m.ctx)
		m.headerComp.UpdateContext(m.ctx)
		m.footerComp.UpdateContext(m.ctx)
		m.footerComp.SetWidth(msg.Width)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Keys.Up):
		m.browser.MoveUp()

	case key.Matches(msg, keys.Keys.Down):
		m.browser.MoveDown()

	case key.Matches(msg, keys.DashboardKeys.NextPhase):
		m.browser.NextPhase()

	case key.Matches(msg, keys.DashboardKeys.PrevPhase):
		m.browser.PrevPhase()

	case key.Matches(msg, keys.DashboardKeys.Run):
		// Sub-view integration deferred to 0.1.6c
		return m, nil

	case key.Matches(msg, keys.DashboardKeys.Analyze):
		return m, nil

	case key.Matches(msg, keys.DashboardKeys.Compare):
		return m, nil

	case key.Matches(msg, keys.DashboardKeys.Reports):
		return m, nil

	case key.Matches(msg, keys.DashboardKeys.Demo):
		return m, nil
	}

	return m, nil
}

// View renders the full dashboard: header, browser, and footer.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		m.headerComp.View(),
		"",
		m.browser.View(),
		"",
		m.footerComp.View(),
	)
}

func buildAppEntries(cfg *config.Config, env string) []appbrowser.AppEntry {
	if len(cfg.Apps) == 0 {
		return nil
	}

	names := make([]string, 0, len(cfg.Apps))
	for name := range cfg.Apps {
		names = append(names, name)
	}
	sort.Strings(names)

	entries := make([]appbrowser.AppEntry, len(names))
	for i, name := range names {
		appCfg := cfg.Apps[name]
		resolved := config.Interpolate(appCfg, name, env, "smoke", cfg.Region, cfg.Defaults.ResultsDir)
		entries[i] = appbrowser.AppEntry{
			Name:    name,
			Service: resolved.Service,
		}
	}
	return entries
}

func dashboardHints() []footer.Hint {
	return []footer.Hint{
		{Key: "enter", Label: "run", Short: "run"},
		{Key: "a", Label: "analyze", Short: "anlz"},
		{Key: "c", Label: "compare", Short: "cmp"},
		{Key: "r", Label: "reports", Short: "rpts"},
		{Key: "j/k", Label: "navigate", Short: "nav"},
		{Key: "h/l", Label: "phase", Short: "phs"},
		{Key: "?", Label: "help", Short: "?"},
		{Key: "q", Label: "quit", Short: "quit"},
	}
}
