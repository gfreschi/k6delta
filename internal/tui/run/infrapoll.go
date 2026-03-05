package runtui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/provider"
)

// infraTickMsg carries a fresh infrastructure snapshot and metrics.
type infraTickMsg struct {
	snapshot provider.Snapshot
	metrics  []provider.MetricResult
	err      error
}

// infraPollCmd returns a tea.Cmd that polls infra after the given interval.
func infraPollCmd(ctx context.Context, prov provider.InfraProvider, interval time.Duration) tea.Cmd {
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		snap, snapErr := prov.TakeSnapshot(ctx)
		now := time.Now()
		metrics, metricsErr := prov.FetchMetrics(ctx, now.Add(-2*interval), now, int32(interval.Seconds()))

		var err error
		if snapErr != nil {
			err = fmt.Errorf("infra snapshot: %w", snapErr)
		} else if metricsErr != nil {
			err = fmt.Errorf("infra metrics: %w", metricsErr)
		}

		return infraTickMsg{snapshot: snap, metrics: metrics, err: err}
	})
}
