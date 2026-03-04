package timechart_test

import (
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/components/timechart"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestNewModel(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "Throughput", "req/s", 60, 12)

	if m.View() == "" {
		t.Error("View() should return non-empty string")
	}
}

func TestSetData(t *testing.T) {
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

func TestSetDataMismatchedLengths(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := timechart.NewModel(ctx, "Test", "units", 60, 12)

	times := []time.Time{time.Now()}
	values := []float64{1.0, 2.0}

	m.SetData(times, values) // should not panic, uses min length
}
