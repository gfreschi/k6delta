package ecs

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"

	"github.com/gfreschi/k6delta/internal/config"
)

type mockECS struct {
	output *ecs.DescribeServicesOutput
	err    error
}

func (m *mockECS) DescribeServices(ctx context.Context, params *ecs.DescribeServicesInput, optFns ...func(*ecs.Options)) (*ecs.DescribeServicesOutput, error) {
	return m.output, m.err
}

type mockASG struct {
	output *autoscaling.DescribeAutoScalingGroupsOutput
	err    error
}

func (m *mockASG) DescribeAutoScalingGroups(ctx context.Context, params *autoscaling.DescribeAutoScalingGroupsInput, optFns ...func(*autoscaling.Options)) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {
	return m.output, m.err
}

func TestTakeSnapshot(t *testing.T) {
	ecsMock := &mockECS{
		output: &ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{
				{RunningCount: 3, DesiredCount: 4},
			},
		},
	}
	asgMock := &mockASG{
		output: &autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []astypes.AutoScalingGroup{
				{
					AutoScalingGroupName: aws.String("myapp-staging-ecs-abc123"),
					DesiredCapacity:      aws.Int32(2),
					Instances: []astypes.Instance{
						{InstanceId: aws.String("i-001"), LifecycleState: astypes.LifecycleStateInService},
						{InstanceId: aws.String("i-002"), LifecycleState: astypes.LifecycleStateInService},
						{InstanceId: aws.String("i-003"), LifecycleState: astypes.LifecycleStateTerminating},
					},
				},
			},
		},
	}

	p := &Provider{
		app: config.ResolvedApp{
			Cluster:   "myapp-staging",
			Service:   "myapp-web-staging",
			ASGPrefix: "myapp-staging-ecs-",
		},
	}

	snap, err := p.takeSnapshotWithClients(context.Background(), ecsMock, asgMock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.TaskRunning != 3 {
		t.Errorf("TaskRunning = %d, want 3", snap.TaskRunning)
	}
	if snap.TaskDesired != 4 {
		t.Errorf("TaskDesired = %d, want 4", snap.TaskDesired)
	}
	if snap.ASGName != "myapp-staging-ecs-abc123" {
		t.Errorf("ASGName = %q, want %q", snap.ASGName, "myapp-staging-ecs-abc123")
	}
	if snap.ASGDesired != 2 {
		t.Errorf("ASGDesired = %d, want 2", snap.ASGDesired)
	}
	if snap.ASGInstances != 2 {
		t.Errorf("ASGInstances = %d, want 2", snap.ASGInstances)
	}
}

func TestTakeSnapshotProgress(t *testing.T) {
	ecsMock := &mockECS{
		output: &ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{
				{RunningCount: 3, DesiredCount: 4},
			},
		},
	}
	asgMock := &mockASG{
		output: &autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []astypes.AutoScalingGroup{
				{
					AutoScalingGroupName: aws.String("myapp-staging-ecs-abc123"),
					DesiredCapacity:      aws.Int32(2),
					Instances: []astypes.Instance{
						{InstanceId: aws.String("i-001"), LifecycleState: astypes.LifecycleStateInService},
					},
				},
			},
		},
	}

	var progress []string
	p := &Provider{
		app: config.ResolvedApp{
			Cluster:   "myapp-staging",
			Service:   "myapp-web-staging",
			ASGPrefix: "myapp-staging-ecs-",
		},
		onProgress: func(id string, current, total int) {
			progress = append(progress, id)
		},
	}

	_, err := p.takeSnapshotWithClients(context.Background(), ecsMock, asgMock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progress) != 2 {
		t.Fatalf("progress calls = %d, want 2; got %v", len(progress), progress)
	}
	if progress[0] != "ECS tasks" {
		t.Errorf("progress[0] = %q, want %q", progress[0], "ECS tasks")
	}
	if progress[1] != "ASG instances" {
		t.Errorf("progress[1] = %q, want %q", progress[1], "ASG instances")
	}
}

func TestTakeSnapshotProgressNoASG(t *testing.T) {
	ecsMock := &mockECS{
		output: &ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{
				{RunningCount: 2, DesiredCount: 2},
			},
		},
	}
	asgMock := &mockASG{
		output: &autoscaling.DescribeAutoScalingGroupsOutput{},
	}

	var progress []string
	p := &Provider{
		app: config.ResolvedApp{
			Cluster: "myapp-staging",
			Service: "myapp-web-staging",
		},
		onProgress: func(id string, current, total int) {
			progress = append(progress, id)
		},
	}

	_, err := p.takeSnapshotWithClients(context.Background(), ecsMock, asgMock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progress) != 1 {
		t.Fatalf("progress calls = %d, want 1; got %v", len(progress), progress)
	}
	if progress[0] != "ECS tasks" {
		t.Errorf("progress[0] = %q, want %q", progress[0], "ECS tasks")
	}
}

func TestTakeSnapshotNoASG(t *testing.T) {
	ecsMock := &mockECS{
		output: &ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{
				{RunningCount: 2, DesiredCount: 2},
			},
		},
	}
	asgMock := &mockASG{
		output: &autoscaling.DescribeAutoScalingGroupsOutput{},
	}

	p := &Provider{
		app: config.ResolvedApp{
			Cluster: "myapp-staging",
			Service: "myapp-worker-staging",
			// ASGPrefix intentionally empty
		},
	}

	snap, err := p.takeSnapshotWithClients(context.Background(), ecsMock, asgMock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.TaskRunning != 2 {
		t.Errorf("TaskRunning = %d, want 2", snap.TaskRunning)
	}
	if snap.ASGName != "" {
		t.Errorf("ASGName = %q, want empty", snap.ASGName)
	}
	if snap.ASGInstances != 0 {
		t.Errorf("ASGInstances = %d, want 0", snap.ASGInstances)
	}
}
