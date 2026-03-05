package compose

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// mockDocker implements dockerPinger and containerLister for testing.
type mockDocker struct {
	pingErr    error
	containers []container.Summary
	listErr    error
}

func (m *mockDocker) Ping(ctx context.Context, opts client.PingOptions) (client.PingResult, error) {
	return client.PingResult{}, m.pingErr
}

func (m *mockDocker) ContainerList(ctx context.Context, opts client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: m.containers}, m.listErr
}
