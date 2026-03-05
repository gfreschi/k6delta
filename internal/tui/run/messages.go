package runtui

import (
	"github.com/gfreschi/k6delta/internal/k6"
	"github.com/gfreschi/k6delta/internal/provider"
	"github.com/gfreschi/k6delta/internal/report"
)

// ProgressMsg is sent by the provider's OnProgress callback via tea.Program.Send.
type ProgressMsg struct {
	ID      string
	Current int
	Total   int
}

type authOKMsg struct{}

type snapshotMsg struct {
	snapshot provider.Snapshot
	label    string
}

type k6DoneMsg struct{ result k6.RunResult }

type analysisMsg struct {
	metrics    []provider.MetricResult
	activities provider.Activities
}

type reportMsg struct {
	report *report.UnifiedReport
	path   string
}

type errMsg struct{ err error }

type k6PointMsg struct{ point k6.K6Point }

type exportDoneMsg struct{ path string }

type openDoneMsg struct{ path string }
