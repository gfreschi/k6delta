// Package table provides a reusable styled table component.
package table

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Column defines a table column.
type Column struct {
	Title string
	Width int
	Grow  bool
	Align lipgloss.Position
}

// Row is a slice of cell strings.
type Row []string

// Model is the table Bubble Tea model.
type Model struct {
	ctx     *tuictx.ProgramContext
	columns []Column
	rows    []Row
	width   int
}

// NewModel creates a table with the given columns.
func NewModel(ctx *tuictx.ProgramContext, columns []Column) Model {
	return Model{ctx: ctx, columns: columns}
}

// SetRows replaces the table rows.
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
}

// SetWidth sets the available width for the table.
func (m *Model) SetWidth(w int) {
	m.width = w
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the table.
func (m Model) View() string {
	if len(m.columns) == 0 {
		return ""
	}

	s := m.ctx.Styles
	var b strings.Builder

	// Header row
	var headerCells []string
	for _, col := range m.columns {
		cell := fmt.Sprintf("%-*s", col.Width, col.Title)
		headerCells = append(headerCells, s.Table.Header.Render(cell))
	}
	b.WriteString(strings.Join(headerCells, " "))
	b.WriteString("\n")

	// Separator
	totalWidth := m.totalWidth()
	b.WriteString(s.Table.Separator.Render(strings.Repeat("─", totalWidth)))
	b.WriteString("\n")

	// Data rows
	for i, row := range m.rows {
		style := s.Table.Row
		if i%2 == 1 {
			style = s.Table.RowAlt
		}
		var cells []string
		for j, col := range m.columns {
			val := ""
			if j < len(row) {
				val = row[j]
			}
			align := lipgloss.Left
			if col.Align != 0 {
				align = col.Align
			}
			cell := lipgloss.NewStyle().Width(col.Width).Align(align).Render(val)
			cells = append(cells, style.Render(cell))
		}
		b.WriteString(strings.Join(cells, " "))
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) totalWidth() int {
	w := 0
	for _, col := range m.columns {
		w += col.Width
	}
	w += len(m.columns) - 1 // spaces between columns
	return w
}
