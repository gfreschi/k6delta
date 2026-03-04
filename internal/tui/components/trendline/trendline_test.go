package trendline_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/trendline"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestNewModel_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := trendline.NewModel(ctx, 25, 1)

	if m.View() == "" {
		t.Error("View() should return non-empty string")
	}
}

func TestPush_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := trendline.NewModel(ctx, 25, 1)

	m.Push(10.0)
	m.Push(50.0)
	m.Push(80.0)

	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty after pushing data")
	}
}

func TestResize_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := trendline.NewModel(ctx, 25, 1)

	m.Push(42.0)
	m.Resize(40, 2)

	view := m.View()
	if view == "" {
		t.Error("View() should work after resize")
	}
}

func TestSetMax(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := trendline.NewModel(ctx, 25, 1)

	m.SetMax(200.0)
	m.Push(150.0)

	view := m.View()
	if view == "" {
		t.Error("View() should work after SetMax + Push")
	}
}

func TestUpdateContext(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := trendline.NewModel(ctx, 25, 1)

	ctx2 := tuictx.New(80, 24)
	m.UpdateContext(ctx2)

	m.Push(50.0)
	view := m.View()
	if view == "" {
		t.Error("View() should work after UpdateContext + Push")
	}
}
