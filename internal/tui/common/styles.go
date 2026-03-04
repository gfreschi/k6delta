// Package common provides shared base styles and pre-rendered glyphs.
package common

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/constants"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

// CommonStyles holds base styles and pre-rendered glyph strings.
type CommonStyles struct {
	MainTextStyle  lipgloss.Style
	FaintTextStyle lipgloss.Style
	BoldStyle      lipgloss.Style
	ErrorStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	WarnStyle      lipgloss.Style
	// Pre-rendered glyphs (styled strings ready to print).
	CheckMark   string
	XMark       string
	WarningSign string
	Bullet      string
}

// BuildStyles constructs CommonStyles from a Theme.
func BuildStyles(t theme.Theme) CommonStyles {
	return CommonStyles{
		MainTextStyle:  lipgloss.NewStyle().Foreground(t.PrimaryText),
		FaintTextStyle: lipgloss.NewStyle().Foreground(t.FaintText),
		BoldStyle:      lipgloss.NewStyle().Bold(true),
		ErrorStyle:     lipgloss.NewStyle().Foreground(t.ErrorText),
		SuccessStyle:   lipgloss.NewStyle().Foreground(t.SuccessText),
		WarnStyle:      lipgloss.NewStyle().Foreground(t.WarningText),
		CheckMark:      lipgloss.NewStyle().Foreground(t.SuccessText).Render(constants.IconDone),
		XMark:          lipgloss.NewStyle().Foreground(t.ErrorText).Render(constants.IconFailed),
		WarningSign:    lipgloss.NewStyle().Foreground(t.WarningText).Render(constants.IconWarning),
		Bullet:         lipgloss.NewStyle().Foreground(t.FaintText).Render(constants.IconBullet),
	}
}
