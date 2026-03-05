package analyzetui

import (
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

	m := NewModel(app, prov, start.Format(time.RFC3339), end.Format(time.RFC3339), 10, false, "")

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
