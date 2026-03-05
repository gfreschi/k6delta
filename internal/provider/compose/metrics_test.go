package compose

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

func TestComputeCPUPercent(t *testing.T) {
	stats := container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 200},
			SystemUsage: 1000,
			OnlineCPUs:  2,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 100},
			SystemUsage: 500,
		},
	}

	// cpu_delta=100, system_delta=500, cpus=2 → (100/500)*2*100 = 40%
	got := computeCPUPercent(stats)
	if got != 40.0 {
		t.Errorf("computeCPUPercent = %v, want 40.0", got)
	}
}

func TestComputeCPUPercent_zeroDelta(t *testing.T) {
	stats := container.StatsResponse{
		CPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 100},
			SystemUsage: 500,
			OnlineCPUs:  2,
		},
		PreCPUStats: container.CPUStats{
			CPUUsage:    container.CPUUsage{TotalUsage: 100},
			SystemUsage: 500,
		},
	}

	got := computeCPUPercent(stats)
	if got != 0.0 {
		t.Errorf("computeCPUPercent = %v, want 0.0", got)
	}
}

func TestComputeMemoryMB(t *testing.T) {
	stats := container.StatsResponse{
		MemoryStats: container.MemoryStats{
			Usage: 256 * 1024 * 1024, // 256 MB
		},
	}

	got := computeMemoryMB(stats)
	if got != 256.0 {
		t.Errorf("computeMemoryMB = %v, want 256.0", got)
	}
}

func TestAggregateStats_returnsCPUAndMemory(t *testing.T) {
	results := aggregateStats([]containerStat{
		{cpuPercent: 45.2, memoryMB: 256.0},
		{cpuPercent: 72.1, memoryMB: 512.0},
	})

	ids := make(map[string]bool)
	for _, r := range results {
		ids[r.ID] = true
		if r.Peak == nil || r.Avg == nil {
			t.Errorf("metric %s has nil Peak or Avg", r.ID)
		}
	}
	if !ids["service_cpu"] {
		t.Error("missing service_cpu metric")
	}
	if !ids["service_memory"] {
		t.Error("missing service_memory metric")
	}

	for _, r := range results {
		if r.ID == "service_cpu" {
			if *r.Peak != 72.1 {
				t.Errorf("service_cpu peak = %v, want 72.1", *r.Peak)
			}
			wantAvg := (45.2 + 72.1) / 2
			if *r.Avg != wantAvg {
				t.Errorf("service_cpu avg = %v, want %v", *r.Avg, wantAvg)
			}
		}
		if r.ID == "service_memory" {
			if *r.Peak != 512.0 {
				t.Errorf("service_memory peak = %v, want 512.0", *r.Peak)
			}
		}
	}
}

func TestAggregateStats_empty(t *testing.T) {
	results := aggregateStats(nil)
	if results != nil {
		t.Errorf("expected nil for empty stats, got %v", results)
	}
}

func statsBody(stats container.StatsResponse) io.ReadCloser {
	data, _ := json.Marshal(stats)
	return io.NopCloser(strings.NewReader(string(data)))
}

func TestFetchMetricsWithClient_success(t *testing.T) {
	mock := &mockStatter{
		mockDocker: mockDocker{
			containers: []container.Summary{
				{ID: "c1", State: container.StateRunning},
			},
		},
		statsResults: map[string]client.ContainerStatsResult{
			"c1": {
				Body: statsBody(container.StatsResponse{
					CPUStats: container.CPUStats{
						CPUUsage:    container.CPUUsage{TotalUsage: 200},
						SystemUsage: 1000,
						OnlineCPUs:  2,
					},
					PreCPUStats: container.CPUStats{
						CPUUsage:    container.CPUUsage{TotalUsage: 100},
						SystemUsage: 500,
					},
					MemoryStats: container.MemoryStats{
						Usage: 128 * 1024 * 1024,
					},
				}),
			},
		},
	}

	p := &Provider{project: "myapp"}
	results, err := p.fetchMetricsWithClient(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}

	ids := make(map[string]bool)
	for _, r := range results {
		ids[r.ID] = true
	}
	if !ids["service_cpu"] {
		t.Error("missing service_cpu metric")
	}
	if !ids["service_memory"] {
		t.Error("missing service_memory metric")
	}
}

func TestFetchMetricsWithClient_noContainers(t *testing.T) {
	mock := &mockStatter{
		mockDocker: mockDocker{
			containers: []container.Summary{},
		},
	}

	p := &Provider{project: "myapp"}
	results, err := p.fetchMetricsWithClient(context.Background(), mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results for no containers, got %v", results)
	}
}
