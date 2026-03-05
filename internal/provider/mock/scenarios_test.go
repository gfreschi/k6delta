package mock

import (
	"testing"
	"time"
)

func TestGetScenario_builtins(t *testing.T) {
	names := []string{"happy-path", "cpu-spike", "scale-out", "cascade-failure"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			s, err := GetScenario(name)
			if err != nil {
				t.Fatalf("GetScenario(%q) error: %v", name, err)
			}
			if s.Name != name {
				t.Errorf("Name = %q, want %q", s.Name, name)
			}
			if s.Description == "" {
				t.Error("Description is empty")
			}
			if s.Duration == 0 {
				t.Error("Duration is zero")
			}
			if len(s.Metrics) == 0 {
				t.Error("Metrics is empty")
			}
		})
	}
}

func TestGetScenario_unknown(t *testing.T) {
	_, err := GetScenario("nonexistent")
	if err == nil {
		t.Error("expected error for unknown scenario")
	}
}

func TestListScenarios(t *testing.T) {
	list := ListScenarios()
	if len(list) < 4 {
		t.Errorf("ListScenarios() returned %d, want >= 4", len(list))
	}
}

func TestScenario_metricsHaveCPUAndMemory(t *testing.T) {
	s, _ := GetScenario("happy-path")
	ids := make(map[string]bool)
	for _, m := range s.Metrics {
		ids[m.ID] = true
	}
	if !ids["service_cpu"] {
		t.Error("missing service_cpu metric")
	}
	if !ids["service_memory"] {
		t.Error("missing service_memory metric")
	}
}

func TestScenario_scaleOutHasActivities(t *testing.T) {
	s, _ := GetScenario("scale-out")
	if len(s.Activities) == 0 {
		t.Error("scale-out scenario should have activities")
	}
}

func TestScenario_snapshotCurve(t *testing.T) {
	s, _ := GetScenario("scale-out")
	if s.PreSnapshot.TaskRunning == s.PostSnapshot.TaskRunning {
		t.Error("scale-out pre and post task counts should differ")
	}
}

func TestScenario_cascadeHasAlarms(t *testing.T) {
	s, _ := GetScenario("cascade-failure")
	if len(s.Alarms) == 0 {
		t.Error("cascade-failure scenario should have alarms")
	}
}

func TestScenario_duration(t *testing.T) {
	s, _ := GetScenario("happy-path")
	if s.Duration < 30*time.Second {
		t.Errorf("Duration = %v, want >= 30s", s.Duration)
	}
}
