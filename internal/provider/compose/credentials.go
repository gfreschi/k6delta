package compose

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

// dockerPinger abstracts Docker ping for testability.
type dockerPinger interface {
	Ping(ctx context.Context, opts client.PingOptions) (client.PingResult, error)
}

// containerLister abstracts Docker container listing for testability.
type containerLister interface {
	ContainerList(ctx context.Context, opts client.ContainerListOptions) (client.ContainerListResult, error)
}

func (p *Provider) checkCredentialsWithClient(ctx context.Context, cli interface {
	dockerPinger
	containerLister
}) error {
	if _, err := cli.Ping(ctx, client.PingOptions{}); err != nil {
		return fmt.Errorf("docker daemon not reachable: %w", err)
	}

	opts := client.ContainerListOptions{
		All:     true,
		Filters: make(client.Filters).Add("label", fmt.Sprintf("com.docker.compose.project=%s", p.project)),
	}
	result, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return fmt.Errorf("list containers: %w", err)
	}
	if len(result.Items) == 0 {
		return fmt.Errorf("no containers found for compose project %q — is the project running?", p.project)
	}

	return nil
}
