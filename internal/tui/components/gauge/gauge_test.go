package gauge_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/gauge"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestGauge_rendersBar(t *testing.T) {
	ctx := tuictx.New(120, 40)
	g := gauge.NewModel(ctx, "CPU", 30)
	g.SetValue(72.0, 100.0)
	view := g.View()
	if !strings.Contains(view, "CPU") {
		t.Error("expected label 'CPU' in gauge view")
	}
	if !strings.Contains(view, "72%") {
		t.Error("expected '72%' in gauge view")
	}
}

func TestGauge_NoData(t *testing.T) {
	ctx := tuictx.New(120, 40)
	g := gauge.NewModel(ctx, "CPU", 30)

	view := g.View()
	if !strings.Contains(view, "—") {
		t.Error("gauge without data should show — indicator")
	}
}

func TestGauge_WithData(t *testing.T) {
	ctx := tuictx.New(120, 40)
	g := gauge.NewModel(ctx, "CPU", 30)
	g.SetValue(45.0, 100.0)

	view := g.View()
	if strings.Contains(view, "—") {
		t.Error("gauge with data should not show — indicator")
	}
}

func TestGauge_thresholdColoring(t *testing.T) {
	ctx := tuictx.New(120, 40)
	g := gauge.NewModel(ctx, "CPU", 30)

	// Below 70% — normal
	g.SetValue(50.0, 100.0)
	normal := g.View()

	// Above 95% — critical
	g.SetValue(96.0, 100.0)
	critical := g.View()

	// Views should differ (different fill + color)
	if normal == critical {
		t.Error("expected different rendering for normal vs critical")
	}
}

func TestGaugeThresholdBoundary(t *testing.T) {
	ctx := tuictx.New(80, 24)

	tests := []struct {
		name  string
		value float64
		max   float64
	}{
		{"below_95", 94.9, 100},
		{"exactly_95", 95.0, 100},
		{"above_95", 96.0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gauge.NewModel(ctx, "Test", 20)
			g.SetValue(tt.value, tt.max)
			view := g.View()
			if len(view) == 0 {
				t.Fatalf("gauge View() returned empty string for value=%.1f", tt.value)
			}
		})
	}
}
