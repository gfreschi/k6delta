package report

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// ComparisonRow holds a single metric comparison between two runs.
type ComparisonRow struct {
	Metric    string `json:"metric"`
	ValueA    string `json:"value_a"`
	ValueB    string `json:"value_b"`
	Delta     string `json:"delta"`
	Direction string `json:"direction"` // "better", "worse", "same", ""
}

// ComparisonResult holds the full comparison between two report runs.
type ComparisonResult struct {
	RunA      RunInfo         `json:"run_a"`
	RunB      RunInfo         `json:"run_b"`
	K6Rows    []ComparisonRow `json:"k6"`
	InfraRows []ComparisonRow `json:"infrastructure"`
}

// ComputeDelta computes the percentage change from a to b.
func ComputeDelta(a, b float64) string {
	if a == 0 && b == 0 {
		return "same"
	}
	if a == 0 {
		return "new"
	}
	pct := (b - a) / a * 100
	if pct == 0 || math.Abs(pct) < 0.05 {
		return "same"
	}
	if pct > 0 {
		return fmt.Sprintf("+%.1f%%", pct)
	}
	return fmt.Sprintf("%.1f%%", pct)
}

// Direction determines whether a delta represents an improvement or regression.
func Direction(delta string, metricType string) string {
	if delta == "same" || delta == "N/A" || delta == "new" || len(delta) == 0 {
		return ""
	}

	sign := delta[0]
	switch metricType {
	case "lower_better":
		if sign == '-' {
			return "better"
		}
		return "worse"
	case "higher_better":
		if sign == '+' {
			return "better"
		}
		return "worse"
	default:
		return ""
	}
}

func loadReport(path string) (*UnifiedReport, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var r UnifiedReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &r, nil
}

func floatStr(v *float64) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%.4g", *v)
}

func intStr(v *int) string {
	if v == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *v)
}

func floatVal(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}

func intFloatVal(v *int) float64 {
	if v == nil {
		return 0
	}
	return float64(*v)
}

// CompareReports loads two unified report JSONs and computes a comparison.
func CompareReports(pathA, pathB string) (*ComparisonResult, error) {
	a, err := loadReport(pathA)
	if err != nil {
		return nil, err
	}
	b, err := loadReport(pathB)
	if err != nil {
		return nil, err
	}

	result := &ComparisonResult{RunA: a.Run, RunB: b.Run}

	type compRow struct {
		metric, valA, valB string
		floatA, floatB     float64
		metricType         string
	}

	var aK6, bK6 K6Metrics
	if a.K6 != nil {
		aK6 = *a.K6
	}
	if b.K6 != nil {
		bK6 = *b.K6
	}

	k6Defs := []compRow{
		{"p95", floatStr(aK6.P95ms), floatStr(bK6.P95ms), floatVal(aK6.P95ms), floatVal(bK6.P95ms), "lower_better"},
		{"p90", floatStr(aK6.P90ms), floatStr(bK6.P90ms), floatVal(aK6.P90ms), floatVal(bK6.P90ms), "lower_better"},
		{"error_rate", floatStr(aK6.ErrorRate), floatStr(bK6.ErrorRate), floatVal(aK6.ErrorRate), floatVal(bK6.ErrorRate), "lower_better"},
		{"throughput", floatStr(aK6.RPSAvg), floatStr(bK6.RPSAvg), floatVal(aK6.RPSAvg), floatVal(bK6.RPSAvg), "higher_better"},
		{"checks_rate", floatStr(aK6.ChecksRate), floatStr(bK6.ChecksRate), floatVal(aK6.ChecksRate), floatVal(bK6.ChecksRate), "higher_better"},
		{"total_requests", intStr(aK6.TotalRequests), intStr(bK6.TotalRequests), intFloatVal(aK6.TotalRequests), intFloatVal(bK6.TotalRequests), "higher_better"},
	}

	for _, d := range k6Defs {
		delta := ComputeDelta(d.floatA, d.floatB)
		dir := Direction(delta, d.metricType)
		result.K6Rows = append(result.K6Rows, ComparisonRow{
			Metric: d.metric, ValueA: d.valA, ValueB: d.valB, Delta: delta, Direction: dir,
		})
	}

	var aInfra, bInfra InfraMetrics
	if a.Infrastructure != nil {
		aInfra = *a.Infrastructure
	}
	if b.Infrastructure != nil {
		bInfra = *b.Infrastructure
	}

	var aCPUPeak, bCPUPeak float64
	aCPUStr, bCPUStr := "-", "-"
	if aInfra.ECSCPU != nil {
		aCPUPeak = floatVal(aInfra.ECSCPU.Peak)
		aCPUStr = floatStr(aInfra.ECSCPU.Peak)
	}
	if bInfra.ECSCPU != nil {
		bCPUPeak = floatVal(bInfra.ECSCPU.Peak)
		bCPUStr = floatStr(bInfra.ECSCPU.Peak)
	}

	var aMemPeak, bMemPeak float64
	aMemStr, bMemStr := "-", "-"
	if aInfra.ECSMemory != nil {
		aMemPeak = floatVal(aInfra.ECSMemory.Peak)
		aMemStr = floatStr(aInfra.ECSMemory.Peak)
	}
	if bInfra.ECSMemory != nil {
		bMemPeak = floatVal(bInfra.ECSMemory.Peak)
		bMemStr = floatStr(bInfra.ECSMemory.Peak)
	}

	infraDefs := []compRow{
		{"ecs_cpu_peak", aCPUStr, bCPUStr, aCPUPeak, bCPUPeak, "lower_better"},
		{"ecs_memory_peak", aMemStr, bMemStr, aMemPeak, bMemPeak, "lower_better"},
		{"tasks_before", fmt.Sprintf("%d", aInfra.Tasks.Before), fmt.Sprintf("%d", bInfra.Tasks.Before), float64(aInfra.Tasks.Before), float64(bInfra.Tasks.Before), ""},
		{"tasks_after", fmt.Sprintf("%d", aInfra.Tasks.After), fmt.Sprintf("%d", bInfra.Tasks.After), float64(aInfra.Tasks.After), float64(bInfra.Tasks.After), ""},
		{"asg_before", fmt.Sprintf("%d", aInfra.ASG.Before), fmt.Sprintf("%d", bInfra.ASG.Before), float64(aInfra.ASG.Before), float64(bInfra.ASG.Before), ""},
		{"asg_after", fmt.Sprintf("%d", aInfra.ASG.After), fmt.Sprintf("%d", bInfra.ASG.After), float64(aInfra.ASG.After), float64(bInfra.ASG.After), ""},
		{"alb_5xx", fmt.Sprintf("%d", aInfra.ALB5xx), fmt.Sprintf("%d", bInfra.ALB5xx), float64(aInfra.ALB5xx), float64(bInfra.ALB5xx), "lower_better"},
	}

	for _, d := range infraDefs {
		delta := ComputeDelta(d.floatA, d.floatB)
		dir := Direction(delta, d.metricType)
		result.InfraRows = append(result.InfraRows, ComparisonRow{
			Metric: d.metric, ValueA: d.valA, ValueB: d.valB, Delta: delta, Direction: dir,
		})
	}

	return result, nil
}

type comparisonJSON struct {
	RunA       compRunJSON      `json:"run_a"`
	RunB       compRunJSON      `json:"run_b"`
	Comparison compSectionsJSON `json:"comparison"`
}

type compRunJSON struct {
	App   string `json:"app"`
	Phase string `json:"phase"`
	Start string `json:"start"`
}

type compSectionsJSON struct {
	K6    map[string]compPairJSON `json:"k6"`
	Infra map[string]compPairJSON `json:"infrastructure"`
}

type compPairJSON struct {
	A interface{} `json:"a"`
	B interface{} `json:"b"`
}

// CompareReportsJSON loads two reports and returns JSON output.
func CompareReportsJSON(pathA, pathB string) ([]byte, error) {
	a, err := loadReport(pathA)
	if err != nil {
		return nil, err
	}
	b, err := loadReport(pathB)
	if err != nil {
		return nil, err
	}

	var aK6, bK6 K6Metrics
	if a.K6 != nil {
		aK6 = *a.K6
	}
	if b.K6 != nil {
		bK6 = *b.K6
	}

	var aInfra, bInfra InfraMetrics
	if a.Infrastructure != nil {
		aInfra = *a.Infrastructure
	}
	if b.Infrastructure != nil {
		bInfra = *b.Infrastructure
	}

	out := comparisonJSON{
		RunA: compRunJSON{App: a.Run.App, Phase: a.Run.Phase, Start: a.Run.Start},
		RunB: compRunJSON{App: b.Run.App, Phase: b.Run.Phase, Start: b.Run.Start},
		Comparison: compSectionsJSON{
			K6: map[string]compPairJSON{
				"p95_ms":         {A: jsonNum(aK6.P95ms), B: jsonNum(bK6.P95ms)},
				"p90_ms":         {A: jsonNum(aK6.P90ms), B: jsonNum(bK6.P90ms)},
				"error_rate":     {A: jsonNum(aK6.ErrorRate), B: jsonNum(bK6.ErrorRate)},
				"checks_rate":    {A: jsonNum(aK6.ChecksRate), B: jsonNum(bK6.ChecksRate)},
				"total_requests": {A: jsonNumInt(aK6.TotalRequests), B: jsonNumInt(bK6.TotalRequests)},
				"rps_avg":        {A: jsonNum(aK6.RPSAvg), B: jsonNum(bK6.RPSAvg)},
				"vus_max":        {A: jsonNumInt(aK6.VUsMax), B: jsonNumInt(bK6.VUsMax)},
			},
			Infra: map[string]compPairJSON{
				"ecs_cpu_peak":    {A: jsonNumFromPeakAvg(aInfra.ECSCPU), B: jsonNumFromPeakAvg(bInfra.ECSCPU)},
				"ecs_memory_peak": {A: jsonNumFromPeakAvg(aInfra.ECSMemory), B: jsonNumFromPeakAvg(bInfra.ECSMemory)},
				"tasks_before":    {A: aInfra.Tasks.Before, B: bInfra.Tasks.Before},
				"tasks_after":     {A: aInfra.Tasks.After, B: bInfra.Tasks.After},
				"asg_before":      {A: aInfra.ASG.Before, B: bInfra.ASG.Before},
				"asg_after":       {A: aInfra.ASG.After, B: bInfra.ASG.After},
				"alb_5xx":         {A: aInfra.ALB5xx, B: bInfra.ALB5xx},
			},
		},
	}

	return json.MarshalIndent(out, "", "  ")
}

func jsonNum(v *float64) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func jsonNumInt(v *int) interface{} {
	if v == nil {
		return nil
	}
	return *v
}

func jsonNumFromPeakAvg(pa *PeakAvg) interface{} {
	if pa == nil || pa.Peak == nil {
		return nil
	}
	return *pa.Peak
}

