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

func TestFooter_SetState(t *testing.T) {
	ctx := tuictx.New(120, 40)
	f := footer.NewModel(ctx, []footer.KeyHint{
		{Key: "q", Action: "quit"},
		{Key: "+", Action: "expand"},
	})

	// Normal state renders all hints
	view := f.View()
	if !strings.Contains(view, "expand") {
		t.Error("expected 'expand' in normal state")
	}

	// Expanded state replaces expand with collapse
	f.SetState(footer.StateExpanded)
	view = f.View()
	if !strings.Contains(view, "collapse") {
		t.Error("expected 'collapse' in expanded state")
	}

	// Help state shows only close hint
	f.SetState(footer.StateHelp)
	view = f.View()
	if !strings.Contains(view, "close") {
		t.Error("expected 'close' in help state")
	}

	// Back to normal
	f.SetState(footer.StateNormal)
	view = f.View()
	if !strings.Contains(view, "expand") {
		t.Error("expected 'expand' restored in normal state")
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

func TestFooter_ViewLabel(t *testing.T) {
	ctx := tuictx.New(120, 40)
	f := footer.NewModelWithHints(ctx, []footer.Hint{
		{Key: "q", Label: "quit", Short: "quit"},
	})
	f.SetViewLabel("staging | ecs")

	view := f.View()
	if !strings.Contains(view, "staging | ecs") {
		t.Fatalf("expected view label in footer, got: %s", view)
	}
}

func TestFooter_ViewLabelEmpty(t *testing.T) {
	ctx := tuictx.New(120, 40)
	f := footer.NewModelWithHints(ctx, []footer.Hint{
		{Key: "q", Label: "quit", Short: "quit"},
	})

	// No view label set -- should render without error
	view := f.View()
	if !strings.Contains(view, "quit") {
		t.Fatalf("expected hints in footer, got: %s", view)
	}
}
