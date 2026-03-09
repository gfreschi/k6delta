package common

import "github.com/charmbracelet/lipgloss"

// CompactItem is a label-value pair for narrow terminal text list rendering.
type CompactItem struct {
	Label string
	Value string
}

// RenderTileGrid arranges tile views into rows of tilesPerRow with 1-line gaps between rows.
func RenderTileGrid(tiles []string, tilesPerRow int) string {
	if len(tiles) == 0 {
		return ""
	}
	var rows []string
	for i := 0; i < len(tiles); i += tilesPerRow {
		end := i + tilesPerRow
		if end > len(tiles) {
			end = len(tiles)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, tiles[i:end]...)
		rows = append(rows, row)
	}
	// Join rows with empty line gap between them
	result := rows[0]
	for i := 1; i < len(rows); i++ {
		result = lipgloss.JoinVertical(lipgloss.Left, result, "", rows[i])
	}
	return result
}

// RenderCompactList renders items as a simple text list for narrow terminals (< 80 width).
func RenderCompactList(items []CompactItem) string {
	if len(items) == 0 {
		return ""
	}
	var lines []string
	for _, item := range items {
		lines = append(lines, "  "+item.Label+": "+item.Value)
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
