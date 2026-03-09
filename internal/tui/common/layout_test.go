package common_test

import (
	"strings"
	"testing"

	"github.com/gfreschi/k6delta/internal/tui/common"
)

func TestRenderTileGrid_multipleRows(t *testing.T) {
	tiles := []string{"[A]", "[B]", "[C]", "[D]"}
	result := common.RenderTileGrid(tiles, 2)
	if !strings.Contains(result, "[A]") || !strings.Contains(result, "[D]") {
		t.Errorf("expected all tiles in output, got %q", result)
	}
	lines := strings.Split(result, "\n")
	// With 2 rows and 1 gap line between them, expect > 2 lines
	if len(lines) < 3 {
		t.Errorf("expected gap between rows, got %d lines", len(lines))
	}
}

func TestRenderTileGrid_singleRow(t *testing.T) {
	tiles := []string{"[A]", "[B]"}
	result := common.RenderTileGrid(tiles, 4)
	if !strings.Contains(result, "[A]") || !strings.Contains(result, "[B]") {
		t.Errorf("expected tiles in output, got %q", result)
	}
}

func TestRenderTileGrid_empty(t *testing.T) {
	result := common.RenderTileGrid(nil, 2)
	if result != "" {
		t.Errorf("expected empty string for nil tiles, got %q", result)
	}
}

func TestRenderCompactList(t *testing.T) {
	items := []common.CompactItem{
		{Label: "CPU", Value: "55%"},
		{Label: "Memory", Value: "512MB"},
	}
	result := common.RenderCompactList(items)
	if !strings.Contains(result, "CPU") || !strings.Contains(result, "55%") {
		t.Errorf("compact list missing content: %q", result)
	}
	if !strings.Contains(result, "Memory") || !strings.Contains(result, "512MB") {
		t.Errorf("compact list missing content: %q", result)
	}
}

func TestRenderCompactList_empty(t *testing.T) {
	result := common.RenderCompactList(nil)
	if result != "" {
		t.Errorf("expected empty string for nil items, got %q", result)
	}
}
