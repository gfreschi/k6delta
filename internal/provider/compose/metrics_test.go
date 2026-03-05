package compose

import (
	"testing"

	"github.com/moby/moby/api/types/container"
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
	if !ids["container_cpu"] {
		t.Error("missing container_cpu metric")
	}
	if !ids["container_memory"] {
		t.Error("missing container_memory metric")
	}

	for _, r := range results {
		if r.ID == "container_cpu" {
			if *r.Peak != 72.1 {
				t.Errorf("container_cpu peak = %v, want 72.1", *r.Peak)
			}
			wantAvg := (45.2 + 72.1) / 2
			if *r.Avg != wantAvg {
				t.Errorf("container_cpu avg = %v, want %v", *r.Avg, wantAvg)
			}
		}
		if r.ID == "container_memory" {
			if *r.Peak != 512.0 {
				t.Errorf("container_memory peak = %v, want 512.0", *r.Peak)
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
