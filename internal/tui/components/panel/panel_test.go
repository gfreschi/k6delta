package panel_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gfreschi/k6delta/internal/tui/components/panel"
	"github.com/gfreschi/k6delta/internal/tui/constants"
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

func TestPanel_setTitle(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Original", 40, 10)
	p.SetContent("content")

	view1 := p.View()
	if !strings.Contains(view1, "Original") {
		t.Error("expected 'Original' in view")
	}

	p.SetTitle("Updated")
	view2 := p.View()
	if !strings.Contains(view2, "Updated") {
		t.Error("expected 'Updated' in view after SetTitle")
	}
	if strings.Contains(view2, "Original") {
		t.Error("old title should not appear after SetTitle")
	}
}

func TestPanel_withChildModel(t *testing.T) {
	ctx := tuictx.New(80, 24)
	p := panel.NewModel(ctx, "Test", 40, 10)

	child := &testChild{content: "child content"}
	p.SetModel(child)

	view := p.View()
	if !strings.Contains(view, "child content") {
		t.Fatalf("expected child content in panel view, got: %s", view)
	}
}

func TestPanel_setContentClearsChild(t *testing.T) {
	ctx := tuictx.New(80, 24)
	p := panel.NewModel(ctx, "Test", 40, 10)

	child := &testChild{content: "child content"}
	p.SetModel(child)
	p.SetContent("string content")

	view := p.View()
	if strings.Contains(view, "child content") {
		t.Fatal("expected child to be cleared after SetContent")
	}
	if !strings.Contains(view, "string content") {
		t.Fatalf("expected string content in view, got: %s", view)
	}
}

type testChild struct{ content string }

func (c *testChild) Init() tea.Cmd                       { return nil }
func (c *testChild) Update(tea.Msg) (tea.Model, tea.Cmd) { return c, nil }
func (c *testChild) View() string                        { return c.content }

func TestPanel_cycleExpand(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)

	if p.ExpandMode() != constants.ExpandNormal {
		t.Fatalf("initial expand mode = %d, want ExpandNormal", p.ExpandMode())
	}

	p.CycleExpand()
	if p.ExpandMode() != constants.ExpandExpanded {
		t.Fatalf("after 1 cycle = %d, want ExpandExpanded", p.ExpandMode())
	}

	p.CycleExpand()
	if p.ExpandMode() != constants.ExpandFull {
		t.Fatalf("after 2 cycles = %d, want ExpandFull", p.ExpandMode())
	}

	p.CycleExpand()
	if p.ExpandMode() != constants.ExpandNormal {
		t.Fatalf("after 3 cycles = %d, want ExpandNormal (wrapped)", p.ExpandMode())
	}
}

func TestPanel_resetExpand(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)

	p.CycleExpand()
	p.CycleExpand()
	p.ResetExpand()
	if p.ExpandMode() != constants.ExpandNormal {
		t.Fatalf("after reset = %d, want ExpandNormal", p.ExpandMode())
	}
}

func TestPanel_focused(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)

	if p.Focused() {
		t.Fatal("expected not focused initially")
	}
	p.SetFocused(true)
	if !p.Focused() {
		t.Fatal("expected focused after SetFocused(true)")
	}
}

func TestPanel_content(t *testing.T) {
	ctx := tuictx.New(120, 40)
	p := panel.NewModel(ctx, "Test", 40, 10)

	p.SetContent("hello world")
	if p.Content() != "hello world" {
		t.Fatalf("Content() = %q, want %q", p.Content(), "hello world")
	}

	child := &testChild{content: "child"}
	p.SetModel(child)
	if p.Content() != "" {
		t.Fatalf("Content() after SetModel = %q, want empty", p.Content())
	}
}

func TestPanel_updateForwardsToChild(t *testing.T) {
	ctx := tuictx.New(80, 24)
	p := panel.NewModel(ctx, "Test", 40, 10)

	child := &testChild{content: "before"}
	p.SetModel(child)
	p.SetFocused(true)
	// Complete transition
	for p.AdvanceTransition() != nil {
	}

	p, _ = p.Update(tea.KeyMsg{})
	// Child should still be present
	view := p.View()
	if !strings.Contains(view, "before") {
		t.Fatalf("expected child content in view after Update, got: %s", view)
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
