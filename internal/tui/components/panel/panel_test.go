package panel_test

import (
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
	focused := p.View()

	if unfocused == focused {
		t.Error("focused and unfocused views should differ")
	}
}
