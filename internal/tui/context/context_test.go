package context_test

import (
	"testing"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestNewContextDefaultsActiveView(t *testing.T) {
	ctx := tuictx.New(80, 24)
	if ctx.ActiveView != tuictx.ViewBrowsing {
		t.Fatalf("expected ActiveView=ViewBrowsing, got %d", ctx.ActiveView)
	}
}

func TestViewTypeConstants(t *testing.T) {
	if tuictx.ViewBrowsing != 0 {
		t.Fatal("ViewBrowsing should be 0")
	}
	if tuictx.ViewRunning != 1 {
		t.Fatal("ViewRunning should be 1")
	}
	if tuictx.ViewAnalyzing != 2 {
		t.Fatal("ViewAnalyzing should be 2")
	}
	if tuictx.ViewComparing != 3 {
		t.Fatal("ViewComparing should be 3")
	}
	if tuictx.ViewReport != 4 {
		t.Fatal("ViewReport should be 4")
	}
}
