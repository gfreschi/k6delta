package compose

import (
	"context"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/events"
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

// mockStatter extends mockDocker with ContainerStats for metrics tests.
type mockStatter struct {
	mockDocker
	statsResults map[string]client.ContainerStatsResult
	statsErr     error
}

func (m *mockStatter) ContainerStats(ctx context.Context, containerID string, options client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	if m.statsErr != nil {
		return client.ContainerStatsResult{}, m.statsErr
	}
	if r, ok := m.statsResults[containerID]; ok {
		return r, nil
	}
	return client.ContainerStatsResult{}, m.statsErr
}

// mockEventEmitter implements dockerEventEmitter for activities tests.
type mockEventEmitter struct {
	messages []events.Message
	err      error
}

func (m *mockEventEmitter) Events(_ context.Context, _ client.EventsListOptions) client.EventsResult {
	msgCh := make(chan events.Message, len(m.messages))
	errCh := make(chan error, 1)
	if m.err != nil {
		// Error path: send error, close errCh, leave msgCh open (no messages).
		errCh <- m.err
		close(errCh)
	} else {
		// Success path: buffer messages, close msgCh, leave errCh open.
		for _, msg := range m.messages {
			msgCh <- msg
		}
		close(msgCh)
	}
	return client.EventsResult{
		Messages: msgCh,
		Err:      errCh,
	}
}
