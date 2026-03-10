package metriccard_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/metriccard"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestPercentageVariant(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 20)
	m.SetValue(72.0, 100.0)

	view := m.View()
	if !strings.Contains(view, "72%") {
		t.Errorf("expected '72%%' in view, got: %s", view)
	}
	if !strings.Contains(view, "CPU") {
		t.Error("expected label 'CPU' in view")
	}
}

func TestCountVariant(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "Tasks", "count", 20)
	m.SetValue(5, 10)
	m.SetDelta("5->3")

	view := m.View()
	if !strings.Contains(view, "5") {
		t.Errorf("expected '5' in count view, got: %s", view)
	}
	if !strings.Contains(view, "Tasks") {
		t.Error("expected label 'Tasks' in view")
	}
}

func TestRateVariant(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "RPS", "req/s", 20)
	m.SetValue(42.5, 100)

	view := m.View()
	if !strings.Contains(view, "42.5") {
		t.Errorf("expected '42.5' in rate view, got: %s", view)
	}
}

func TestNoData(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 20)

	view := m.View()
	if !strings.Contains(view, "—") {
		t.Error("expected no-data indicator")
	}
}

func TestBlockSparkline(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 20)
	for i := 0; i < 10; i++ {
		m.PushSparkline(float64(i * 10))
	}
	m.SetValue(90.0, 100.0)

	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view with sparkline data")
	}
}

func TestSeverityBorders(t *testing.T) {
	ctx := tuictx.New(120, 40)

	tests := []struct {
		name  string
		value float64
		max   float64
	}{
		{"ok", 50.0, 100.0},
		{"warn", 85.0, 100.0},
		{"error", 96.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metriccard.NewModel(ctx, "Test", "%", 20)
			m.SetValue(tt.value, tt.max)
			view := m.View()
			if len(view) == 0 {
				t.Fatalf("empty view for severity %s", tt.name)
			}
		})
	}
}

func TestSetReportData(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 20)
	m.SetReportData(85.0, 67.0, []float64{50, 60, 70, 80, 85})

	view := m.View()
	if !strings.Contains(view, "85%") {
		t.Errorf("expected peak '85%%' in report view, got: %s", view)
	}
}

func TestSetSeverityOverride(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "5xx", "count", 20)
	m.SetValue(0, 100)
	m.SetSeverity(metriccard.SeverityOK)

	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view")
	}
}

func TestCustomThresholds(t *testing.T) {
	ctx := tuictx.New(120, 40)

	tests := []struct {
		name       string
		value      float64
		max        float64
		thresholds metriccard.SeverityThresholds
	}{
		{"default warn at 80%", 82.0, 100.0, metriccard.DefaultSeverityThresholds()},
		{"custom warn at 50%", 52.0, 100.0, metriccard.SeverityThresholds{WarnRatio: 0.50, ErrorRatio: 0.90}},
		{"custom err at 70%", 72.0, 100.0, metriccard.SeverityThresholds{WarnRatio: 0.50, ErrorRatio: 0.70}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := metriccard.NewModel(ctx, "Test", "%", 20)
			m.SetThresholds(tt.thresholds)
			m.SetValue(tt.value, tt.max)
			view := m.View()
			if len(view) == 0 {
				t.Fatal("expected non-empty view with custom thresholds")
			}
		})
	}
}

func TestSetStale(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 14)
	m.SetValue(55.0, 100.0)
	m.SetStale(true)

	view := m.View()
	if view == "" {
		t.Error("stale tile rendered empty")
	}
	// Stale tile should still show the value
	if !strings.Contains(view, "55%") {
		t.Errorf("stale tile should still show value, got: %s", view)
	}
}

func TestPulseOnValueChange(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 14)
	m.SetValue(50.0, 100.0)

	// Change value -- should return a tick command for pulse
	cmd := m.SetValue(60.0, 100.0)
	if cmd == nil {
		t.Error("expected tick command for pulse, got nil")
	}

	// Same value -- no pulse
	cmd = m.SetValue(60.0, 100.0)
	if cmd != nil {
		t.Error("expected nil command for same value, got non-nil")
	}
}

func TestClearPulse(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 14)
	m.SetValue(50.0, 100.0)
	m.SetValue(60.0, 100.0) // triggers pulse

	m.ClearPulse()
	// After clear, view should still render (no panic)
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view after ClearPulse")
	}
}

func TestDirectionArrow(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 14)
	m.SetValue(50.0, 100.0)
	m.SetValue(60.0, 100.0) // increase

	view := m.View()
	if !strings.Contains(view, "\u25b2") { // IconArrowUp
		t.Errorf("expected up arrow after increase, got %q", view)
	}

	m.SetValue(40.0, 100.0) // decrease
	view = m.View()
	if !strings.Contains(view, "\u25bc") { // IconArrowDn
		t.Errorf("expected down arrow after decrease, got %q", view)
	}
}

func TestDirectionArrow_decays(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 14)
	m.SetValue(50.0, 100.0)
	m.SetValue(60.0, 100.0) // triggers arrow, TTL=2

	// First call with same value: TTL decrements to 1, arrow still visible
	m.SetValue(60.0, 100.0)
	view := m.View()
	if !strings.Contains(view, "\u25b2") {
		t.Errorf("expected arrow still visible after 1 same-value call, got %q", view)
	}

	// Second call with same value: TTL decrements to 0, arrow gone
	m.SetValue(60.0, 100.0)
	view = m.View()
	if strings.Contains(view, "\u25b2") {
		t.Errorf("expected arrow gone after 2 same-value calls, got %q", view)
	}
}

func TestUpdateContext(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := metriccard.NewModel(ctx, "CPU", "%", 20)

	ctx2 := tuictx.New(80, 24)
	m.UpdateContext(ctx2)
	m.SetValue(50.0, 100.0)

	view := m.View()
	if len(view) == 0 {
		t.Fatal("expected non-empty view after UpdateContext")
	}
}
