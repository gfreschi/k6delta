package table_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/components/table"
	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

func testContext() *tuictx.ProgramContext {
	return tuictx.New(120, 40)
}

func TestTable_rendersHeadersAndRows(t *testing.T) {
	ctx := testContext()
	tbl := table.NewModel(ctx, []table.Column{
		{Title: "Metric", Width: 20},
		{Title: "Value", Width: 10},
	})
	tbl.SetRows([]table.Row{
		{"p95 latency", "456ms"},
		{"Error rate", "0.00%"},
	})

	view := tbl.View()
	if !strings.Contains(view, "Metric") {
		t.Error("expected header 'Metric' in view")
	}
	if !strings.Contains(view, "456ms") {
		t.Error("expected '456ms' in view")
	}
}

func TestTable_emptyRows(t *testing.T) {
	ctx := testContext()
	tbl := table.NewModel(ctx, []table.Column{
		{Title: "Name", Width: 20},
	})
	tbl.SetRows(nil)

	view := tbl.View()
	if !strings.Contains(view, "Name") {
		t.Error("expected header even with empty rows")
	}
}
