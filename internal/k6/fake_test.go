package k6

import (
	"testing"
	"time"
)

func TestFakeStream_sendsPoints(t *testing.T) {
	ch := make(chan K6Point, 256)
	duration := 500 * time.Millisecond

	go func() {
		FakeStream(duration, 1.0, "cascade-failure", ch)
		close(ch)
	}()

	var count int
	var hasReqs, hasDuration, hasFailed bool
	for p := range ch {
		count++
		switch p.Metric {
		case "http_reqs":
			hasReqs = true
		case "http_req_duration":
			hasDuration = true
		case "http_req_failed":
			hasFailed = true
		}
	}

	if count == 0 {
		t.Error("FakeStream sent 0 points")
	}
	if !hasReqs {
		t.Error("missing http_reqs points")
	}
	if !hasDuration {
		t.Error("missing http_req_duration points")
	}
	if !hasFailed {
		t.Error("missing http_req_failed points")
	}
}

func TestFakeStream_speedMultiplier(t *testing.T) {
	ch := make(chan K6Point, 256)

	start := time.Now()
	FakeStream(1*time.Second, 10.0, "happy-path", ch) // 10x speed = ~100ms real
	close(ch)
	elapsed := time.Since(start)

	if elapsed > 500*time.Millisecond {
		t.Errorf("10x speed took %v, expected < 500ms", elapsed)
	}
}

func TestFakeStream_resultTiming(t *testing.T) {
	ch := make(chan K6Point, 256)
	duration := 200 * time.Millisecond

	result := FakeStream(duration, 1.0, "happy-path", ch)
	close(ch)

	if result.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0 for happy-path", result.ExitCode)
	}
	if result.EndTime.Before(result.StartTime) {
		t.Error("EndTime before StartTime")
	}
}

func TestFakeStream_cascadeExitCode(t *testing.T) {
	ch := make(chan K6Point, 256)
	result := FakeStream(200*time.Millisecond, 1.0, "cascade-failure", ch)
	close(ch)

	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1 for cascade-failure", result.ExitCode)
	}
}
