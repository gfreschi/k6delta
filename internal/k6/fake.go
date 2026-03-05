package k6

import (
	"math"
	"math/rand/v2"
	"time"
)

// scenarioExitCodes maps scenario names to the exit code they should produce.
var scenarioExitCodes = map[string]int{
	"happy-path":      0,
	"cpu-spike":       0,
	"scale-out":       0,
	"cascade-failure": 1,
}

// FakeStream generates synthetic K6Points and sends them to the channel.
// It simulates a k6 test running for the given duration at the given speed.
// The caller must close the channel after this returns.
func FakeStream(duration time.Duration, speed float64, scenario string, points chan<- K6Point) RunResult {
	if speed <= 0 {
		speed = 1.0
	}

	realDuration := time.Duration(float64(duration) / speed)
	tickInterval := 100 * time.Millisecond / time.Duration(speed)
	if tickInterval < 10*time.Millisecond {
		tickInterval = 10 * time.Millisecond
	}

	result := RunResult{
		StartTime: time.Now(),
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	start := time.Now()
	deadline := start.Add(realDuration)

	for now := range ticker.C {
		if now.After(deadline) {
			break
		}

		elapsed := now.Sub(start)
		t := float64(elapsed) / float64(realDuration) // normalized [0, 1]

		ts := result.StartTime.Add(time.Duration(float64(duration) * t)).Format(time.RFC3339Nano)

		// http_reqs: counter, increases with time
		points <- K6Point{
			Metric: "http_reqs",
			Value:  1.0,
			Time:   ts,
		}

		// http_req_duration: trend, varies by scenario
		baseDuration := 120.0
		switch scenario {
		case "cascade-failure":
			baseDuration = 120.0 + 3000.0*t*t // exponential rise
		case "cpu-spike":
			baseDuration = 120.0 + 200.0*math.Sin(math.Pi*t)
		}
		points <- K6Point{
			Metric: "http_req_duration",
			Value:  baseDuration + (rand.Float64()*40 - 20),
			Time:   ts,
		}

		// http_req_failed: rate, spikes in cascade-failure
		failRate := 0.001
		if scenario == "cascade-failure" && t > 0.6 {
			failRate = 0.15 + rand.Float64()*0.1
		}
		if rand.Float64() < failRate {
			points <- K6Point{
				Metric: "http_req_failed",
				Value:  1.0,
				Time:   ts,
			}
		}
	}

	// Guarantee at least one http_req_failed point for failure scenarios
	if scenarioExitCodes[scenario] != 0 {
		ts := result.StartTime.Add(duration).Format(time.RFC3339Nano)
		points <- K6Point{
			Metric: "http_req_failed",
			Value:  1.0,
			Time:   ts,
		}
	}

	result.EndTime = time.Now()
	result.ExitCode = scenarioExitCodes[scenario]
	return result
}
