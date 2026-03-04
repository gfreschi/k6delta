package timechart_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/components/timechart"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestNewModel_viewContainsTitleAndUnit(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "Latency", "ms", 60, 12)

	view := m.View()
	if !strings.Contains(view, "Latency") {
		t.Error("View() should contain title 'Latency'")
	}
	if !strings.Contains(view, "ms") {
		t.Error("View() should contain unit 'ms'")
	}
}

func TestSetData_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "Latency", "ms", 60, 12)

	now := time.Now()
	times := []time.Time{now, now.Add(time.Minute), now.Add(2 * time.Minute)}
	values := []float64{100.0, 250.0, 180.0}

	m.SetData(times, values)

	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty after SetData")
	}
}

func TestSetData_mismatchedLengths(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "Test", "units", 60, 12)

	times := []time.Time{time.Now()}
	values := []float64{1.0, 2.0}

	m.SetData(times, values) // should not panic, uses min length
}

func TestResize_viewNonEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "Latency", "ms", 60, 12)

	now := time.Now()
	m.SetData(
		[]time.Time{now, now.Add(time.Minute)},
		[]float64{100.0, 200.0},
	)

	m.Resize(80, 16)
	view := m.View()
	if view == "" {
		t.Error("View() should return non-empty after resize")
	}
}

func TestResize_zeroWidth(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "RPS", "req/s", 60, 12)

	m.SetData(
		[]time.Time{time.Now()},
		[]float64{10.0},
	)
	m.Resize(0, 0) // should not panic after S5 fix
}

func TestUpdateContext(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "RPS", "req/s", 60, 12)

	ctx2 := tuictx.New(80, 24)
	m.UpdateContext(ctx2)

	view := m.View()
	if !strings.Contains(view, "RPS") {
		t.Error("View() should still contain title after UpdateContext")
	}
}
