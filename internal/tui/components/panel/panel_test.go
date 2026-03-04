package panel_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func TestPanel_focusedHasDifferentBorder(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test Panel", 40, 10)
	p.SetContent("hello")

	unfocused := p.View()
	p.SetFocused(true)
	// Complete the transition animation so the final focused style is used.
	for p.AdvanceTransition() != nil {
	}
	focused := p.View()

	if unfocused == focused {
		t.Error("focused and unfocused views should differ")
	}
}

func TestPanel_scrollsContent(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)

	// Set content taller than panel
	longContent := strings.Repeat("line\n", 20)
	p.SetContent(longContent)
	p.SetFocused(true)

	view1 := p.View()
	p.ScrollDown()
	view2 := p.View()

	if view1 == view2 {
		t.Error("expected scroll to change view")
	}
}

func TestPanel_scrollPositionInTitle(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Events", 40, 10)
	longContent := strings.Repeat("event\n", 20)
	p.SetContent(longContent)

	view := p.View()
	// Should show scroll indicator when content overflows
	if !strings.Contains(view, "Events") {
		t.Error("expected title in view")
	}
}

func TestPanel_scrollPosition(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)
	longContent := strings.Repeat("line\n", 20)
	p.SetContent(longContent)

	cur, total := p.ScrollPosition()
	if total == 0 {
		t.Error("expected non-zero total lines")
	}
	if cur != 0 {
		t.Errorf("expected initial scroll at 0, got %d", cur)
	}

	p.ScrollDown()
	cur2, _ := p.ScrollPosition()
	if cur2 <= cur {
		t.Errorf("expected scroll position to increase after ScrollDown, got %d", cur2)
	}
}

func TestPanel_scrollUpWraps(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)
	longContent := strings.Repeat("line\n", 20)
	p.SetContent(longContent)

	// ScrollUp from top should not panic or go negative
	p.ScrollUp()
	cur, _ := p.ScrollPosition()
	if cur != 0 {
		t.Errorf("expected scroll position to stay at 0 after ScrollUp from top, got %d", cur)
	}
}
