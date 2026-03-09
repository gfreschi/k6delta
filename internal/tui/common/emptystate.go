package common

import "github.com/charmbracelet/lipgloss"

// EmptyStateVariant selects the icon and styling for an empty state.
type EmptyStateVariant int

const (
	EmptyNoData  EmptyStateVariant = iota // "--" icon, no data available
	EmptyPending                          // "..." icon, data loading
	EmptyError                            // "!" icon, error occurred
)

func emptyIcon(v EmptyStateVariant) string {
	switch v {
	case EmptyPending:
		return "..."
	case EmptyError:
		return "!"
	default:
		return "--"
	}
}

// RenderEmptyState renders a centered empty state block with icon, title, and optional subtitle.
func RenderEmptyState(styles CommonStyles, variant EmptyStateVariant, title, subtitle string) string {
	icon := styles.FaintTextStyle.Render(emptyIcon(variant))
	titleLine := styles.FaintTextStyle.Render(title)

	var lines []string
	lines = append(lines, "", icon, titleLine)
	if subtitle != "" {
		lines = append(lines, styles.FaintTextStyle.Render(subtitle))
	}
	lines = append(lines, "")

	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}
