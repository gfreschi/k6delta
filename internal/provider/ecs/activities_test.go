package ecs

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/applicationautoscaling"
	appastypes "github.com/aws/aws-sdk-go-v2/service/applicationautoscaling/types"
	autoscalingapi "github.com/aws/aws-sdk-go-v2/service/autoscaling"
	astypes "github.com/aws/aws-sdk-go-v2/service/autoscaling/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/gfreschi/k6delta/internal/config"
)

type mockECSScaling struct {
	output *applicationautoscaling.DescribeScalingActivitiesOutput
	err    error
}

func (m *mockECSScaling) DescribeScalingActivities(ctx context.Context, params *applicationautoscaling.DescribeScalingActivitiesInput, optFns ...func(*applicationautoscaling.Options)) (*applicationautoscaling.DescribeScalingActivitiesOutput, error) {
	return m.output, m.err
}

type mockASGScaling struct {
	output *autoscalingapi.DescribeScalingActivitiesOutput
	err    error
}

func (m *mockASGScaling) DescribeScalingActivities(ctx context.Context, params *autoscalingapi.DescribeScalingActivitiesInput, optFns ...func(*autoscalingapi.Options)) (*autoscalingapi.DescribeScalingActivitiesOutput, error) {
	return m.output, m.err
}

type mockAlarm struct {
	output *cloudwatch.DescribeAlarmHistoryOutput
	err    error
}

func (m *mockAlarm) DescribeAlarmHistory(ctx context.Context, params *cloudwatch.DescribeAlarmHistoryInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmHistoryOutput, error) {
	return m.output, m.err
}

func TestFetchActivities(t *testing.T) {
	start := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)
	inWindow := time.Date(2026, 3, 3, 11, 0, 0, 0, time.UTC)

	cause := "scaling policy triggered"
	desc := "scaled from 1 to 2"

	ecsScalingMock := &mockECSScaling{
		output: &applicationautoscaling.DescribeScalingActivitiesOutput{
			ScalingActivities: []appastypes.ScalingActivity{
				{StartTime: &inWindow, StatusCode: appastypes.ScalingActivityStatusCodeSuccessful, Cause: &cause, Description: &desc},
			},
		},
	}
	asgScalingMock := &mockASGScaling{
		output: &autoscalingapi.DescribeScalingActivitiesOutput{
			Activities: []astypes.Activity{
				{StartTime: &inWindow, StatusCode: astypes.ScalingActivityStatusCodeSuccessful, Cause: &cause, ActivityId: aws.String("a1"), AutoScalingGroupName: aws.String("my-asg")},
			},
		},
	}
	alarmMock := &mockAlarm{
		output: &cloudwatch.DescribeAlarmHistoryOutput{
			AlarmHistoryItems: []cwtypes.AlarmHistoryItem{
				{
					AlarmName:   aws.String("myapp-staging-cpu"),
					Timestamp:   &inWindow,
					HistoryData: aws.String(`{"oldState":{"stateValue":"OK"},"newState":{"stateValue":"ALARM"}}`),
				},
			},
		},
	}
	asgMock := &mockASG{
		output: &autoscalingapi.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []astypes.AutoScalingGroup{
				{AutoScalingGroupName: aws.String("myapp-staging-ecs-abc")},
			},
		},
	}

	p := &Provider{
		app: config.ResolvedApp{
			Cluster:     "myapp-staging",
			Service:     "myapp-web-staging",
			ASGPrefix:   "myapp-staging-ecs-",
			AlarmPrefix: "myapp-staging",
		},
	}

	activities, err := p.fetchActivitiesWithClients(context.Background(), ecsScalingMock, asgScalingMock, alarmMock, asgMock, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities.ECSScaling) != 1 {
		t.Errorf("ECSScaling = %d, want 1", len(activities.ECSScaling))
	}
	if len(activities.ASGScaling) != 1 {
		t.Errorf("ASGScaling = %d, want 1", len(activities.ASGScaling))
	}
	if len(activities.Alarms) != 1 {
		t.Errorf("Alarms = %d, want 1", len(activities.Alarms))
	}
	if activities.Alarms[0].OldState != "OK" {
		t.Errorf("Alarms[0].OldState = %q, want OK", activities.Alarms[0].OldState)
	}
	if activities.Alarms[0].NewState != "ALARM" {
		t.Errorf("Alarms[0].NewState = %q, want ALARM", activities.Alarms[0].NewState)
	}
}

func TestFetchActivitiesProgress(t *testing.T) {
	start := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	ecsScalingMock := &mockECSScaling{
		output: &applicationautoscaling.DescribeScalingActivitiesOutput{},
	}
	asgScalingMock := &mockASGScaling{
		output: &autoscalingapi.DescribeScalingActivitiesOutput{},
	}
	alarmMock := &mockAlarm{
		output: &cloudwatch.DescribeAlarmHistoryOutput{},
	}
	asgMock := &mockASG{
		output: &autoscalingapi.DescribeAutoScalingGroupsOutput{
			AutoScalingGroups: []astypes.AutoScalingGroup{
				{AutoScalingGroupName: aws.String("myapp-staging-ecs-abc")},
			},
		},
	}

	var progress []string
	p := &Provider{
		app: config.ResolvedApp{
			Cluster:     "c",
			Service:     "s",
			ASGPrefix:   "myapp-staging-ecs-",
			AlarmPrefix: "myapp-",
		},
		onProgress: func(id string, current, total int) {
			progress = append(progress, id)
		},
	}

	_, err := p.fetchActivitiesWithClients(context.Background(), ecsScalingMock, asgScalingMock, alarmMock, asgMock, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progress) != 3 {
		t.Fatalf("progress calls = %d, want 3; got %v", len(progress), progress)
	}
	if progress[0] != "ECS scaling" {
		t.Errorf("progress[0] = %q, want %q", progress[0], "ECS scaling")
	}
	if progress[1] != "ASG scaling" {
		t.Errorf("progress[1] = %q, want %q", progress[1], "ASG scaling")
	}
	if progress[2] != "alarm history" {
		t.Errorf("progress[2] = %q, want %q", progress[2], "alarm history")
	}
}

func TestFetchActivitiesProgressMinimal(t *testing.T) {
	start := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	ecsScalingMock := &mockECSScaling{
		output: &applicationautoscaling.DescribeScalingActivitiesOutput{},
	}
	asgScalingMock := &mockASGScaling{
		output: &autoscalingapi.DescribeScalingActivitiesOutput{},
	}
	alarmMock := &mockAlarm{
		output: &cloudwatch.DescribeAlarmHistoryOutput{},
	}
	asgMock := &mockASG{
		output: &autoscalingapi.DescribeAutoScalingGroupsOutput{},
	}

	var progress []string
	p := &Provider{
		app: config.ResolvedApp{Cluster: "c", Service: "s"},
		onProgress: func(id string, current, total int) {
			progress = append(progress, id)
		},
	}

	_, err := p.fetchActivitiesWithClients(context.Background(), ecsScalingMock, asgScalingMock, alarmMock, asgMock, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(progress) != 1 {
		t.Fatalf("progress calls = %d, want 1; got %v", len(progress), progress)
	}
	if progress[0] != "ECS scaling" {
		t.Errorf("progress[0] = %q, want %q", progress[0], "ECS scaling")
	}
}

func TestFetchActivitiesNoASG(t *testing.T) {
	start := time.Date(2026, 3, 3, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 3, 12, 0, 0, 0, time.UTC)

	ecsScalingMock := &mockECSScaling{
		output: &applicationautoscaling.DescribeScalingActivitiesOutput{},
	}
	asgScalingMock := &mockASGScaling{
		output: &autoscalingapi.DescribeScalingActivitiesOutput{},
	}
	alarmMock := &mockAlarm{
		output: &cloudwatch.DescribeAlarmHistoryOutput{},
	}
	asgMock := &mockASG{
		output: &autoscalingapi.DescribeAutoScalingGroupsOutput{},
	}

	p := &Provider{
		app: config.ResolvedApp{Cluster: "c", Service: "s"},
	}

	activities, err := p.fetchActivitiesWithClients(context.Background(), ecsScalingMock, asgScalingMock, alarmMock, asgMock, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities.ECSScaling) != 0 {
		t.Errorf("ECSScaling = %d, want 0", len(activities.ECSScaling))
	}
	if len(activities.ASGScaling) != 0 {
		t.Errorf("ASGScaling = %d, want 0", len(activities.ASGScaling))
	}
	if len(activities.Alarms) != 0 {
		t.Errorf("Alarms = %d, want 0", len(activities.Alarms))
	}
}
