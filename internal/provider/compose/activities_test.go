package compose

import (
	"context"
	"fmt"
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
	if len(activities.ServiceScaling) != 2 {
		t.Errorf("got %d activities, want 2", len(activities.ServiceScaling))
	}
	if activities.ServiceScaling[0].Description == "" {
		t.Error("expected non-empty description")
	}
	if activities.ServiceScaling[0].Status != "start" {
		t.Errorf("status = %q, want %q", activities.ServiceScaling[0].Status, "start")
	}
	if activities.ServiceScaling[1].Status != "die" {
		t.Errorf("status = %q, want %q", activities.ServiceScaling[1].Status, "die")
	}
}

func TestParseDockerEvents_empty(t *testing.T) {
	activities := parseDockerEvents(nil)
	if len(activities.ServiceScaling) != 0 {
		t.Errorf("got %d activities, want 0", len(activities.ServiceScaling))
	}
}

func TestFetchActivitiesWithClient_success(t *testing.T) {
	now := time.Now()
	mock := &mockEventEmitter{
		messages: []events.Message{
			{
				Type:   events.ContainerEventType,
				Action: events.ActionStart,
				Actor:  events.Actor{Attributes: map[string]string{"name": "web-1"}},
				Time:   now.Unix(),
			},
		},
	}

	p := &Provider{project: "myapp"}
	start := now.Add(-1 * time.Minute)
	end := now.Add(1 * time.Minute)

	activities, err := p.fetchActivitiesWithClient(context.Background(), mock, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(activities.ServiceScaling) != 1 {
		t.Errorf("got %d activities, want 1", len(activities.ServiceScaling))
	}
}

func TestFetchActivitiesWithClient_error(t *testing.T) {
	mock := &mockEventEmitter{
		err: fmt.Errorf("connection lost"),
	}

	p := &Provider{project: "myapp"}
	now := time.Now()
	_, err := p.fetchActivitiesWithClient(context.Background(), mock, now, now)
	if err == nil {
		t.Error("expected error from docker events, got nil")
	}
}
