// Package provider defines the InfraProvider interface and shared types
// for infrastructure monitoring across different platforms.
package provider

import (
	"context"
	"time"
)

// Snapshot captures infrastructure state at a point in time.
type Snapshot struct {
	TaskRunning  int
	TaskDesired  int
	ASGName      string
	ASGInstances int
	ASGDesired   int
}

// MetricResult holds a single CloudWatch metric time series.
type MetricResult struct {
	ID         string
	Values     []float64
	Timestamps []time.Time
	Peak       *float64
	Avg        *float64
}

// ScalingActivity represents a single scaling event.
type ScalingActivity struct {
	Time        string
	Status      string
	Cause       string
	Description string
}

// AlarmEvent represents a CloudWatch alarm state change.
type AlarmEvent struct {
	AlarmName string
	Time      string
	OldState  string
	NewState  string
}

// Activities groups scaling events and alarm history.
type Activities struct {
	ServiceScaling []ScalingActivity
	NodeScaling    []ScalingActivity
	Alarms         []AlarmEvent
}

// InfraProvider abstracts infrastructure monitoring for different platforms.
// In v1, only the ECS provider exists. This interface is internal and not
// exported — it will be promoted to pkg/provider/ when a second provider
// (EKS, Prometheus) is added in v1.1+.
type InfraProvider interface {
	CheckCredentials(ctx context.Context) error
	TakeSnapshot(ctx context.Context) (Snapshot, error)
	FetchMetrics(ctx context.Context, start, end time.Time, period int32) ([]MetricResult, error)
	FetchActivities(ctx context.Context, start, end time.Time) (Activities, error)
}
