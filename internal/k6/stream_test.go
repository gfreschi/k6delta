package k6_test

import (
	"testing"

	"github.com/gfreschi/k6delta/internal/k6"
)

func TestParseK6JSONLine_metric(t *testing.T) {
	line := `{"type":"Point","data":{"time":"2026-03-04T12:00:00Z","value":234.5},"metric":"http_reqs","tags":{}}`

	point, err := k6.ParseJSONLine(line)
	if err != nil {
		t.Fatalf("ParseJSONLine() error = %v", err)
	}
	if point.Metric != "http_reqs" {
		t.Errorf("Metric = %q, want %q", point.Metric, "http_reqs")
	}
	if point.Value != 234.5 {
		t.Errorf("Value = %v, want 234.5", point.Value)
	}
}

func TestParseK6JSONLine_ignoresNonPoint(t *testing.T) {
	line := `{"type":"Metric","data":{"name":"http_reqs","type":"counter"}}`

	point, err := k6.ParseJSONLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if point != nil {
		t.Error("expected nil for non-Point type")
	}
}
