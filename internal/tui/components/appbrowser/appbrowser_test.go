package appbrowser_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/appbrowser"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestAppBrowser_RenderApps(t *testing.T) {
	ctx := tuictx.New(120, 40)
	apps := []appbrowser.AppEntry{
		{Name: "web", Service: "myapp-web-staging"},
		{Name: "worker", Service: "myapp-worker-staging"},
	}
	m := appbrowser.NewModel(ctx, apps)
	view := m.View()

	if !strings.Contains(view, "web") {
		t.Fatalf("expected 'web' in view, got: %s", view)
	}
	if !strings.Contains(view, "worker") {
		t.Fatalf("expected 'worker' in view, got: %s", view)
	}
}

func TestAppBrowser_SelectedApp(t *testing.T) {
	ctx := tuictx.New(120, 40)
	apps := []appbrowser.AppEntry{
		{Name: "web", Service: "myapp-web-staging"},
		{Name: "worker", Service: "myapp-worker-staging"},
	}
	m := appbrowser.NewModel(ctx, apps)

	if m.SelectedApp() != "web" {
		t.Fatalf("expected default selection 'web', got %q", m.SelectedApp())
	}

	m.MoveDown()
	if m.SelectedApp() != "worker" {
		t.Fatalf("expected 'worker' after MoveDown, got %q", m.SelectedApp())
	}

	m.MoveDown() // should clamp at last
	if m.SelectedApp() != "worker" {
		t.Fatalf("expected 'worker' after clamp, got %q", m.SelectedApp())
	}

	m.MoveUp()
	if m.SelectedApp() != "web" {
		t.Fatalf("expected 'web' after MoveUp, got %q", m.SelectedApp())
	}
}

func TestAppBrowser_PhaseSelection(t *testing.T) {
	ctx := tuictx.New(120, 40)
	apps := []appbrowser.AppEntry{{Name: "web", Service: "svc"}}
	m := appbrowser.NewModel(ctx, apps)

	if m.SelectedPhase() != "smoke" {
		t.Fatalf("expected default phase 'smoke', got %q", m.SelectedPhase())
	}

	m.NextPhase()
	if m.SelectedPhase() != "load" {
		t.Fatalf("expected 'load' after NextPhase, got %q", m.SelectedPhase())
	}

	m.NextPhase()
	m.NextPhase()
	if m.SelectedPhase() != "soak" {
		t.Fatalf("expected 'soak' after 3x NextPhase, got %q", m.SelectedPhase())
	}

	m.NextPhase() // should wrap to smoke
	if m.SelectedPhase() != "smoke" {
		t.Fatalf("expected wrap to 'smoke', got %q", m.SelectedPhase())
	}
}

func TestAppBrowser_PrevPhaseWraps(t *testing.T) {
	ctx := tuictx.New(120, 40)
	apps := []appbrowser.AppEntry{{Name: "web", Service: "svc"}}
	m := appbrowser.NewModel(ctx, apps)

	m.PrevPhase() // wrap from smoke to soak
	if m.SelectedPhase() != "soak" {
		t.Fatalf("expected wrap to 'soak', got %q", m.SelectedPhase())
	}
}

func TestAppBrowser_EmptyApps(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := appbrowser.NewModel(ctx, nil)

	if m.SelectedApp() != "" {
		t.Fatalf("expected empty selection for nil apps, got %q", m.SelectedApp())
	}

	view := m.View()
	if view == "" {
		t.Fatal("expected non-empty view even with no apps")
	}
}

func TestAppBrowser_SetWidth(t *testing.T) {
	ctx := tuictx.New(120, 40)
	apps := []appbrowser.AppEntry{{Name: "web", Service: "svc"}}
	m := appbrowser.NewModel(ctx, apps)
	m.SetWidth(80)

	view := m.View()
	if !strings.Contains(view, "web") {
		t.Fatalf("expected 'web' in narrow view, got: %s", view)
	}
}
