package ecs

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"

	"github.com/gfreschi/k6delta/internal/provider"
)

// ECSDescriber abstracts the ECS DescribeServices API for testing.
type ECSDescriber interface {
	DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error)
}

// ASGDescriber abstracts the autoscaling DescribeAutoScalingGroups API for testing.
type ASGDescriber interface {
	DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error)
}

// TakeSnapshot captures current ECS service and ASG state.
func (p *Provider) TakeSnapshot(ctx context.Context) (provider.Snapshot, error) {
	ecsClient := ecs.NewFromConfig(p.cfg)
	asgClient := autoscaling.NewFromConfig(p.cfg)
	snap, err := p.takeSnapshotWithClients(ctx, ecsClient, asgClient)
	if err == nil && !p.asgResolved && snap.ASGName != "" {
		p.asgName = snap.ASGName
		p.asgResolved = true
	}
	return snap, err
}

func (p *Provider) takeSnapshotWithClients(ctx context.Context, ecsClient ECSDescriber, asgClient ASGDescriber) (provider.Snapshot, error) {
	snap := provider.Snapshot{}

	totalSteps := 1
	if p.app.ASGPrefix != "" {
		totalSteps = 2
	}

	// ECS service counts
	p.reportProgress("ECS tasks", 1, totalSteps)
	out, err := ecsClient.DescribeServices(ctx, &ecs.DescribeServicesInput{
		Cluster:  &p.app.Cluster,
		Services: []string{p.app.Service},
	})
	if err != nil {
		return snap, fmt.Errorf("describe services: %w", err)
	}
	if len(out.Services) == 0 {
		return snap, fmt.Errorf("no services found for %s/%s", p.app.Cluster, p.app.Service)
	}
	svc := out.Services[0]
	snap.TaskRunning = int(svc.RunningCount)
	snap.TaskDesired = int(svc.DesiredCount)

	// ASG counts (optional)
	if p.app.ASGPrefix == "" {
		return snap, nil
	}

	p.reportProgress("ASG instances", 2, 2)
	asgName, err := discoverASGName(ctx, asgClient, p.app.ASGPrefix)
	if err != nil {
		return snap, err
	}
	if asgName == "" {
		return snap, nil
	}
	snap.ASGName = asgName

	asgOut, err := asgClient.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []string{asgName},
	})
	if err != nil {
		return snap, fmt.Errorf("describe auto scaling groups: %w", err)
	}
	if len(asgOut.AutoScalingGroups) == 0 {
		return snap, nil
	}
	asg := asgOut.AutoScalingGroups[0]
	if asg.DesiredCapacity != nil {
		snap.ASGDesired = int(*asg.DesiredCapacity)
	}
	for _, inst := range asg.Instances {
		if inst.LifecycleState == astypes.LifecycleStateInService {
			snap.ASGInstances++
		}
	}

	return snap, nil
}

func discoverASGName(ctx context.Context, client ASGDescriber, prefix string) (string, error) {
	out, err := client.DescribeAutoScalingGroups(ctx, &autoscaling.DescribeAutoScalingGroupsInput{})
	if err != nil {
		return "", fmt.Errorf("describe auto scaling groups: %w", err)
	}
	for _, asg := range out.AutoScalingGroups {
		if asg.AutoScalingGroupName != nil && strings.HasPrefix(*asg.AutoScalingGroupName, prefix) {
			return *asg.AutoScalingGroupName, nil
		}
	}
	return "", nil
}
