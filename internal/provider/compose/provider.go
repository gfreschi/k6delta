// Package compose implements InfraProvider for Docker Compose environments.
package compose

import (
	"context"
	"fmt"
	"time"

	"github.com/moby/moby/client"

	"github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
)

var _ provider.InfraProvider = (*Provider)(nil)

// Provider monitors Docker Compose services during load tests.
type Provider struct {
	app        config.ResolvedApp
	project    string
	docker     client.APIClient
	onProgress func(id string, current, total int)
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

// New creates a Provider for Docker Compose monitoring.
func New(app config.ResolvedApp, project string) (*Provider, error) {
	if project == "" {
		return nil, fmt.Errorf("compose_project is required for docker-compose provider")
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &Provider{
		app:     app,
		project: project,
		docker:  cli,
	}, nil
}

// CheckCredentials verifies Docker is reachable and containers exist for the project.
func (p *Provider) CheckCredentials(ctx context.Context) error {
	return p.checkCredentialsWithClient(ctx, p.docker)
}

// TakeSnapshot captures current container state for the compose project.
func (p *Provider) TakeSnapshot(ctx context.Context) (provider.Snapshot, error) {
	return p.takeSnapshotWithClient(ctx, p.docker)
}

// FetchMetrics collects CPU and memory from container stats.
func (p *Provider) FetchMetrics(ctx context.Context, start, end time.Time, period int32) ([]provider.MetricResult, error) {
	return p.fetchMetricsWithClient(ctx, p.docker)
}

// FetchActivities captures container start/stop/restart events.
func (p *Provider) FetchActivities(ctx context.Context, start, end time.Time) (provider.Activities, error) {
	return p.fetchActivitiesWithClient(ctx, p.docker, start, end)
}
