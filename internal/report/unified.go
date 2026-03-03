package report

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

type k6Summary struct {
	Metrics map[string]k6Metric `json:"metrics"`
}

type k6Metric struct {
	Values     map[string]json.RawMessage `json:"values"`
	Thresholds map[string]k6Threshold     `json:"thresholds"`
}

type k6Threshold struct {
	OK bool `json:"ok"`
}

// ParseK6Summary reads a k6 JSON summary file and extracts key metrics.
func ParseK6Summary(path string) (*K6Metrics, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read k6 summary %q: %w", path, err)
	}

	var summary k6Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, fmt.Errorf("parse k6 summary %q: %w", path, err)
	}

	m := &K6Metrics{}

	if dur, ok := summary.Metrics["http_req_duration"]; ok {
		m.P95ms = extractFloat(dur.Values, "p(95)")
		if m.P95ms != nil {
			v := roundTo(*m.P95ms, 2)
			m.P95ms = &v
		}
		m.P90ms = extractFloat(dur.Values, "p(90)")
		if m.P90ms != nil {
			v := roundTo(*m.P90ms, 2)
			m.P90ms = &v
		}
	}

	if failed, ok := summary.Metrics["http_req_failed"]; ok {
		m.ErrorRate = extractFloat(failed.Values, "rate")
		if m.ErrorRate != nil {
			v := roundTo(*m.ErrorRate, 4)
			m.ErrorRate = &v
		}
	}

	if checks, ok := summary.Metrics["checks"]; ok {
		m.ChecksRate = extractFloat(checks.Values, "rate")
	}

	if reqs, ok := summary.Metrics["http_reqs"]; ok {
		m.TotalRequests = extractInt(reqs.Values, "count")
		m.RPSAvg = extractFloat(reqs.Values, "rate")
	}

	if vus, ok := summary.Metrics["vus_max"]; ok {
		m.VUsMax = extractInt(vus.Values, "max")
	}

	passed, failed := countThresholds(summary.Metrics)
	m.Thresholds = ThresholdSummary{Passed: passed, Failed: failed}

	return m, nil
}

// BuildUnifiedReport assembles a full UnifiedReport from the given inputs.
func BuildUnifiedReport(info RunInfo, k6SummaryPath string, analysisJSON []byte, preTasks, postTasks, preASG, postASG int) (*UnifiedReport, error) {
	report := &UnifiedReport{
		Run: info,
	}

	if k6SummaryPath != "" {
		k6, err := ParseK6Summary(k6SummaryPath)
		if err != nil {
			return nil, fmt.Errorf("parse k6 summary: %w", err)
		}
		report.K6 = k6
	}

	if len(analysisJSON) > 0 {
		var infra InfraMetrics
		if err := json.Unmarshal(analysisJSON, &infra); err != nil {
			return nil, fmt.Errorf("parse analysis JSON: %w", err)
		}
		report.Infrastructure = &infra
	}

	if report.Infrastructure == nil {
		report.Infrastructure = &InfraMetrics{}
	}
	report.Infrastructure.Tasks = BeforeAfter{Before: preTasks, After: postTasks}
	report.Infrastructure.ASG = BeforeAfter{Before: preASG, After: postASG}

	return report, nil
}

// BuildUnifiedReportFromInfra assembles a UnifiedReport using an InfraMetrics struct directly,
// avoiding a JSON round-trip.
func BuildUnifiedReportFromInfra(info RunInfo, k6SummaryPath string, infra *InfraMetrics, preTasks, postTasks, preASG, postASG int) (*UnifiedReport, error) {
	report := &UnifiedReport{
		Run: info,
	}

	if k6SummaryPath != "" {
		k6, err := ParseK6Summary(k6SummaryPath)
		if err != nil {
			return nil, fmt.Errorf("parse k6 summary: %w", err)
		}
		report.K6 = k6
	}

	if infra != nil {
		report.Infrastructure = infra
	} else {
		report.Infrastructure = &InfraMetrics{}
	}
	report.Infrastructure.Tasks = BeforeAfter{Before: preTasks, After: postTasks}
	report.Infrastructure.ASG = BeforeAfter{Before: preASG, After: postASG}

	return report, nil
}

// WriteReport serializes the report as indented JSON and writes it to the given path.
func WriteReport(report *UnifiedReport, path string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write report to %q: %w", path, err)
	}
	return nil
}

func extractFloat(values map[string]json.RawMessage, key string) *float64 {
	raw, ok := values[key]
	if !ok {
		return nil
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	return &v
}

func extractInt(values map[string]json.RawMessage, key string) *int {
	raw, ok := values[key]
	if !ok {
		return nil
	}
	var v float64
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	i := int(v)
	return &i
}

func roundTo(v float64, decimals int) float64 {
	p := math.Pow(10, float64(decimals))
	return math.Round(v*p) / p
}

func countThresholds(metrics map[string]k6Metric) (passed, failed int) {
	for _, m := range metrics {
		for _, t := range m.Thresholds {
			if t.OK {
				passed++
			} else {
				failed++
			}
		}
	}
	return passed, failed
}
