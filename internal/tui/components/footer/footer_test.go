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
