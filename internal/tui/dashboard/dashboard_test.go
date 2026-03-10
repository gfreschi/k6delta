package dashboard_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/tui/context"
	"github.com/gfreschi/k6delta/internal/tui/dashboard"
)

func testConfig() *config.Config {
	return &config.Config{
		Provider: "mock",
		Region:   "us-east-1",
		Defaults: config.Defaults{Env: "staging", Phase: "smoke"},
		Apps: map[string]config.AppConfig{
			"web":    {Service: "myapp-web-${env}", TestFile: "tests/smoke.js"},
			"worker": {Service: "myapp-worker-${env}", TestFile: "tests/smoke.js"},
		},
	}
}

func sendResize(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	resized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return resized
}

func TestDashboard_InitialState(t *testing.T) {
	cfg := testConfig()
	m := dashboard.NewModel(cfg, "staging", nil)

	if m.ViewType() != context.ViewBrowsing {
		t.Fatalf("expected ViewBrowsing, got %d", m.ViewType())
	}
}

func TestDashboard_RenderShowsApps(t *testing.T) {
	cfg := testConfig()
	m := dashboard.NewModel(cfg, "staging", nil)

	model := sendResize(t, m)
	view := model.(dashboard.Model).View()

	if !strings.Contains(view, "web") {
		t.Fatalf("expected 'web' in dashboard view, got:\n%s", view)
	}
	if !strings.Contains(view, "worker") {
		t.Fatalf("expected 'worker' in dashboard view, got:\n%s", view)
	}
}

func TestDashboard_NavigateApps(t *testing.T) {
	cfg := testConfig()
	m := dashboard.NewModel(cfg, "staging", nil)

	model := sendResize(t, m)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	d := model.(dashboard.Model)
	if d.SelectedApp() == "" {
		t.Fatal("expected a selected app after navigation")
	}
}

func TestDashboard_PhaseNavigation(t *testing.T) {
	cfg := testConfig()
	m := dashboard.NewModel(cfg, "staging", nil)

	model := sendResize(t, m)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	d := model.(dashboard.Model)
	if d.SelectedPhase() != "load" {
		t.Fatalf("expected 'load' after pressing 'l', got %q", d.SelectedPhase())
	}
}

func TestDashboard_QuitKey(t *testing.T) {
	cfg := testConfig()
	m := dashboard.NewModel(cfg, "staging", nil)

	model := sendResize(t, m)
	_, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Fatal("expected quit command on 'q' press")
	}
}

func TestDashboard_EmptyConfig(t *testing.T) {
	cfg := &config.Config{Apps: map[string]config.AppConfig{}}
	m := dashboard.NewModel(cfg, "staging", nil)

	model := sendResize(t, m)
	view := model.(dashboard.Model).View()

	if !strings.Contains(view, "No apps") {
		t.Fatalf("expected empty state message, got:\n%s", view)
	}
}

func TestDashboard_NilConfig(t *testing.T) {
	cfg := &config.Config{}
	m := dashboard.NewModel(cfg, "staging", nil)

	model := sendResize(t, m)
	view := model.(dashboard.Model).View()

	if !strings.Contains(view, "k6delta init") {
		t.Fatalf("expected init guidance in empty state, got:\n%s", view)
	}
}
