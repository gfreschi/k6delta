package context

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

// Styles holds pre-built lipgloss styles grouped by component.
type Styles struct {
	Common  common.CommonStyles
	Header  HeaderStyles
	Panel   PanelStyles
	Table   TableStyles
	Footer  FooterStyles
	Stepper StepperStyles
	Verdict VerdictStyles
	Delta   DeltaStyles
}

// HeaderStyles for the app/env/phase context bar.
type HeaderStyles struct {
	Root    lipgloss.Style
	Title   lipgloss.Style
	Context lipgloss.Style
}

// PanelStyles for bordered panels with focus state.
type PanelStyles struct {
	Root    lipgloss.Style
	Focused lipgloss.Style
	Border  lipgloss.Style
}

// TableStyles for metric and comparison tables.
type TableStyles struct {
	Header       lipgloss.Style
	Row          lipgloss.Style
	RowAlt       lipgloss.Style
	Cell         lipgloss.Style
	SelectedCell lipgloss.Style
	Separator    lipgloss.Style
	Label        lipgloss.Style
}

// FooterStyles for the keybinding bar.
type FooterStyles struct {
	Root      lipgloss.Style
	Key       lipgloss.Style
	Action    lipgloss.Style
	Separator lipgloss.Style
}

// StepperStyles for the step tracker.
type StepperStyles struct {
	Pending lipgloss.Style
	Running lipgloss.Style
	Done    lipgloss.Style
	Failed  lipgloss.Style
	Detail  lipgloss.Style
	Elapsed lipgloss.Style
}

// VerdictStyles for pass/warn/fail rendering.
type VerdictStyles struct {
	Pass   lipgloss.Style
	Warn   lipgloss.Style
	Fail   lipgloss.Style
	Reason lipgloss.Style
}

// DeltaStyles for comparison delta coloring with intensity tiers.
type DeltaStyles struct {
	Better        lipgloss.Style
	BetterMild    lipgloss.Style
	BetterStrong  lipgloss.Style
	Worse         lipgloss.Style
	WorseMild     lipgloss.Style
	WorseModerate lipgloss.Style
	WorseSevere   lipgloss.Style
	Neutral       lipgloss.Style
}

// Tiers returns a common.DeltaStyleTiers for use with common.DeltaStyle.
func (ds DeltaStyles) Tiers() common.DeltaStyleTiers {
	return common.DeltaStyleTiers{
		Better:        ds.Better,
		BetterMild:    ds.BetterMild,
		BetterStrong:  ds.BetterStrong,
		Worse:         ds.Worse,
		WorseMild:     ds.WorseMild,
		WorseModerate: ds.WorseModerate,
		WorseSevere:   ds.WorseSevere,
		Neutral:       ds.Neutral,
	}
}

// InitStyles builds all styles from a theme. Called once at startup.
func InitStyles(t theme.Theme) Styles {
	return Styles{
		Common: common.BuildStyles(t),
		Header: HeaderStyles{
			Root:    lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText),
			Title:   lipgloss.NewStyle().Bold(true).Underline(true),
			Context: lipgloss.NewStyle().Foreground(t.FaintText),
		},
		Panel: PanelStyles{
			Root:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.PrimaryBorder),
			Focused: lipgloss.NewStyle().Border(lipgloss.DoubleBorder()).BorderForeground(t.FocusedBorder),
			Border:  lipgloss.NewStyle().Foreground(t.PrimaryBorder),
		},
		Table: TableStyles{
			Header:    lipgloss.NewStyle().Bold(true).Foreground(t.SecondaryText),
			Row:       lipgloss.NewStyle().Foreground(t.PrimaryText),
			RowAlt:    lipgloss.NewStyle().Foreground(t.PrimaryText).Faint(true),
			Cell:      lipgloss.NewStyle().Foreground(t.PrimaryText),
			Separator: lipgloss.NewStyle().Foreground(t.FaintBorder),
			Label:     lipgloss.NewStyle().Width(26).Foreground(t.FaintText),
		},
		Footer: FooterStyles{
			Root:      lipgloss.NewStyle().Foreground(t.FaintText),
			Key:       lipgloss.NewStyle().Bold(true).Foreground(t.HeaderText),
			Action:    lipgloss.NewStyle().Foreground(t.FaintText),
			Separator: lipgloss.NewStyle().Foreground(t.FaintBorder),
		},
		Stepper: StepperStyles{
			Pending: lipgloss.NewStyle().Foreground(t.FaintText),
			Running: lipgloss.NewStyle().Bold(true).Foreground(t.PrimaryText),
			Done:    lipgloss.NewStyle().Foreground(t.SuccessText),
			Failed:  lipgloss.NewStyle().Foreground(t.ErrorText),
			Detail:  lipgloss.NewStyle().Foreground(t.SecondaryText),
			Elapsed: lipgloss.NewStyle().Foreground(t.FaintText),
		},
		Verdict: VerdictStyles{
			Pass:   lipgloss.NewStyle().Bold(true).Foreground(t.SuccessText),
			Warn:   lipgloss.NewStyle().Bold(true).Foreground(t.WarningText),
			Fail:   lipgloss.NewStyle().Bold(true).Foreground(t.ErrorText),
			Reason: lipgloss.NewStyle().Foreground(t.SecondaryText),
		},
		Delta: DeltaStyles{
			Better:        lipgloss.NewStyle().Foreground(t.DeltaBetter),
			BetterMild:    lipgloss.NewStyle().Foreground(t.DeltaBetter).Faint(true),
			BetterStrong:  lipgloss.NewStyle().Bold(true).Foreground(t.DeltaBetter),
			Worse:         lipgloss.NewStyle().Foreground(t.DeltaWorse),
			WorseMild:     lipgloss.NewStyle().Foreground(t.DeltaWorse).Faint(true),
			WorseModerate: lipgloss.NewStyle().Foreground(t.DeltaWorse),
			WorseSevere:   lipgloss.NewStyle().Bold(true).Foreground(t.DeltaWorse),
			Neutral:       lipgloss.NewStyle().Foreground(t.DeltaNeutral),
		},
	}
}
