package compose

import (
	"context"
	"fmt"
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestCheckCredentials_dockerDown(t *testing.T) {
	p := &Provider{project: "myapp"}
	mock := &mockDocker{pingErr: fmt.Errorf("connection refused")}
	err := p.checkCredentialsWithClient(context.Background(), mock)
	if err == nil {
		t.Error("expected error when Docker is unreachable")
	}
}

func TestCheckCredentials_noContainers(t *testing.T) {
	p := &Provider{project: "myapp"}
	mock := &mockDocker{containers: []container.Summary{}}
	err := p.checkCredentialsWithClient(context.Background(), mock)
	if err == nil {
		t.Error("expected error when no containers found for project")
	}
}

func TestCheckCredentials_success(t *testing.T) {
	p := &Provider{project: "myapp"}
	mock := &mockDocker{
		containers: []container.Summary{{State: container.StateRunning}},
	}
	err := p.checkCredentialsWithClient(context.Background(), mock)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
