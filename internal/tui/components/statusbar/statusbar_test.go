package statusbar_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/statusbar"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestEmptyView(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := statusbar.NewModel(ctx)

	if m.View() != "" {
		t.Error("expected empty string when no items set")
	}
}

func TestSingleItem(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := statusbar.NewModel(ctx)
	m.SetItems([]statusbar.Item{
		{Label: "elapsed", Value: "2m 34s"},
	})

	view := m.View()
	if !strings.Contains(view, "elapsed") {
		t.Error("expected 'elapsed' in view")
	}
	if !strings.Contains(view, "2m 34s") {
		t.Error("expected '2m 34s' in view")
	}
}

func TestMultipleItems(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := statusbar.NewModel(ctx)
	m.SetItems([]statusbar.Item{
		{Label: "elapsed", Value: "2m 34s"},
		{Label: "RPS", Value: "150"},
		{Label: "CPU", Value: "67%"},
	})

	view := m.View()
	if !strings.Contains(view, "elapsed") {
		t.Error("expected 'elapsed' in view")
	}
	if !strings.Contains(view, "RPS") {
		t.Error("expected 'RPS' in view")
	}
	if !strings.Contains(view, "CPU") {
		t.Error("expected 'CPU' in view")
	}
}

func TestUpdateContext(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := statusbar.NewModel(ctx)

	ctx2 := tuictx.New(80, 24)
	m.UpdateContext(ctx2)
	m.SetItems([]statusbar.Item{{Label: "test", Value: "ok"}})

	view := m.View()
	if len(view) == 0 {
		t.Error("expected non-empty view after UpdateContext")
	}
}

func TestSetItemsReplacesAll(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := statusbar.NewModel(ctx)

	m.SetItems([]statusbar.Item{{Label: "a", Value: "1"}})
	m.SetItems([]statusbar.Item{{Label: "b", Value: "2"}})

	view := m.View()
	if strings.Contains(view, "a") {
		t.Error("old items should be replaced")
	}
	if !strings.Contains(view, "b") {
		t.Error("expected new item 'b' in view")
	}
}
