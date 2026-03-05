package compose

import (
	"context"
	"fmt"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/gfreschi/k6delta/internal/provider"
)

func (p *Provider) takeSnapshotWithClient(ctx context.Context, cli containerLister) (provider.Snapshot, error) {
	p.reportProgress("Containers", 1, 1)

	opts := client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", fmt.Sprintf("com.docker.compose.project=%s", p.project)),
	}

	result, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return provider.Snapshot{}, fmt.Errorf("list containers: %w", err)
	}

	var running int
	for _, c := range result.Items {
		if c.State == container.StateRunning {
			running++
		}
	}

	return provider.Snapshot{
		TaskRunning: running,
		TaskDesired: len(result.Items),
	}, nil
}
