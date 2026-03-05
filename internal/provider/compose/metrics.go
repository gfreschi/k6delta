package compose

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/gfreschi/k6delta/internal/provider"
)

type containerStat struct {
	cpuPercent float64
	memoryMB   float64
}

// containerStatter abstracts Docker container stats for testability.
type containerStatter interface {
	containerLister
	ContainerStats(ctx context.Context, containerID string, options client.ContainerStatsOptions) (client.ContainerStatsResult, error)
}

func (p *Provider) fetchMetricsWithClient(ctx context.Context, cli containerStatter) ([]provider.MetricResult, error) {
	p.reportProgress("Metrics", 1, 2)

	opts := client.ContainerListOptions{
		Filters: make(client.Filters).Add("label", fmt.Sprintf("com.docker.compose.project=%s", p.project)).Add("status", "running"),
	}

	result, err := cli.ContainerList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}

	var stats []containerStat
	for _, c := range result.Items {
		stat, err := collectContainerStats(ctx, cli, c.ID)
		if err != nil {
			continue // skip containers that fail stats collection
		}
		stats = append(stats, stat)
	}

	p.reportProgress("Metrics", 2, 2)
	return aggregateStats(stats), nil
}

func collectContainerStats(ctx context.Context, cli containerStatter, containerID string) (containerStat, error) {
	statsResult, err := cli.ContainerStats(ctx, containerID, client.ContainerStatsOptions{
		IncludePreviousSample: true,
	})
	if err != nil {
		return containerStat{}, fmt.Errorf("container stats: %w", err)
	}
	defer func() { _ = statsResult.Body.Close() }()

	var statsResp container.StatsResponse
	if err := json.NewDecoder(statsResult.Body).Decode(&statsResp); err != nil {
		return containerStat{}, fmt.Errorf("decode stats: %w", err)
	}

	return containerStat{
		cpuPercent: computeCPUPercent(statsResp),
		memoryMB:   computeMemoryMB(statsResp),
	}, nil
}

// computeCPUPercent calculates CPU usage percentage from Docker stats.
// Formula: (cpu_delta / system_delta) * online_cpus * 100
func computeCPUPercent(stats container.StatsResponse) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta <= 0 || cpuDelta <= 0 {
		return 0.0
	}

	cpus := float64(stats.CPUStats.OnlineCPUs)
	if cpus == 0 {
		cpus = 1
	}

	return (cpuDelta / systemDelta) * cpus * 100.0
}

// computeMemoryMB returns memory usage in megabytes.
func computeMemoryMB(stats container.StatsResponse) float64 {
	return float64(stats.MemoryStats.Usage) / (1024 * 1024)
}

// aggregateStats computes peak and average across containers.
func aggregateStats(stats []containerStat) []provider.MetricResult {
	if len(stats) == 0 {
		return nil
	}

	cpuValues := make([]float64, len(stats))
	memValues := make([]float64, len(stats))
	for i, s := range stats {
		cpuValues[i] = s.cpuPercent
		memValues[i] = s.memoryMB
	}

	return []provider.MetricResult{
		metricFromValues("service_cpu", cpuValues),
		metricFromValues("service_memory", memValues),
	}
}

func metricFromValues(id string, values []float64) provider.MetricResult {
	mr := provider.MetricResult{
		ID:     id,
		Values: values,
	}
	if len(values) > 0 {
		peak := values[0]
		sum := 0.0
		for _, v := range values {
			sum += v
			if v > peak {
				peak = v
			}
		}
		avg := sum / float64(len(values))
		mr.Peak = &peak
		mr.Avg = &avg
	}
	return mr
}

