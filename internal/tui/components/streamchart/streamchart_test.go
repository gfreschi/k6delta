package streamchart_test

import (
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/components/streamchart"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestNewModel(t *testing.T) {
	ctx := tuictx.New(120, 40)
	m := streamchart.NewModel(ctx, "Throughput", "req/s", 60, 12)

	if m.View() == "" {
		t.Error("View() should return non-empty string for empty chart")
	}
}

func TestPush(t *testing.T) {
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

func TestResize(t *testing.T) {
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
