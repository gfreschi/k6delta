package analyzetui

import "github.com/gfreschi/k6delta/internal/provider"

// ProgressMsg is sent by the provider's OnProgress callback via tea.Program.Send.
type ProgressMsg struct {
	ID      string
	Current int
	Total   int
}

type authOKMsg struct{}
type stateDoneMsg struct{ snapshot provider.Snapshot }
type metricsDoneMsg struct{ metrics []provider.MetricResult }
type activitiesDoneMsg struct{ activities provider.Activities }
type errMsg struct{ err error }
