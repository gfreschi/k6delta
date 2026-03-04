package ecs

import (
	"testing"
)

func TestSetOnProgress(t *testing.T) {
	p := &Provider{}

	var calls []string
	p.SetOnProgress(func(id string, current, total int) {
		calls = append(calls, id)
	})

	if p.onProgress == nil {
		t.Fatal("onProgress is nil after SetOnProgress")
	}

	p.reportProgress("test", 1, 1)
	if len(calls) != 1 || calls[0] != "test" {
		t.Errorf("calls = %v, want [test]", calls)
	}
}

func TestReportProgress_NilSafe(t *testing.T) {
	p := &Provider{}
	// Should not panic when onProgress is nil
	p.reportProgress("test", 1, 1)
}
