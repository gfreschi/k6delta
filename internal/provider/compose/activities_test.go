package compose

import (
	"testing"
	"time"

	"github.com/moby/moby/api/types/events"
)

func TestParseDockerEvents(t *testing.T) {
	now := time.Now()
	dockerEvents := []events.Message{
		{
			Type:   events.ContainerEventType,
			Action: events.ActionStart,
			Actor:  events.Actor{Attributes: map[string]string{"name": "web-1"}},
			Time:   now.Unix(),
		},
		{
			Type:   events.ContainerEventType,
			Action: events.ActionDie,
			Actor:  events.Actor{Attributes: map[string]string{"name": "web-2"}},
			Time:   now.Add(30 * time.Second).Unix(),
		},
	}

	activities := parseDockerEvents(dockerEvents)
	if len(activities.ECSScaling) != 2 {
		t.Errorf("got %d activities, want 2", len(activities.ECSScaling))
	}
	if activities.ECSScaling[0].Description == "" {
		t.Error("expected non-empty description")
	}
	if activities.ECSScaling[0].Status != "start" {
		t.Errorf("status = %q, want %q", activities.ECSScaling[0].Status, "start")
	}
	if activities.ECSScaling[1].Status != "die" {
		t.Errorf("status = %q, want %q", activities.ECSScaling[1].Status, "die")
	}
}

func TestParseDockerEvents_empty(t *testing.T) {
	activities := parseDockerEvents(nil)
	if len(activities.ECSScaling) != 0 {
		t.Errorf("got %d activities, want 0", len(activities.ECSScaling))
	}
}
