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
)

// Provider implements provider.InfraProvider for AWS ECS.
type Provider struct {
	cfg         aws.Config
	app         k6config.ResolvedApp
	asgName     string
	asgResolved bool
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
