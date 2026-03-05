package mock

import (
	"context"
	"time"

	"github.com/gfreschi/k6delta/internal/provider"
)

var _ provider.InfraProvider = (*Provider)(nil)

// Provider implements InfraProvider using synthetic scenario data.
type Provider struct {
	scenario      Scenario
	snapshotCount int
	onProgress    func(id string, current, total int)
}

// New creates a mock provider for the given scenario name.
func New(scenarioName string) (*Provider, error) {
	s, err := GetScenario(scenarioName)
	if err != nil {
		return nil, err
	}
	return &Provider{scenario: s}, nil
}

// SetOnProgress sets the progress callback for TUI integration.
func (p *Provider) SetOnProgress(fn func(id string, current, total int)) {
	p.onProgress = fn
}

func (p *Provider) reportProgress(id string, current, total int) {
	if p.onProgress != nil {
		p.onProgress(id, current, total)
	}
}

// CheckCredentials always succeeds for mock provider.
func (p *Provider) CheckCredentials(_ context.Context) error {
	p.reportProgress("Credentials", 1, 1)
	return nil
}

// TakeSnapshot returns pre-snapshot on first call, post-snapshot on subsequent calls.
func (p *Provider) TakeSnapshot(_ context.Context) (provider.Snapshot, error) {
	p.reportProgress("Snapshot", 1, 1)
	p.snapshotCount++
	if p.snapshotCount <= 1 {
		return p.scenario.PreSnapshot, nil
	}
	return p.scenario.PostSnapshot, nil
}

// FetchMetrics samples scenario generators at period-second intervals.
func (p *Provider) FetchMetrics(_ context.Context, start, end time.Time, period int32) ([]provider.MetricResult, error) {
	duration := end.Sub(start)
	n := int(duration.Seconds() / float64(period))
	if n < 1 {
		n = 1
	}

	p.reportProgress("Metrics", 1, len(p.scenario.Metrics))

	results := make([]provider.MetricResult, 0, len(p.scenario.Metrics))
	for i, mc := range p.scenario.Metrics {
		values := Sample(mc.Generator, n)
		timestamps := make([]time.Time, n)
		for j := range n {
			timestamps[j] = start.Add(time.Duration(j) * time.Duration(period) * time.Second)
		}

		peak, avg := computeStats(values)
		results = append(results, provider.MetricResult{
			ID:         mc.ID,
			Values:     values,
			Timestamps: timestamps,
			Peak:       &peak,
			Avg:        &avg,
		})

		p.reportProgress("Metrics", i+1, len(p.scenario.Metrics))
	}

	return results, nil
}

// FetchActivities returns templated events mapped to absolute timestamps.
func (p *Provider) FetchActivities(_ context.Context, start, end time.Time) (provider.Activities, error) {
	p.reportProgress("Activities", 1, 1)
	duration := end.Sub(start)

	var scaling []provider.ScalingActivity
	for _, a := range p.scenario.Activities {
		t := start.Add(time.Duration(float64(duration) * a.AtNormalized))
		scaling = append(scaling, provider.ScalingActivity{
			Time:        t.Format(time.RFC3339),
			Status:      a.Status,
			Description: a.Description,
		})
	}

	var alarms []provider.AlarmEvent
	for _, a := range p.scenario.Alarms {
		t := start.Add(time.Duration(float64(duration) * a.AtNormalized))
		alarms = append(alarms, provider.AlarmEvent{
			AlarmName: a.AlarmName,
			Time:      t.Format(time.RFC3339),
			OldState:  a.OldState,
			NewState:  a.NewState,
		})
	}

	return provider.Activities{
		ServiceScaling: scaling,
		Alarms:         alarms,
	}, nil
}

func computeStats(values []float64) (peak, avg float64) {
	if len(values) == 0 {
		return 0, 0
	}
	peak = values[0]
	sum := 0.0
	for _, v := range values {
		sum += v
		if v > peak {
			peak = v
		}
	}
	avg = sum / float64(len(values))
	return peak, avg
}
