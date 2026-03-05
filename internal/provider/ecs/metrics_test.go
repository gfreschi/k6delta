package ecs

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/autoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
	ecstypes "github.com/aws/aws-sdk-go-v2/service/ecs/types"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"
	elbtypes "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2/types"

	"github.com/gfreschi/k6delta/internal/config"
)

type mockCW struct {
	output *cloudwatch.GetMetricDataOutput
	err    error
}

func (m *mockCW) GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	return m.output, m.err
}

type mockELB struct {
	output *elbv2.DescribeTargetGroupsOutput
	err    error
}

func (m *mockELB) DescribeTargetGroups(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error) {
	return m.output, m.err
}

func TestBuildMetricQueries_AllFields(t *testing.T) {
	queries := buildMetricQueries(
		"my-cluster", "my-service", "my-asg",
		"arn:aws:elasticloadbalancing:eu-west-1:123:targetgroup/my-tg/abc",
		"app/my-alb/def", "my-cp", 60,
	)
	if len(queries) != 11 {
		t.Fatalf("got %d queries, want 11", len(queries))
	}
	expectedIDs := []string{
		"service_cpu", "service_memory", "cluster_cpu_reservation", "cluster_memory_reservation",
		"capacity_provider_reservation", "asg_desired", "asg_in_service",
		"alb_requests_per_target", "alb_response_time", "alb_5xx", "alb_healthy_hosts",
	}
	for i, q := range queries {
		if aws.ToString(q.Id) != expectedIDs[i] {
			t.Errorf("query[%d].Id = %q, want %q", i, aws.ToString(q.Id), expectedIDs[i])
		}
	}
}

func TestBuildMetricQueries_Minimal(t *testing.T) {
	queries := buildMetricQueries("my-cluster", "my-service", "", "", "", "", 300)
	if len(queries) != 4 {
		t.Fatalf("got %d queries, want 4", len(queries))
	}
}

func TestFetchMetricsProgress(t *testing.T) {
	t1 := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 3, 10, 1, 0, 0, time.UTC)

	cwMock := &mockCW{
		output: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []cwtypes.MetricDataResult{
				{Id: aws.String("service_cpu"), Values: []float64{10.0}, Timestamps: []time.Time{t1}},
			},
		},
	}
	ecsMock := &mockECS{
		output: &ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{{LoadBalancers: []ecstypes.LoadBalancer{}}},
		},
	}
	elbMock := &mockELB{
		output: &elbv2.DescribeTargetGroupsOutput{
			TargetGroups: []elbtypes.TargetGroup{},
		},
	}
	asgMock := &mockASG{
		output: &autoscaling.DescribeAutoScalingGroupsOutput{},
	}

	var progress []string
	p := &Provider{
		app: config.ResolvedApp{Cluster: "c", Service: "s"},
		onProgress: func(id string, current, total int) {
			progress = append(progress, id)
		},
	}

	_, err := p.fetchMetricsWithClients(context.Background(), cwMock, ecsMock, elbMock, asgMock, t1, t2, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progress) < 2 {
		t.Fatalf("progress calls = %d, want >= 2; got %v", len(progress), progress)
	}
	if progress[0] != "discovering resources" {
		t.Errorf("progress[0] = %q, want %q", progress[0], "discovering resources")
	}
	if progress[1] != "querying CloudWatch" {
		t.Errorf("progress[1] = %q, want %q", progress[1], "querying CloudWatch")
	}
}

func TestFetchMetrics(t *testing.T) {
	t1 := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 3, 10, 1, 0, 0, time.UTC)

	cwMock := &mockCW{
		output: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []cwtypes.MetricDataResult{
				{
					Id:         aws.String("service_cpu"),
					Values:     []float64{10.0, 30.0, 20.0},
					Timestamps: []time.Time{t1, t2, t1},
				},
			},
		},
	}
	ecsMock := &mockECS{
		output: &ecs.DescribeServicesOutput{
			Services: []ecstypes.Service{{LoadBalancers: []ecstypes.LoadBalancer{}}},
		},
	}
	elbMock := &mockELB{
		output: &elbv2.DescribeTargetGroupsOutput{
			TargetGroups: []elbtypes.TargetGroup{},
		},
	}
	asgMock := &mockASG{
		output: &autoscaling.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []astypes.AutoScalingGroup{},
		},
	}

	p := &Provider{
		app: config.ResolvedApp{Cluster: "c", Service: "s"},
	}

	results, err := p.fetchMetricsWithClients(context.Background(), cwMock, ecsMock, elbMock, asgMock, t1, t2, 60)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Peak == nil || *r.Peak != 30.0 {
		t.Errorf("Peak = %v, want 30.0", r.Peak)
	}
	if r.Avg == nil || *r.Avg != 20.0 {
		t.Errorf("Avg = %v, want 20.0", r.Avg)
	}
}
