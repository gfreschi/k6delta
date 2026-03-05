package mock

import (
	"fmt"
	"time"

	"github.com/gfreschi/k6delta/internal/provider"
)

// MetricCurve pairs a metric ID with a time-series generator.
type MetricCurve struct {
	ID        string
	Generator Generator
}

// ActivityTemplate defines a scaling event at a relative time offset.
type ActivityTemplate struct {
	AtNormalized float64 // when in [0,1] normalized time
	Status       string
	Description  string
}

// AlarmTemplate defines an alarm event at a relative time offset.
type AlarmTemplate struct {
	AtNormalized float64
	AlarmName    string
	OldState     string
	NewState     string
}

// Scenario defines the "story" of a simulated load test.
type Scenario struct {
	Name         string
	Description  string
	Duration     time.Duration
	PreSnapshot  provider.Snapshot
	PostSnapshot provider.Snapshot
	Metrics      []MetricCurve
	Activities   []ActivityTemplate
	Alarms       []AlarmTemplate
}

// ScenarioInfo is the public summary returned by ListScenarios.
type ScenarioInfo struct {
	Name        string
	Description string
}

var scenarios = map[string]Scenario{
	"happy-path": {
		Name:        "happy-path",
		Description: "Stable system: CPU ~40%, memory flat, no scaling events. PASS verdict.",
		Duration:    60 * time.Second,
		PreSnapshot: provider.Snapshot{
			TaskRunning: 4, TaskDesired: 4,
			ASGInstances: 2, ASGDesired: 2, ASGName: "mock-asg",
		},
		PostSnapshot: provider.Snapshot{
			TaskRunning: 4, TaskDesired: 4,
			ASGInstances: 2, ASGDesired: 2, ASGName: "mock-asg",
		},
		Metrics: []MetricCurve{
			{ID: "service_cpu", Generator: Noise(Constant(38.0), 4.0)},
			{ID: "service_memory", Generator: Noise(Constant(512.0), 20.0)},
			{ID: "cluster_cpu_reservation", Generator: Noise(Constant(45.0), 3.0)},
			{ID: "alb_response_time", Generator: Noise(Constant(120.0), 15.0)},
		},
	},
	"cpu-spike": {
		Name:        "cpu-spike",
		Description: "CPU ramps to 92% at midpoint, drops back. WARN verdict.",
		Duration:    60 * time.Second,
		PreSnapshot: provider.Snapshot{
			TaskRunning: 4, TaskDesired: 4,
			ASGInstances: 2, ASGDesired: 2, ASGName: "mock-asg",
		},
		PostSnapshot: provider.Snapshot{
			TaskRunning: 4, TaskDesired: 4,
			ASGInstances: 2, ASGDesired: 2, ASGName: "mock-asg",
		},
		Metrics: []MetricCurve{
			{ID: "service_cpu", Generator: Noise(Sine(55.0, 37.0), 3.0)},
			{ID: "service_memory", Generator: Noise(Ramp(480.0, 620.0), 15.0)},
			{ID: "cluster_cpu_reservation", Generator: Noise(Sine(50.0, 25.0), 2.0)},
			{ID: "alb_response_time", Generator: Noise(Sine(200.0, 150.0), 20.0)},
		},
	},
	"scale-out": {
		Name:        "scale-out",
		Description: "2->5 containers, ASG fires, CPU recovers after scaling. PASS verdict with scaling events.",
		Duration:    90 * time.Second,
		PreSnapshot: provider.Snapshot{
			TaskRunning: 2, TaskDesired: 2,
			ASGInstances: 1, ASGDesired: 1, ASGName: "mock-asg",
		},
		PostSnapshot: provider.Snapshot{
			TaskRunning: 5, TaskDesired: 5,
			ASGInstances: 3, ASGDesired: 3, ASGName: "mock-asg",
		},
		Metrics: []MetricCurve{
			{ID: "service_cpu", Generator: func() Generator {
				// Spikes to 85% then drops after scaling at t=0.5
				spike := Sine(60.0, 25.0)
				recovery := Ramp(85.0, 35.0)
				return func(t float64) float64 {
					if t < 0.5 {
						return spike(t)
					}
					return recovery((t - 0.5) * 2)
				}
			}()},
			{ID: "service_memory", Generator: Noise(Ramp(400.0, 700.0), 25.0)},
			{ID: "cluster_cpu_reservation", Generator: Step(40.0, 75.0, 0.4)},
			{ID: "alb_response_time", Generator: Noise(Sine(180.0, 80.0), 25.0)},
		},
		Activities: []ActivityTemplate{
			{AtNormalized: 0.35, Status: "Successful", Description: "ECS service scaled from 2 to 4 tasks"},
			{AtNormalized: 0.45, Status: "Successful", Description: "ASG launched instance i-mock-003"},
			{AtNormalized: 0.55, Status: "Successful", Description: "ECS service scaled from 4 to 5 tasks"},
		},
	},
	"cascade-failure": {
		Name:        "cascade-failure",
		Description: "CPU 99%, 5xx errors spike, containers restart. FAIL verdict.",
		Duration:    60 * time.Second,
		PreSnapshot: provider.Snapshot{
			TaskRunning: 4, TaskDesired: 4,
			ASGInstances: 2, ASGDesired: 2, ASGName: "mock-asg",
		},
		PostSnapshot: provider.Snapshot{
			TaskRunning: 3, TaskDesired: 4,
			ASGInstances: 2, ASGDesired: 2, ASGName: "mock-asg",
		},
		Metrics: []MetricCurve{
			{ID: "service_cpu", Generator: Ramp(50.0, 99.0)},
			{ID: "service_memory", Generator: Ramp(512.0, 1800.0)},
			{ID: "cluster_cpu_reservation", Generator: Ramp(55.0, 98.0)},
			{ID: "alb_response_time", Generator: Ramp(150.0, 4500.0)},
			{ID: "alb_5xx", Generator: Step(0.0, 25.0, 0.6)},
		},
		Activities: []ActivityTemplate{
			{AtNormalized: 0.6, Status: "Failed", Description: "Container web-1 OOMKilled"},
			{AtNormalized: 0.7, Status: "InProgress", Description: "ECS restarting task web-1"},
			{AtNormalized: 0.8, Status: "Failed", Description: "Container web-3 OOMKilled"},
		},
		Alarms: []AlarmTemplate{
			{AtNormalized: 0.5, AlarmName: "high-cpu-alarm", OldState: "OK", NewState: "ALARM"},
			{AtNormalized: 0.65, AlarmName: "5xx-error-alarm", OldState: "OK", NewState: "ALARM"},
		},
	},
}

// GetScenario returns a scenario by name.
func GetScenario(name string) (Scenario, error) {
	s, ok := scenarios[name]
	if !ok {
		return Scenario{}, fmt.Errorf("unknown scenario %q (available: %s)", name, scenarioNames())
	}
	return s, nil
}

// ListScenarios returns all available scenario summaries.
func ListScenarios() []ScenarioInfo {
	order := []string{"happy-path", "cpu-spike", "scale-out", "cascade-failure"}
	infos := make([]ScenarioInfo, 0, len(order))
	for _, name := range order {
		s := scenarios[name]
		infos = append(infos, ScenarioInfo{Name: s.Name, Description: s.Description})
	}
	return infos
}

func scenarioNames() string {
	names := ""
	for _, info := range ListScenarios() {
		if names != "" {
			names += ", "
		}
		names += info.Name
	}
	return names
}
