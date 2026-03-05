package compose

import (
	"context"
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestTakeSnapshot_countsContainers(t *testing.T) {
	mock := &mockDocker{
		containers: []container.Summary{
			{State: container.StateRunning},
			{State: container.StateRunning},
			{State: container.StateExited},
		},
	}

	p := &Provider{project: "myapp"}
	snap, err := p.takeSnapshotWithClient(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.TaskRunning != 2 {
		t.Errorf("TaskRunning = %d, want 2", snap.TaskRunning)
	}
	if snap.TaskDesired != 3 {
		t.Errorf("TaskDesired = %d, want 3 (total containers)", snap.TaskDesired)
	}
}

func TestTakeSnapshot_allRunning(t *testing.T) {
	mock := &mockDocker{
		containers: []container.Summary{
			{State: container.StateRunning},
			{State: container.StateRunning},
		},
	}

	p := &Provider{project: "myapp"}
	snap, err := p.takeSnapshotWithClient(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.TaskRunning != 2 {
		t.Errorf("TaskRunning = %d, want 2", snap.TaskRunning)
	}
	if snap.TaskDesired != 2 {
		t.Errorf("TaskDesired = %d, want 2", snap.TaskDesired)
	}
}

func TestTakeSnapshot_progressCallback(t *testing.T) {
	mock := &mockDocker{
		containers: []container.Summary{{State: container.StateRunning}},
	}

	var called bool
	p := &Provider{
		project: "myapp",
		onProgress: func(id string, current, total int) {
			called = true
		},
	}
	if _, err := p.takeSnapshotWithClient(context.Background(), mock); err != nil {
		t.Fatalf("takeSnapshotWithClient error: %v", err)
	}
	if !called {
		t.Error("expected progress callback to be called")
	}
}
