// Package ecs implements the InfraProvider interface for AWS ECS,
// collecting metrics from CloudWatch, ECS, ASG, and ALB.
package ecs

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"

	k6config "github.com/gfreschi/k6delta/internal/config"
	"github.com/gfreschi/k6delta/internal/provider"
)

var _ provider.InfraProvider = (*Provider)(nil)

// Provider implements provider.InfraProvider for AWS ECS.
type Provider struct {
	cfg         aws.Config
	app         k6config.ResolvedApp
	asgName     string
	asgResolved bool
	onProgress  func(id string, current, total int)
}

// SetOnProgress sets a callback that receives progress updates during
// long-running operations like FetchMetrics and TakeSnapshot.
func (p *Provider) SetOnProgress(fn func(id string, current, total int)) {
	p.onProgress = fn
}

// reportProgress calls the progress callback if set.
func (p *Provider) reportProgress(id string, current, total int) {
	if p.onProgress != nil {
		p.onProgress(id, current, total)
	}
}

// New creates a new ECS provider from a resolved app config.
func New(ctx context.Context, app k6config.ResolvedApp) (*Provider, error) {
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(app.Region))
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}
	return &Provider{cfg: cfg, app: app}, nil
}

// CheckCredentials verifies AWS credentials via STS GetCallerIdentity.
func (p *Provider) CheckCredentials(ctx context.Context) error {
	client := sts.NewFromConfig(p.cfg)
	_, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return fmt.Errorf("AWS credentials check failed: %w", err)
	}
	return nil
}
