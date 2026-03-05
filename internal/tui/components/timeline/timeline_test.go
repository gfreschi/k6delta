package timeline

import (
	"strings"
	"testing"
	"time"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestTimeline_rendersTimeAxis(t *testing.T) {
	ctx := tuictx.New(120, 40)
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(20 * time.Minute)

	m := NewModel(ctx, start, end, 80)
	view := m.View()
	if !strings.Contains(view, "10:30") {
		t.Fatalf("expected '10:30' in timeline, got:\n%s", view)
	}
}

func TestTimeline_AddLane(t *testing.T) {
	ctx := tuictx.New(120, 40)
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(20 * time.Minute)

	m := NewModel(ctx, start, end, 80)
	m.AddLane(Lane{
		Label:  "cpu",
		Values: []float64{35, 42, 55, 48, 38},
		Peak:   55,
		Unit:   "%",
	})

	view := m.View()
	if !strings.Contains(view, "cpu") {
		t.Fatalf("expected 'cpu' label in timeline, got:\n%s", view)
	}
}

func TestTimeline_AddEvent(t *testing.T) {
	ctx := tuictx.New(120, 40)
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(20 * time.Minute)

	m := NewModel(ctx, start, end, 80)
	m.AddEvent(Event{
		Start: start.Add(5 * time.Minute),
		End:   start.Add(10 * time.Minute),
		Type:  EventAlarm,
		Label: "alarm ON",
	})

	view := m.View()
	if !strings.Contains(view, "alarm") {
		t.Fatalf("expected 'alarm' in timeline events, got:\n%s", view)
	}
}

func TestTimeline_threshold(t *testing.T) {
	ctx := tuictx.New(120, 40)
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(20 * time.Minute)

	m := NewModel(ctx, start, end, 80)
	m.AddLane(Lane{
		Label:     "cpu",
		Values:    []float64{35, 42, 55, 48, 38},
		Peak:      55,
		Unit:      "%",
		Threshold: 90,
	})

	view := m.View()
	if !strings.Contains(view, "threshold") {
		t.Fatalf("expected 'threshold' in timeline, got:\n%s", view)
	}
}

func TestTimeline_abbreviated(t *testing.T) {
	ctx := tuictx.New(50, 24)
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(20 * time.Minute)

	m := NewModel(ctx, start, end, 50)
	m.AddLane(Lane{
		Label:  "cpu",
		Values: []float64{35, 42, 55},
		Peak:   55,
		Unit:   "%",
	})

	view := m.View()
	// Abbreviated mode: no time axis, but still has lane label
	if !strings.Contains(view, "cpu") {
		t.Fatalf("expected 'cpu' in abbreviated view, got:\n%s", view)
	}
	// Should NOT have time axis in abbreviated mode
	if strings.Contains(view, "10:30") {
		t.Fatalf("abbreviated view should not contain time axis, got:\n%s", view)
	}
}

func TestTimeline_eventList(t *testing.T) {
	ctx := tuictx.New(30, 24)
	start := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	end := start.Add(20 * time.Minute)

	m := NewModel(ctx, start, end, 30)
	m.AddEvent(Event{
		Start: start.Add(5 * time.Minute),
		End:   start.Add(10 * time.Minute),
		Type:  EventScaling,
		Label: "scale 2->4",
	})

	view := m.View()
	if !strings.Contains(view, "scale 2->4") {
		t.Fatalf("expected event label in list view, got:\n%s", view)
	}
}

func TestResample(t *testing.T) {
	tests := []struct {
		name      string
		values    []float64
		targetLen int
		wantLen   int
	}{
		{"empty", nil, 10, 0},
		{"zero target", []float64{1, 2, 3}, 0, 0},
		{"pad short", []float64{1, 2, 3}, 5, 5},
		{"downsample", []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 5, 5},
		{"exact", []float64{1, 2, 3}, 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resample(tt.values, tt.targetLen)
			if len(got) != tt.wantLen {
				t.Fatalf("resample(%v, %d) = len %d, want %d", tt.values, tt.targetLen, len(got), tt.wantLen)
			}
		})
	}
}
