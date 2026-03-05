// Package overlay provides a centered help overlay component.
package overlay

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tuictx "github.com/gfreschi/k6delta/internal/tui/context"
)

// HelpGroup defines a group of keybindings for the help overlay.
type HelpGroup struct {
	Title string
	Keys  [][2]string
}

// RenderHelp renders a centered help overlay with the given key groups.
func RenderHelp(ctx *tuictx.ProgramContext, groups []HelpGroup) string {
	s := ctx.Styles
	w := ctx.ContentWidth
	h := ctx.ContentHeight

	var lines []string
	lines = append(lines, s.Header.Root.Render("Keyboard Shortcuts"), "")
	for _, g := range groups {
		lines = append(lines, s.Common.BoldStyle.Render("  "+g.Title))
		for _, kv := range g.Keys {
			lines = append(lines, fmt.Sprintf("    %-22s %s", s.Footer.Key.Render(kv[0]), kv[1]))
		}
		lines = append(lines, "")
	}
	lines = append(lines, s.Common.FaintTextStyle.Render("  Press ? or esc to close"))

	content := strings.Join(lines, "\n")
	box := s.Overlay.Box.
		Width(min(w-4, 60)).
		Height(min(h-2, len(lines)+2)).
		Render(content)

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box)
}
