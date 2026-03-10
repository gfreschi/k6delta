package analyzetui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/provider/mock"
	"github.com/gfreschi/k6delta/internal/tui/golden"
	"github.com/gfreschi/k6delta/internal/tui/testutil"
)

func TestAnalyzeModel_goldenHappyPath(t *testing.T) {
	prov, err := mock.New("happy-path")
	if err != nil {
		t.Fatal(err)
	}

	app := testutil.ResolvedApp()
	start := testutil.ReferenceTime
	end := start.Add(60 * time.Second)

	m := NewModel(app, prov, start.Format(time.RFC3339), end.Format(time.RFC3339), 10, "")

	// Set terminal size
	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Drive through phases with deterministic data (bypass provider's Noise generators).
	// Phase 1: Auth
	model, _ = model.Update(authOKMsg{})

	// Phase 2: State
	model, _ = model.Update(stateDoneMsg{snapshot: testutil.SampleSnapshot()})

	// Phase 3: Metrics
	model, _ = model.Update(metricsDoneMsg{metrics: testutil.SampleMetrics()})

	// Phase 4: Activities
	model, _ = model.Update(activitiesDoneMsg{activities: testutil.SampleActivities()})

	out := model.View()
	golden.RequireEqual(t, []byte(out))
}

func TestAnalyzeModel_goldenHappyPath_stacked(t *testing.T) {
	prov, err := mock.New("happy-path")
	if err != nil {
		t.Fatal(err)
	}

	app := testutil.ResolvedApp()
	start := testutil.ReferenceTime
	end := start.Add(60 * time.Second)

	m := NewModel(app, prov, start.Format(time.RFC3339), end.Format(time.RFC3339), 10, "")

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	model, _ = model.Update(authOKMsg{})
	model, _ = model.Update(stateDoneMsg{snapshot: testutil.SampleSnapshot()})
	model, _ = model.Update(metricsDoneMsg{metrics: testutil.SampleMetrics()})
	model, _ = model.Update(activitiesDoneMsg{activities: testutil.SampleActivities()})

	out := model.View()
	golden.RequireEqual(t, []byte(out))
}

func TestAnalyzeModel_refreshCountdown(t *testing.T) {
	prov, err := mock.New("happy-path")
	if err != nil {
		t.Fatal(err)
	}

	app := testutil.ResolvedApp()
	start := testutil.ReferenceTime
	end := start.Add(60 * time.Second)

	m := NewModel(app, prov, start.Format(time.RFC3339), end.Format(time.RFC3339), 10, "", 30)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Drive to phaseDisplay
	model, _ = model.Update(authOKMsg{})
	model, _ = model.Update(stateDoneMsg{snapshot: testutil.SampleSnapshot()})
	model, _ = model.Update(metricsDoneMsg{metrics: testutil.SampleMetrics()})
	model, _ = model.Update(activitiesDoneMsg{activities: testutil.SampleActivities()})

	analyzeModel := model.(Model)
	if analyzeModel.refreshCountdown != 30 {
		t.Errorf("refreshCountdown = %d, want 30", analyzeModel.refreshCountdown)
	}

	// Tick should decrement
	model, _ = model.Update(refreshTickMsg{})
	analyzeModel = model.(Model)
	if analyzeModel.refreshCountdown != 29 {
		t.Errorf("after tick: refreshCountdown = %d, want 29", analyzeModel.refreshCountdown)
	}

	// View should show countdown
	out := model.View()
	if !strings.Contains(out, "29s") {
		t.Error("view should show refresh countdown")
	}
}

func TestAnalyzeModel_refreshDisabled(t *testing.T) {
	prov, err := mock.New("happy-path")
	if err != nil {
		t.Fatal(err)
	}

	app := testutil.ResolvedApp()
	start := testutil.ReferenceTime
	end := start.Add(60 * time.Second)

	m := NewModel(app, prov, start.Format(time.RFC3339), end.Format(time.RFC3339), 10, "", 0)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	model, _ = model.Update(authOKMsg{})
	model, _ = model.Update(stateDoneMsg{snapshot: testutil.SampleSnapshot()})
	model, _ = model.Update(metricsDoneMsg{metrics: testutil.SampleMetrics()})
	model, _ = model.Update(activitiesDoneMsg{activities: testutil.SampleActivities()})

	analyzeModel := model.(Model)
	if analyzeModel.refreshCountdown != 0 {
		t.Errorf("refreshCountdown = %d, want 0 when disabled", analyzeModel.refreshCountdown)
	}

	// refreshTickMsg should be a no-op
	model, cmd := model.Update(refreshTickMsg{})
	if cmd != nil {
		t.Error("refreshTickMsg should return nil cmd when refresh disabled")
	}

	// View should NOT show refresh countdown
	out := model.View()
	if strings.Contains(out, "refresh") {
		t.Error("view should not show refresh when disabled")
	}
}

func TestAnalyzeModel_refreshDataUpdates(t *testing.T) {
	prov, err := mock.New("happy-path")
	if err != nil {
		t.Fatal(err)
	}

	app := testutil.ResolvedApp()
	start := testutil.ReferenceTime
	end := start.Add(60 * time.Second)

	m := NewModel(app, prov, start.Format(time.RFC3339), end.Format(time.RFC3339), 10, "", 30)

	var model tea.Model = m
	model, _ = model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	model, _ = model.Update(authOKMsg{})
	model, _ = model.Update(stateDoneMsg{snapshot: testutil.SampleSnapshot()})
	model, _ = model.Update(metricsDoneMsg{metrics: testutil.SampleMetrics()})
	model, _ = model.Update(activitiesDoneMsg{activities: testutil.SampleActivities()})

	// Simulate refresh data arriving
	newSnapshot := testutil.SampleSnapshot()
	newMetrics := testutil.SampleMetrics()
	newActivities := testutil.SampleActivities()

	model, cmd := model.Update(refreshDataMsg{
		snapshot:   newSnapshot,
		metrics:    newMetrics,
		activities: newActivities,
	})

	// Should restart countdown and schedule next tick
	analyzeModel := model.(Model)
	if analyzeModel.refreshCountdown != 30 {
		t.Errorf("after refresh: refreshCountdown = %d, want 30", analyzeModel.refreshCountdown)
	}
	if cmd == nil {
		t.Error("refreshDataMsg should return a tick cmd")
	}
}
