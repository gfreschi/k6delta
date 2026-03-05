package mock_test

import (
	"context"
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/provider/mock"
)

// Compile-time interface check.
var _ provider.InfraProvider = (*mock.Provider)(nil)

func TestNew_validScenario(t *testing.T) {
	p, err := mock.New("happy-path")
	if err != nil {
		t.Fatalf("New(happy-path) error: %v", err)
	}
	if p == nil {
		t.Fatal("New returned nil provider")
	}
}

func TestNew_unknownScenario(t *testing.T) {
	_, err := mock.New("nonexistent")
	if err == nil {
		t.Error("expected error for unknown scenario")
	}
}

func TestCheckCredentials(t *testing.T) {
	p, _ := mock.New("happy-path")
	if err := p.CheckCredentials(context.Background()); err != nil {
		t.Errorf("CheckCredentials error: %v", err)
	}
}

func TestTakeSnapshot_preAndPost(t *testing.T) {
	p, _ := mock.New("scale-out")

	pre, err := p.TakeSnapshot(context.Background())
	if err != nil {
		t.Fatalf("first TakeSnapshot error: %v", err)
	}
	if pre.TaskRunning != 2 {
		t.Errorf("pre TaskRunning = %d, want 2", pre.TaskRunning)
	}

	post, err := p.TakeSnapshot(context.Background())
	if err != nil {
		t.Fatalf("second TakeSnapshot error: %v", err)
	}
	if post.TaskRunning != 5 {
		t.Errorf("post TaskRunning = %d, want 5", post.TaskRunning)
	}
}

func TestFetchMetrics(t *testing.T) {
	p, _ := mock.New("happy-path")
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	end := start.Add(60 * time.Second)

	metrics, err := p.FetchMetrics(context.Background(), start, end, 10)
	if err != nil {
		t.Fatalf("FetchMetrics error: %v", err)
	}

	if len(metrics) == 0 {
		t.Fatal("FetchMetrics returned no metrics")
	}

	ids := make(map[string]bool)
	for _, m := range metrics {
		ids[m.ID] = true
		if m.Peak == nil {
			t.Errorf("metric %s has nil Peak", m.ID)
		}
		if m.Avg == nil {
			t.Errorf("metric %s has nil Avg", m.ID)
		}
		if len(m.Values) == 0 {
			t.Errorf("metric %s has no values", m.ID)
		}
		if len(m.Timestamps) != len(m.Values) {
			t.Errorf("metric %s: %d timestamps vs %d values", m.ID, len(m.Timestamps), len(m.Values))
		}
	}

	if !ids["service_cpu"] {
		t.Error("missing service_cpu metric")
	}
}

func TestFetchActivities_happyPath(t *testing.T) {
	p, _ := mock.New("happy-path")
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	end := start.Add(60 * time.Second)

	acts, err := p.FetchActivities(context.Background(), start, end)
	if err != nil {
		t.Fatalf("FetchActivities error: %v", err)
	}
	if len(acts.ServiceScaling) != 0 {
		t.Errorf("happy-path should have 0 activities, got %d", len(acts.ServiceScaling))
	}
}

func TestFetchActivities_scaleOut(t *testing.T) {
	p, _ := mock.New("scale-out")
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Second)

	acts, err := p.FetchActivities(context.Background(), start, end)
	if err != nil {
		t.Fatalf("FetchActivities error: %v", err)
	}
	if len(acts.ServiceScaling) != 3 {
		t.Errorf("scale-out should have 3 activities, got %d", len(acts.ServiceScaling))
	}
}

func TestFetchActivities_cascadeHasAlarms(t *testing.T) {
	p, _ := mock.New("cascade-failure")
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	end := start.Add(60 * time.Second)

	acts, err := p.FetchActivities(context.Background(), start, end)
	if err != nil {
		t.Fatalf("FetchActivities error: %v", err)
	}
	if len(acts.Alarms) != 2 {
		t.Errorf("cascade-failure should have 2 alarms, got %d", len(acts.Alarms))
	}
}

func TestSetOnProgress(t *testing.T) {
	p, _ := mock.New("happy-path")
	var called bool
	p.SetOnProgress(func(id string, current, total int) {
		called = true
	})
	if _, err := p.TakeSnapshot(context.Background()); err != nil {
		t.Fatalf("TakeSnapshot error: %v", err)
	}
	if !called {
		t.Error("progress callback not called")
	}
}
