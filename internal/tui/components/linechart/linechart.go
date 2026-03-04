// Package linechart provides a Unicode line chart for real-time data.
package linechart

import (
	"fmt"
	"math"
	"strings"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// Model is the line chart Bubble Tea model.
type Model struct {
	ctx       *tuictx.ProgramContext
	title     string
	unit      string
	width     int
	height    int
	data      []float64
	maxPoints int
}

// NewModel creates a line chart with title, unit label, and dimensions.
func NewModel(ctx *tuictx.ProgramContext, title, unit string, width, height int) Model {
	return Model{
		ctx:       ctx,
		title:     title,
		unit:      unit,
		width:     width,
		height:    height,
		maxPoints: width - 6, // reserve space for Y axis labels
	}
}

// AddPoint appends a data point, maintaining rolling window.
func (m *Model) AddPoint(value float64) {
	m.data = append(m.data, value)
	if len(m.data) > m.maxPoints {
		m.data = m.data[len(m.data)-m.maxPoints:]
	}
}

// UpdateContext updates the shared context.
func (m *Model) UpdateContext(ctx *tuictx.ProgramContext) {
	m.ctx = ctx
}

// View renders the chart using Unicode box-drawing characters.
func (m Model) View() string {
	s := m.ctx.Styles
	var b strings.Builder

	// Title
	b.WriteString(s.Header.Root.Render(fmt.Sprintf("─ %s (%s) ", m.title, m.unit)))
	b.WriteString("\n")

	if len(m.data) == 0 {
		for i := 0; i < m.height-1; i++ {
			b.WriteString(s.Common.FaintTextStyle.Render("  waiting for data..."))
			if i < m.height-2 {
				b.WriteString("\n")
			}
		}
		return b.String()
	}

	// Compute Y axis range
	minVal, maxVal := m.data[0], m.data[0]
	for _, v := range m.data {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	if maxVal == minVal {
		maxVal = minVal + 1
	}
	// Round up maxVal for clean axis labels
	maxVal = math.Ceil(maxVal/10) * 10

	plotWidth := m.width - 6  // Y axis label width
	plotHeight := m.height - 1 // title takes 1 line

	// Build character grid
	grid := make([][]rune, plotHeight)
	for i := range grid {
		grid[i] = make([]rune, plotWidth)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Plot data points
	for i, v := range m.data {
		if i >= plotWidth {
			break
		}
		row := plotHeight - 1 - int(float64(plotHeight-1)*(v-minVal)/(maxVal-minVal))
		if row < 0 {
			row = 0
		}
		if row >= plotHeight {
			row = plotHeight - 1
		}
		grid[row][i] = '●'

		// Connect to previous point with line
		if i > 0 {
			prevRow := plotHeight - 1 - int(float64(plotHeight-1)*(m.data[i-1]-minVal)/(maxVal-minVal))
			if prevRow < 0 {
				prevRow = 0
			}
			if prevRow >= plotHeight {
				prevRow = plotHeight - 1
			}
			if prevRow == row {
				grid[row][i] = '─'
			} else if prevRow < row {
				grid[row][i] = '╯'
				for r := prevRow + 1; r < row; r++ {
					grid[r][i] = '│'
				}
			} else {
				grid[row][i] = '╮'
				for r := row + 1; r < prevRow; r++ {
					grid[r][i] = '│'
				}
			}
		}
	}

	// Render with Y axis labels
	for row := 0; row < plotHeight; row++ {
		yVal := maxVal - float64(row)*(maxVal-minVal)/float64(plotHeight-1)
		label := fmt.Sprintf("%4.0f", yVal)
		if row == 0 || row == plotHeight-1 {
			b.WriteString(s.Common.FaintTextStyle.Render(label + "┤"))
		} else {
			b.WriteString(s.Common.FaintTextStyle.Render("    │"))
		}
		b.WriteString(string(grid[row]))
		if row < plotHeight-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}
