package header_test

import (
	"strings"
	"testing"
	"time"

	"github.com/gfreschi/k6delta/internal/tui/components/header"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestHeaderBasicView(t *testing.T) {
	ctx := tuictx.New(80, 24)
	h := header.NewModel(ctx, "myapp", "prod", "ramp-up")

	view := h.View()
	if !strings.Contains(view, "myapp") {
		t.Fatalf("expected app name in header, got: %s", view)
	}
	if !strings.Contains(view, "prod") {
		t.Fatalf("expected env in header, got: %s", view)
	}
	if !strings.Contains(view, "ramp-up") {
		t.Fatalf("expected phase in header, got: %s", view)
	}
}

func TestHeaderElapsedDisplay(t *testing.T) {
	ctx := tuictx.New(80, 24)
	h := header.NewModel(ctx, "myapp", "prod", "ramp-up")
	h.SetElapsed(2*time.Minute + 34*time.Second)
	h.SetStatus("Running")

	view := h.View()
	if !strings.Contains(view, "2m 34s") {
		t.Fatalf("expected elapsed time in header, got: %s", view)
	}
	if !strings.Contains(view, "Running") {
		t.Fatalf("expected status in header, got: %s", view)
	}
}

func TestHeaderSetStatus(t *testing.T) {
	ctx := tuictx.New(80, 24)
	h := header.NewModel(ctx, "myapp", "prod", "ramp-up")
	h.SetStatus("Done")

	view := h.View()
	if !strings.Contains(view, "Done") {
		t.Fatalf("expected status in header, got: %s", view)
	}
}

func TestHeaderSpinnerUpdate(t *testing.T) {
	ctx := tuictx.New(80, 24)
	h := header.NewModel(ctx, "myapp", "prod", "ramp-up")
	h.SetActive(true)

	// Update should not panic
	h2, _ := h.Update(nil)
	view := h2.View()
	if len(view) == 0 {
		t.Fatal("header View() returned empty")
	}
}
