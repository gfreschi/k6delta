package linechart_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/linechart"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestLineChart_rendersWithData(t *testing.T) {
	ctx := tuictx.New(120, 40)
	lc := linechart.NewModel(ctx, "Throughput", "req/s", 40, 8)
	lc.AddPoint(100)
	lc.AddPoint(200)
	lc.AddPoint(150)
	lc.AddPoint(250)

	view := lc.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
	if !strings.Contains(view, "Throughput") {
		t.Error("expected title in view")
	}
}

func TestLineChart_emptyData(t *testing.T) {
	ctx := tuictx.New(120, 40)
	lc := linechart.NewModel(ctx, "Latency", "ms", 40, 8)
	view := lc.View()
	if view == "" {
		t.Error("expected non-empty view even without data")
	}
}

func TestLineChart_autoScalesYAxis(t *testing.T) {
	ctx := tuictx.New(120, 40)
	lc := linechart.NewModel(ctx, "RPS", "req/s", 40, 8)
	for i := 0; i < 20; i++ {
		lc.AddPoint(float64(i * 50))
	}
	view := lc.View()
	// Should not panic and should produce content
	if view == "" {
		t.Error("expected non-empty view with scaled data")
	}
}
