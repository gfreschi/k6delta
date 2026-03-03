package ecs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	autoscalingapi "github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	ecsapi "github.com/aws/aws-sdk-go-v2/service/ecs"
	elbv2 "github.com/aws/aws-sdk-go-v2/service/elasticloadbalancingv2"

	"github.com/gfreschi/k6delta/internal/provider"
)

// CloudWatchGetter abstracts the CloudWatch GetMetricData API for testing.
type CloudWatchGetter interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
}

// ELBDescriber abstracts the ELBv2 DescribeTargetGroups API for testing.
type ELBDescriber interface {
	DescribeTargetGroups(ctx context.Context, params *elbv2.DescribeTargetGroupsInput, optFns ...func(*elbv2.Options)) (*elbv2.DescribeTargetGroupsOutput, error)
}

// FetchMetrics queries CloudWatch for infrastructure metrics during the test window.
func (p *Provider) FetchMetrics(ctx context.Context, start, end time.Time, period int32) ([]provider.MetricResult, error) {
	ecsClient := ecsapi.NewFromConfig(p.cfg)
	elbClient := elbv2.NewFromConfig(p.cfg)
	cwClient := cloudwatch.NewFromConfig(p.cfg)
	asgClient := autoscalingapi.NewFromConfig(p.cfg)
	return p.fetchMetricsWithClients(ctx, cwClient, ecsClient, elbClient, asgClient, start, end, period)
}

func (p *Provider) fetchMetricsWithClients(ctx context.Context, cwClient CloudWatchGetter, ecsClient ECSDescriber, elbClient ELBDescriber, asgClient ASGDescriber, start, end time.Time, period int32) ([]provider.MetricResult, error) {
	// Discover ELB resources for ALB metrics
	var tgARN, albSuffix string
	tgARN, _ = discoverTargetGroup(ctx, ecsClient, p.app.Cluster, p.app.Service)
	if tgARN != "" {
		albSuffix, _ = discoverALBSuffix(ctx, elbClient, tgARN)
	}

	// Use cached ASG name if available, otherwise discover
	var asgName string
	if p.asgResolved {
		asgName = p.asgName
	} else if p.app.ASGPrefix != "" {
		asgName, _ = discoverASGName(ctx, asgClient, p.app.ASGPrefix)
	}

	queries := buildMetricQueries(p.app.Cluster, p.app.Service, asgName, tgARN, albSuffix, p.app.CapacityProvider, period)

	out, err := cwClient.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		MetricDataQueries: queries,
		StartTime:         &start,
		EndTime:           &end,
	})
	if err != nil {
		return nil, fmt.Errorf("get metric data: %w", err)
	}

	var results []provider.MetricResult
	for _, r := range out.MetricDataResults {
		mr := provider.MetricResult{
			ID:         aws.ToString(r.Id),
			Values:     r.Values,
			Timestamps: r.Timestamps,
		}
		if len(r.Values) > 0 {
			peak := r.Values[0]
			sum := 0.0
			for _, v := range r.Values {
				sum += v
				if v > peak {
					peak = v
				}
			}
			avg := sum / float64(len(r.Values))
			mr.Peak = &peak
			mr.Avg = &avg
		}
		results = append(results, mr)
	}
	return results, nil
}

// discoverTargetGroup finds the target group ARN from the ECS service's load balancer.
func discoverTargetGroup(ctx context.Context, client ECSDescriber, cluster, service string) (string, error) {
	out, err := client.DescribeServices(ctx, &ecsapi.DescribeServicesInput{
		Cluster:  &cluster,
		Services: []string{service},
	})
	if err != nil {
		return "", fmt.Errorf("describe services: %w", err)
	}
	if len(out.Services) == 0 {
		return "", nil
	}
	svc := out.Services[0]
	if len(svc.LoadBalancers) == 0 || svc.LoadBalancers[0].TargetGroupArn == nil {
		return "", nil
	}
	return *svc.LoadBalancers[0].TargetGroupArn, nil
}

// discoverALBSuffix extracts the ALB suffix from a target group's load balancer ARN.
func discoverALBSuffix(ctx context.Context, client ELBDescriber, tgARN string) (string, error) {
	out, err := client.DescribeTargetGroups(ctx, &elbv2.DescribeTargetGroupsInput{
		TargetGroupArns: []string{tgARN},
	})
	if err != nil {
		return "", fmt.Errorf("describe target groups: %w", err)
	}
	if len(out.TargetGroups) == 0 || len(out.TargetGroups[0].LoadBalancerArns) == 0 {
		return "", nil
	}
	albARN := out.TargetGroups[0].LoadBalancerArns[0]
	const marker = "loadbalancer/"
	idx := strings.Index(albARN, marker)
	if idx == -1 {
		return "", nil
	}
	return albARN[idx+len(marker):], nil
}

// tgARNSuffix extracts the "targetgroup/..." portion from a full target group ARN.
func tgARNSuffix(tgARN string) string {
	const marker = "targetgroup/"
	idx := strings.Index(tgARN, marker)
	if idx == -1 {
		return ""
	}
	return tgARN[idx:]
}

// buildMetricQueries constructs CloudWatch MetricDataQuery slices.
// Always includes 4 ECS metrics. Conditionally adds capacity provider, ASG, and ALB metrics.
func buildMetricQueries(cluster, service, asgName, tgARN, albSuffix, cpName string, period int32) []cwtypes.MetricDataQuery {
	returnData := true
	var queries []cwtypes.MetricDataQuery

	queries = append(queries, cwtypes.MetricDataQuery{
		Id:         aws.String("ecs_cpu"),
		ReturnData: &returnData,
		MetricStat: &cwtypes.MetricStat{
			Metric: &cwtypes.Metric{
				Namespace:  aws.String("AWS/ECS"),
				MetricName: aws.String("CPUUtilization"),
				Dimensions: []cwtypes.Dimension{
					{Name: aws.String("ClusterName"), Value: aws.String(cluster)},
					{Name: aws.String("ServiceName"), Value: aws.String(service)},
				},
			},
			Period: &period,
			Stat:   aws.String("Average"),
		},
	})

	queries = append(queries, cwtypes.MetricDataQuery{
		Id:         aws.String("ecs_memory"),
		ReturnData: &returnData,
		MetricStat: &cwtypes.MetricStat{
			Metric: &cwtypes.Metric{
				Namespace:  aws.String("AWS/ECS"),
				MetricName: aws.String("MemoryUtilization"),
				Dimensions: []cwtypes.Dimension{
					{Name: aws.String("ClusterName"), Value: aws.String(cluster)},
					{Name: aws.String("ServiceName"), Value: aws.String(service)},
				},
			},
			Period: &period,
			Stat:   aws.String("Average"),
		},
	})

	queries = append(queries, cwtypes.MetricDataQuery{
		Id:         aws.String("cluster_cpu_reservation"),
		ReturnData: &returnData,
		MetricStat: &cwtypes.MetricStat{
			Metric: &cwtypes.Metric{
				Namespace:  aws.String("AWS/ECS"),
				MetricName: aws.String("CPUReservation"),
				Dimensions: []cwtypes.Dimension{
					{Name: aws.String("ClusterName"), Value: aws.String(cluster)},
				},
			},
			Period: &period,
			Stat:   aws.String("Average"),
		},
	})

	queries = append(queries, cwtypes.MetricDataQuery{
		Id:         aws.String("cluster_memory_reservation"),
		ReturnData: &returnData,
		MetricStat: &cwtypes.MetricStat{
			Metric: &cwtypes.Metric{
				Namespace:  aws.String("AWS/ECS"),
				MetricName: aws.String("MemoryReservation"),
				Dimensions: []cwtypes.Dimension{
					{Name: aws.String("ClusterName"), Value: aws.String(cluster)},
				},
			},
			Period: &period,
			Stat:   aws.String("Average"),
		},
	})

	if cpName != "" {
		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("capacity_provider_reservation"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/ECS"),
					MetricName: aws.String("CapacityProviderReservation"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("ClusterName"), Value: aws.String(cluster)},
						{Name: aws.String("CapacityProviderName"), Value: aws.String(cpName)},
					},
				},
				Period: &period,
				Stat:   aws.String("Average"),
			},
		})
	}

	if asgName != "" {
		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("asg_desired"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/AutoScaling"),
					MetricName: aws.String("GroupDesiredCapacity"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("AutoScalingGroupName"), Value: aws.String(asgName)},
					},
				},
				Period: &period,
				Stat:   aws.String("Maximum"),
			},
		})

		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("asg_in_service"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/AutoScaling"),
					MetricName: aws.String("GroupInServiceInstances"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("AutoScalingGroupName"), Value: aws.String(asgName)},
					},
				},
				Period: &period,
				Stat:   aws.String("Maximum"),
			},
		})
	}

	if tgARN != "" && albSuffix != "" {
		tgSuffix := tgARNSuffix(tgARN)

		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("alb_requests_per_target"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/ApplicationELB"),
					MetricName: aws.String("RequestCountPerTarget"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("TargetGroup"), Value: aws.String(tgSuffix)},
					},
				},
				Period: &period,
				Stat:   aws.String("Sum"),
			},
		})

		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("alb_response_time"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/ApplicationELB"),
					MetricName: aws.String("TargetResponseTime"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("LoadBalancer"), Value: aws.String(albSuffix)},
						{Name: aws.String("TargetGroup"), Value: aws.String(tgSuffix)},
					},
				},
				Period: &period,
				Stat:   aws.String("p95"),
			},
		})

		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("alb_5xx"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/ApplicationELB"),
					MetricName: aws.String("HTTPCode_Target_5XX_Count"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("LoadBalancer"), Value: aws.String(albSuffix)},
						{Name: aws.String("TargetGroup"), Value: aws.String(tgSuffix)},
					},
				},
				Period: &period,
				Stat:   aws.String("Sum"),
			},
		})

		queries = append(queries, cwtypes.MetricDataQuery{
			Id:         aws.String("alb_healthy_hosts"),
			ReturnData: &returnData,
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/ApplicationELB"),
					MetricName: aws.String("HealthyHostCount"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("LoadBalancer"), Value: aws.String(albSuffix)},
						{Name: aws.String("TargetGroup"), Value: aws.String(tgSuffix)},
					},
				},
				Period: &period,
				Stat:   aws.String("Average"),
			},
		})
	}

	return queries
}
