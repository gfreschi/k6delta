package ecs

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	appastypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	autoscalingapi "github.com/aws/aws-sdk-go-v2/service/autoscaling"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gfreschi/k6delta/internal/provider"
)

// ECSScalingDescriber abstracts the Application Auto Scaling API for testing.
type ECSScalingDescriber interface {
	DescribeScalingActivities(ctx context.Context, params *applicationautoscaling.DescribeScalingActivitiesInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalingActivitiesOutput, error)
}

// ASGScalingDescriber abstracts the ASG DescribeScalingActivities API for testing.
type ASGScalingDescriber interface {
	DescribeScalingActivities(ctx context.Context, params *autoscalingapi.DescribeScalingActivitiesInput, optFns ...func(*autoscalingapi.Options)) (*autoscalingapi.DescribeScalingActivitiesOutput, error)
}

// AlarmDescriber abstracts the CloudWatch DescribeAlarmHistory API for testing.
type AlarmDescriber interface {
	DescribeAlarmHistory(ctx context.Context, params *cloudwatch.DescribeAlarmHistoryInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmHistoryOutput, error)
}

// FetchActivities retrieves scaling events and alarm history for the test window.
func (p *Provider) FetchActivities(ctx context.Context, start, end time.Time) (provider.Activities, error) {
	ecsScaling := applicationautoscaling.NewFromConfig(p.cfg)
	asgClient := autoscalingapi.NewFromConfig(p.cfg)
	cwClient := cloudwatch.NewFromConfig(p.cfg)
	return p.fetchActivitiesWithClients(ctx, ecsScaling, asgClient, cwClient, asgClient, start, end)
}

func (p *Provider) fetchActivitiesWithClients(ctx context.Context, ecsScaling ECSScalingDescriber, asgScaling ASGScalingDescriber, cwClient AlarmDescriber, asgClient ASGDescriber, start, end time.Time) (provider.Activities, error) {
	var result provider.Activities

	// Resolve ASG name early to compute totalSteps
	var asgName string
	if p.asgResolved {
		asgName = p.asgName
	} else if p.app.ASGPrefix != "" {
		asgName, _ = discoverASGName(ctx, asgClient, p.app.ASGPrefix)
	}

	// Compute totalSteps dynamically based on optional sections
	totalSteps := 1 // always ECS scaling
	if asgName != "" {
		totalSteps++
	}
	if p.app.AlarmPrefix != "" {
		totalSteps++
	}
	step := 1

	// ECS Application Auto Scaling activities
	p.reportProgress("ECS scaling", step, totalSteps)
	resourceID := fmt.Sprintf("service/%s/%s", p.app.Cluster, p.app.Service)
	ecsOut, err := ecsScaling.DescribeScalingActivities(ctx, &applicationautoscaling.DescribeScalingActivitiesInput{
		ServiceNamespace: appastypes.ServiceNamespaceEcs,
		ResourceId:       &resourceID,
	})
	if err != nil {
		return result, fmt.Errorf("describe ECS scaling activities: %w", err)
	}
	for _, a := range ecsOut.ScalingActivities {
		if a.StartTime == nil || a.StartTime.Before(start) || a.StartTime.After(end) {
			continue
		}
		sa := provider.ScalingActivity{
			Time:   a.StartTime.Format(time.RFC3339),
			Status: string(a.StatusCode),
		}
		if a.Cause != nil {
			sa.Cause = *a.Cause
		}
		if a.Description != nil {
			sa.Description = *a.Description
		}
		result.ECSScaling = append(result.ECSScaling, sa)
	}

	// ASG scaling activities (optional)
	if asgName != "" {
		step++
		p.reportProgress("ASG scaling", step, totalSteps)
		asgOut, err := asgScaling.DescribeScalingActivities(ctx, &autoscalingapi.DescribeScalingActivitiesInput{
			AutoScalingGroupName: &asgName,
		})
		if err != nil {
			return result, fmt.Errorf("describe ASG scaling activities: %w", err)
		}
		for _, a := range asgOut.Activities {
			if a.StartTime == nil || a.StartTime.Before(start) || a.StartTime.After(end) {
				continue
			}
			sa := provider.ScalingActivity{
				Time:   a.StartTime.Format(time.RFC3339),
				Status: string(a.StatusCode),
			}
			if a.Cause != nil {
				sa.Cause = *a.Cause
			}
			if a.Description != nil {
				sa.Description = *a.Description
			}
			result.ASGScaling = append(result.ASGScaling, sa)
		}
	}

	// CloudWatch alarm history (optional)
	if p.app.AlarmPrefix != "" {
		step++
		p.reportProgress("alarm history", step, totalSteps)
		alarmOut, err := cwClient.DescribeAlarmHistory(ctx, &cloudwatch.DescribeAlarmHistoryInput{
			StartDate:       &start,
			EndDate:         &end,
			HistoryItemType: cwtypes.HistoryItemTypeStateUpdate,
		})
		if err != nil {
			return result, fmt.Errorf("describe alarm history: %w", err)
		}
		for _, item := range alarmOut.AlarmHistoryItems {
			alarmName := aws.ToString(item.AlarmName)
			if !strings.HasPrefix(alarmName, p.app.AlarmPrefix) {
				continue
			}
			event := provider.AlarmEvent{AlarmName: alarmName}
			if item.Timestamp != nil {
				event.Time = item.Timestamp.Format(time.RFC3339)
			}
			if item.HistoryData != nil {
				var hd alarmHistoryData
				if err := json.Unmarshal([]byte(*item.HistoryData), &hd); err == nil {
					event.OldState = hd.OldState.StateValue
					event.NewState = hd.NewState.StateValue
				}
				if event.OldState == "" {
					event.OldState = "UNKNOWN"
				}
				if event.NewState == "" {
					event.NewState = "UNKNOWN"
				}
			} else {
				event.OldState = "UNKNOWN"
				event.NewState = "UNKNOWN"
			}
			result.Alarms = append(result.Alarms, event)
		}
	}

	return result, nil
}

type alarmHistoryData struct {
	OldState struct {
		StateValue string `json:"stateValue"`
	} `json:"oldState"`
	NewState struct {
		StateValue string `json:"stateValue"`
	} `json:"newState"`
}
