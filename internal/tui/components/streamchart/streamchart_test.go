package streamchart_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/components/streamchart"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestNewModel_viewContainsTitleAndUnit(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "Throughput", "req/s", 60, 12)

	view := m.View()
	if !strings.Contains(view, "Throughput") {
		t.Error("View() should contain title 'Throughput'")
	}
	if !strings.Contains(view, "req/s") {
		t.Error("View() should contain unit 'req/s'")
	}
}

func TestPush_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "Throughput", "req/s", 60, 12)

	now := time.Now()
	m.Push(now, 42.5)
	m.Push(now.Add(time.Second), 55.0)
	m.Push(now.Add(2*time.Second), 30.0)

	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty string after pushing data")
	}
}

func TestResize_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "Throughput", "req/s", 60, 12)

	now := time.Now()
	m.Push(now, 42.5)

	m.Resize(80, 16)
	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty string after resize")
	}
}

func TestResize_zeroWidth(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "RPS", "req/s", 60, 12)

	m.Push(time.Now(), 10.0)
	m.Resize(0, 0) // should not panic due to width guard
}

func TestUpdateContext(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "RPS", "req/s", 60, 12)

	ctx2 := tuictx.New(80, 24)
	m.UpdateContext(ctx2)

	view := m.View()
	if !strings.Contains(view, "RPS") {
		t.Error("View() should still contain title after UpdateContext")
	}
}

func TestAnnotation_viewContainsLabel(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "RPS", "req/s", 60, 12)

	now := time.Now()
	m.Push(now, 10.0)
	m.AddAnnotation(streamchart.Annotation{
		Time:  now.Add(5 * time.Second),
		Label: "scale 2->3",
		Style: ctx.Styles.Timeline.Scaling,
	})

	view := m.View()
	if !strings.Contains(view, "scale 2->3") {
		t.Errorf("expected annotation label in view, got: %s", view)
	}
	if !strings.Contains(view, "▼") {
		t.Error("expected annotation marker ▼ in view")
	}
}

func TestAnnotation_maxThreeShown(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "RPS", "req/s", 60, 12)

	now := time.Now()
	m.Push(now, 10.0)
	for i := 0; i < 5; i++ {
		m.AddAnnotation(streamchart.Annotation{
			Time:  now.Add(time.Duration(i) * time.Second),
			Label: strings.Repeat("x", 1),
			Style: ctx.Styles.Timeline.Scaling,
		})
	}

	view := m.View()
	// Should contain the marker but only last 3
	count := strings.Count(view, "▼")
	if count > 3 {
		t.Errorf("expected at most 3 annotation markers, got %d", count)
	}
}

func TestSetIdle(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "RPS", "req/s", 40, 12)
	m.Resize(40, 12)
	m.SetIdle(true)

	view := m.View()
	if !strings.Contains(view, "(paused)") {
		t.Errorf("idle chart should show '(paused)', got %q", view)
	}
}

func TestAnnotation_empty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "RPS", "req/s", 60, 12)

	view := m.View()
	if strings.Contains(view, "▼") {
		t.Error("should not contain annotation marker when no annotations")
	}
}
