// Package table provides a reusable styled table component.
package table

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// CellStyleFunc returns a style for a specific cell by row, column, and value.
type CellStyleFunc func(row, col int, value string) lipgloss.Style

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
	ctx       *tuictx.ProgramContext
	columns   []Column
	rows      []Row
	width     int
	cellStyle CellStyleFunc
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

// SetCellStyle sets a function that provides per-cell styling.
func (m *Model) SetCellStyle(fn CellStyleFunc) {
	m.cellStyle = fn
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

	cols := m.resolvedColumns()
	s := m.ctx.Styles
	var b strings.Builder

	// Header row
	var headerCells []string
	for _, col := range cols {
		cell := fmt.Sprintf("%-*s", col.Width, col.Title)
		headerCells = append(headerCells, s.Table.Header.Render(cell))
	}
	b.WriteString(strings.Join(headerCells, " "))
	b.WriteString("\n")

	// Separator
	totalWidth := m.totalWidthResolved(cols)
	b.WriteString(s.Table.Separator.Render(strings.Repeat("─", totalWidth)))
	b.WriteString("\n")

	// Data rows
	for i, row := range m.rows {
		rowStyle := s.Table.Row
		if i%2 == 1 {
			rowStyle = s.Table.RowAlt
		}
		var cells []string
		for j, col := range cols {
			val := ""
			if j < len(row) {
				val = row[j]
			}
			val = TruncateCell(val, col.Width)
			align := lipgloss.Left
			if col.Align != 0 {
				align = col.Align
			}

			cellContent := s.Table.Cell.Width(col.Width).Align(align).Render(val)

			if m.cellStyle != nil {
				cells = append(cells, m.cellStyle(i, j, val).Render(cellContent))
			} else {
				cells = append(cells, rowStyle.Render(cellContent))
			}
		}
		b.WriteString(strings.Join(cells, " "))
		if i < len(m.rows)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// TruncateCell truncates a cell value to maxWidth, adding "..." if truncated.
func TruncateCell(value string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= maxWidth {
		return value
	}
	if maxWidth <= 3 {
		return value[:maxWidth]
	}
	// Truncate rune-by-rune, tracking width incrementally to avoid O(n^2).
	budget := maxWidth - 3 // reserve space for "..."
	var b strings.Builder
	width := 0
	for _, r := range value {
		rw := lipgloss.Width(string(r))
		if width+rw > budget {
			break
		}
		b.WriteRune(r)
		width += rw
	}
	b.WriteString("...")
	return b.String()
}

// resolvedColumns returns columns with Grow columns expanded to fill available width.
// When m.width is set and no Grow columns exist, scales all columns proportionally.
func (m Model) resolvedColumns() []Column {
	if m.width == 0 {
		return m.columns
	}

	growCount := 0
	originalTotal := 0
	for _, c := range m.columns {
		originalTotal += c.Width
		if c.Grow {
			growCount++
		}
	}

	spacing := len(m.columns) - 1
	resolved := make([]Column, len(m.columns))
	copy(resolved, m.columns)

	if growCount > 0 {
		fixed := 0
		for _, c := range m.columns {
			if !c.Grow {
				fixed += c.Width
			}
		}
		remaining := m.width - fixed - spacing
		if remaining < 0 {
			remaining = 0
		}
		growWidth := remaining / growCount
		for i := range resolved {
			if resolved[i].Grow {
				resolved[i].Width = max(growWidth, 4)
			}
		}
		return resolved
	}

	// No Grow columns: redistribute proportionally
	available := m.width - spacing
	if available <= 0 || originalTotal <= 0 {
		return resolved
	}
	ratio := float64(available) / float64(originalTotal)
	if ratio >= 1.0 {
		return resolved
	}
	for i := range resolved {
		w := int(float64(resolved[i].Width) * ratio)
		if w < 4 {
			w = 4
		}
		resolved[i].Width = w
	}
	return resolved
}

func (m Model) totalWidthResolved(cols []Column) int {
	w := 0
	for _, col := range cols {
		w += col.Width
	}
	w += len(cols) - 1 // spaces between columns
	return w
}
