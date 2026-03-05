package compose

import (
	"context"
	"fmt"
	"time"

	"github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"

	"github.com/gfreschi/k6delta/internal/provider"
)

func (p *Provider) fetchActivitiesWithClient(ctx context.Context, cli interface {
	Events(ctx context.Context, options client.EventsListOptions) client.EventsResult
}, start, end time.Time) (provider.Activities, error) {
	p.reportProgress("Activities", 1, 1)

	f := make(client.Filters).
		Add("label", fmt.Sprintf("com.docker.compose.project=%s", p.project)).
		Add("type", string(events.ContainerEventType)).
		Add("event", string(events.ActionStart), string(events.ActionDie), string(events.ActionRestart), string(events.ActionStop))

	result := cli.Events(ctx, client.EventsListOptions{
		Since:   start.Format(time.RFC3339),
		Until:   end.Format(time.RFC3339),
		Filters: f,
	})

	var msgs []events.Message
	for {
		select {
		case event, ok := <-result.Messages:
			if !ok {
				return parseDockerEvents(msgs), nil
			}
			msgs = append(msgs, event)
		case err, ok := <-result.Err:
			if !ok {
				return parseDockerEvents(msgs), nil
			}
			if err != nil {
				return parseDockerEvents(msgs), nil
			}
		}
	}
}

func parseDockerEvents(msgs []events.Message) provider.Activities {
	var activities []provider.ScalingActivity
	for _, msg := range msgs {
		name := msg.Actor.Attributes["name"]
		activities = append(activities, provider.ScalingActivity{
			Time:        time.Unix(msg.Time, 0).Format(time.RFC3339),
			Status:      string(msg.Action),
			Description: fmt.Sprintf("Container %s: %s", name, msg.Action),
		})
	}
	return provider.Activities{
		ECSScaling: activities,
	}
}
