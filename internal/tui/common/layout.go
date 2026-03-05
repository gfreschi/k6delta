package common

import "github.com/charmbracelet/lipgloss"

// RenderTileGrid arranges tile views into rows of tilesPerRow.
func RenderTileGrid(tiles []string, tilesPerRow int) string {
	var rows []string
	for i := 0; i < len(tiles); i += tilesPerRow {
		end := i + tilesPerRow
		if end > len(tiles) {
			end = len(tiles)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, tiles[i:end]...)
		rows = append(rows, row)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}
