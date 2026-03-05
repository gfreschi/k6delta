package footer_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/footer"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestFooter_rendersKeyHints(t *testing.T) {
	ctx := tuictx.New(120, 40)
	f := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "tab", Action: "next panel"},
	})
	view := f.View()
	if !strings.Contains(view, "quit") {
		t.Error("expected 'quit' in footer view")
	}
}

func TestFooterResponsiveCollapse(t *testing.T) {
	ctx := tuictx.New(80, 24)

	hints := []footer.Hint{
		{Key: "tab", Label: "next panel", Short: "next"},
		{Key: "q", Label: "quit", Short: "quit"},
	}

	f := footer.NewModelWithHints(ctx, hints)

	// Wide: full labels
	f.SetWidth(120)
	wide := f.View()
	if !strings.Contains(wide, "next panel") {
		t.Fatalf("expected full label at width=120, got: %s", wide)
	}

	// Narrow: short labels
	f.SetWidth(90)
	narrow := f.View()
	if !strings.Contains(narrow, "next") {
		t.Fatalf("expected short label at width=90, got: %s", narrow)
	}
	if strings.Contains(narrow, "next panel") {
		t.Fatalf("expected short label at width=90, not full label, got: %s", narrow)
	}
}

func TestFooterKeyHintBackwardCompat(t *testing.T) {
	ctx := tuictx.New(120, 40)
	f := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
	})

	// SetHints should work with legacy type
	f.SetHints([]footer.KeyHint{
		{Key: "e", Action: "export"},
	})

	view := f.View()
	if !strings.Contains(view, "export") {
		t.Fatalf("expected 'export' after SetHints, got: %s", view)
	}
}
