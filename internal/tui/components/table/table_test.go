package table_test

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

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

func TestTableCellStyleFunc(t *testing.T) {
	ctx := testContext()
	tbl := table.NewModel(ctx, []table.Column{
		{Title: "Metric", Width: 20},
		{Title: "Value", Width: 10},
	})
	tbl.SetRows([]table.Row{{"CPU", "95%"}, {"Memory", "40%"}})

	called := false
	tbl.SetCellStyle(func(row, col int, value string) lipgloss.Style {
		called = true
		return lipgloss.NewStyle()
	})

	_ = tbl.View()
	if !called {
		t.Fatal("CellStyleFunc was never called")
	}
}

func TestTableGrow(t *testing.T) {
	ctx := testContext()
	tbl := table.NewModel(ctx, []table.Column{
		{Title: "Name", Width: 10},
		{Title: "Desc", Width: 10, Grow: true},
	})
	tbl.SetWidth(80)
	tbl.SetRows([]table.Row{{"foo", "bar"}})

	view := tbl.View()
	if len(view) == 0 {
		t.Fatal("table View() returned empty")
	}
	// Grow column should expand: 80 - 10 (fixed) - 1 (spacing) = 69
	if !strings.Contains(view, "bar") {
		t.Fatal("expected 'bar' in view")
	}
}

func TestSetWidth_redistributesColumns(t *testing.T) {
	ctx := testContext()
	cols := []table.Column{
		{Title: "Metric", Width: 26},
		{Title: "Value", Width: 12},
		{Title: "Delta", Width: 12},
	}
	m := table.NewModel(ctx, cols)
	m.SetRows([]table.Row{{"p95 latency", "150ms", "-10%"}})

	// Set width narrower than original total
	m.SetWidth(40)

	view := m.View()
	if view == "" {
		t.Error("View() returned empty string after SetWidth")
	}
}

func TestTruncateCell(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		maxWidth int
		want     string
	}{
		{"fits", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"very short max", "hello", 3, "hel"},
		{"empty", "", 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := table.TruncateCell(tt.value, tt.maxWidth)
			if got != tt.want {
				t.Errorf("TruncateCell(%q, %d) = %q, want %q", tt.value, tt.maxWidth, got, tt.want)
			}
		})
	}
}

func TestTableGrowNoWidth(t *testing.T) {
	ctx := testContext()
	tbl := table.NewModel(ctx, []table.Column{
		{Title: "Name", Width: 10},
		{Title: "Desc", Width: 10, Grow: true},
	})
	// No SetWidth — should fall back to original widths
	tbl.SetRows([]table.Row{{"foo", "bar"}})

	view := tbl.View()
	if !strings.Contains(view, "bar") {
		t.Fatal("expected 'bar' in view without SetWidth")
	}
}
