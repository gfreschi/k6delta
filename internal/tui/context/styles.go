package context

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/gfreschi/k6delta/internal/tui/common"
	"github.com/gfreschi/k6delta/internal/tui/theme"
)

// Styles holds pre-built lipgloss styles grouped by component.
type Styles struct {
	Common    common.CommonStyles
	Header    HeaderStyles
	Panel     PanelStyles
	Table     TableStyles
	Footer    FooterStyles
	Stepper   StepperStyles
	Verdict   VerdictStyles
	Delta     DeltaStyles
	Chart     ChartStyles
	Tile      TileStyles
	Timeline  TimelineStyles
	StatusBar StatusBarStyles
	Overlay   OverlayStyles
	Layout    LayoutStyles
}

// ChartStyles for ntcharts chart configuration (axis, label, line).
type ChartStyles struct {
	Axis  lipgloss.Style
	Label lipgloss.Style
	Line  lipgloss.Style
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

// TileStyles for KPI tile borders with status coloring.
type TileStyles struct {
	Border      lipgloss.Style
	BorderOK    lipgloss.Style
	BorderWarn  lipgloss.Style
	BorderError lipgloss.Style
}

// OverlayStyles for centered modal overlays (e.g., help screen).
type OverlayStyles struct {
	Box lipgloss.Style
}

// TimelineStyles for event timeline entries.
type TimelineStyles struct {
	Alarm     lipgloss.Style
	Scaling   lipgloss.Style
	Resolved  lipgloss.Style
	Lane      lipgloss.Style
	Threshold lipgloss.Style
}

// StatusBarStyles for the bottom status bar.
type StatusBarStyles struct {
	Root  lipgloss.Style
	Label lipgloss.Style
	Value lipgloss.Style
}

// LayoutStyles for structural containers with no visual styling.
type LayoutStyles struct {
	Column lipgloss.Style
}

// DeltaStyles for comparison delta coloring with intensity tiers.
type DeltaStyles struct {
	Better        lipgloss.Style
	BetterMild    lipgloss.Style
	BetterStrong  lipgloss.Style
	Worse       lipgloss.Style
	WorseMild   lipgloss.Style
	WorseSevere lipgloss.Style
	Neutral       lipgloss.Style
}

// Tiers returns a common.DeltaStyleTiers for use with common.DeltaStyle.
func (ds DeltaStyles) Tiers() common.DeltaStyleTiers {
	return common.DeltaStyleTiers{
		Better:        ds.Better,
		BetterMild:    ds.BetterMild,
		BetterStrong:  ds.BetterStrong,
		Worse:       ds.Worse,
		WorseMild:   ds.WorseMild,
		WorseSevere: ds.WorseSevere,
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
		Chart: ChartStyles{
			Axis:  lipgloss.NewStyle().Foreground(t.FaintText),
			Label: lipgloss.NewStyle().Foreground(t.FaintText),
			Line:  lipgloss.NewStyle().Foreground(t.PrimaryText),
		},
		Tile: TileStyles{
			Border:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.TileBorder),
			BorderOK:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.TileBorderOK),
			BorderWarn:  lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.TileBorderWarn),
			BorderError: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(t.TileBorderError),
		},
		Timeline: TimelineStyles{
			Alarm:     lipgloss.NewStyle().Foreground(t.TimelineAlarm),
			Scaling:   lipgloss.NewStyle().Foreground(t.TimelineScaling),
			Resolved:  lipgloss.NewStyle().Foreground(t.TimelineResolved),
			Lane:      lipgloss.NewStyle().Foreground(t.FaintText),
			Threshold: lipgloss.NewStyle().Foreground(t.WarningText).Faint(true),
		},
		StatusBar: StatusBarStyles{
			Root:  lipgloss.NewStyle().Foreground(t.FaintText),
			Label: lipgloss.NewStyle().Bold(true).Foreground(t.SecondaryText),
			Value: lipgloss.NewStyle().Foreground(t.PrimaryText),
		},
		Overlay: OverlayStyles{
			Box: lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(t.FocusedBorder).
				Padding(1, 2),
		},
		Layout: LayoutStyles{
			Column: lipgloss.NewStyle(),
		},
		Delta: DeltaStyles{
			Better:        lipgloss.NewStyle().Foreground(t.DeltaBetter),
			BetterMild:    lipgloss.NewStyle().Foreground(t.DeltaBetter).Faint(true),
			BetterStrong:  lipgloss.NewStyle().Bold(true).Foreground(t.DeltaBetter),
			Worse:       lipgloss.NewStyle().Foreground(t.DeltaWorse),
			WorseMild:   lipgloss.NewStyle().Foreground(t.DeltaWorse).Faint(true),
			WorseSevere: lipgloss.NewStyle().Bold(true).Foreground(t.DeltaWorse),
			Neutral:       lipgloss.NewStyle().Foreground(t.DeltaNeutral),
		},
	}
}
