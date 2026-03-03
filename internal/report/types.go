// Package report defines the unified report schema and provides
// comparison logic for k6 and infrastructure metrics.
package report

// UnifiedReport is the top-level structure for a load test report.
type UnifiedReport struct {
	Run               RunInfo       `json:"run"`
	K6                *K6Metrics    `json:"k6"`
	Infrastructure    *InfraMetrics `json:"infrastructure"`
	ScalingActivities interface{}   `json:"scaling_activities"`
	AlarmHistory      interface{}   `json:"alarm_history"`
}

// RunInfo captures metadata about a single load test execution.
type RunInfo struct {
	App             string `json:"app"`
	Env             string `json:"env"`
	Phase           string `json:"phase"`
	Start           string `json:"start"`
	End             string `json:"end"`
	K6Exit          int    `json:"k6_exit"`
	DurationSeconds int    `json:"duration_seconds"`
}

// K6Metrics holds the key metrics extracted from a k6 JSON summary file.
type K6Metrics struct {
	P95ms         *float64         `json:"p95_ms"`
	P90ms         *float64         `json:"p90_ms"`
	ErrorRate     *float64         `json:"error_rate"`
	ChecksRate    *float64         `json:"checks_rate"`
	TotalRequests *int             `json:"total_requests"`
	RPSAvg        *float64         `json:"rps_avg"`
	VUsMax        *int             `json:"vus_max"`
	Thresholds    ThresholdSummary `json:"thresholds"`
}

// ThresholdSummary counts how many k6 thresholds passed and failed.
type ThresholdSummary struct {
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// InfraMetrics holds infrastructure metrics collected during the test.
type InfraMetrics struct {
	ECSCPU                      *PeakAvg    `json:"ecs_cpu,omitempty"`
	ECSMemory                   *PeakAvg    `json:"ecs_memory,omitempty"`
	ClusterCPUReservation       *PeakAvg    `json:"cluster_cpu_reservation,omitempty"`
	ClusterMemoryReservation    *PeakAvg    `json:"cluster_memory_reservation,omitempty"`
	CapacityProviderReservation *PeakAvg    `json:"capacity_provider_reservation,omitempty"`
	Tasks                       BeforeAfter `json:"tasks"`
	ASG                         BeforeAfter `json:"asg"`
	ALB5xx                      int         `json:"alb_5xx"`
	ALBResponseTimeP95          *float64    `json:"alb_response_time_p95,omitempty"`
	ALBHealthyHosts             *MinMax     `json:"alb_healthy_hosts,omitempty"`
}

// PeakAvg stores the peak and average values for a metric time series.
type PeakAvg struct {
	Peak *float64 `json:"peak"`
	Avg  *float64 `json:"avg"`
}

// BeforeAfter stores a metric's value before and after the test.
type BeforeAfter struct {
	Before int `json:"before"`
	After  int `json:"after"`
}

// MinMax stores the minimum and maximum values for a metric time series.
type MinMax struct {
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
}
