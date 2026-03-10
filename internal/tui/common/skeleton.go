package common

import "strings"

// SkeletonTable renders a faint placeholder table with header and rows.
func SkeletonTable(styles CommonStyles, width, rows int) string {
	s := styles.FaintTextStyle
	barW := min(width, 40)
	header := s.Render(strings.Repeat("_", barW))
	var lines []string
	lines = append(lines, header)
	for i := range rows {
		w := barW
		if i%2 == 1 {
			w = barW * 3 / 4
		}
		lines = append(lines, s.Render(strings.Repeat("_", w)))
	}
	return strings.Join(lines, "\n")
}

// SkeletonTileRow renders faint placeholder tile outlines.
func SkeletonTileRow(styles CommonStyles, width, count int) string {
	s := styles.FaintTextStyle
	tileW := min(14, width/max(count, 1))
	var tiles []string
	for range count {
		tile := s.Render("[" + strings.Repeat("_", max(tileW-2, 1)) + "]")
		tiles = append(tiles, tile)
	}
	return strings.Join(tiles, " ")
}

// SkeletonChart renders a faint placeholder chart area.
func SkeletonChart(styles CommonStyles, width, height int) string {
	s := styles.FaintTextStyle
	barW := min(width, 60)
	var lines []string
	for i := range height {
		if i == 0 || i == height-1 {
			lines = append(lines, s.Render(strings.Repeat("_", barW)))
		} else {
			lines = append(lines, s.Render("|"+strings.Repeat(" ", max(barW-2, 0))+"|"))
		}
	}
	return strings.Join(lines, "\n")
}
