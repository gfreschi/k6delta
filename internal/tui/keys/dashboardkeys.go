package keys

import "github.com/charmbracelet/bubbles/key"

// DashboardKeyMap defines key bindings for the dashboard browsing view.
type DashboardKeyMap struct {
	Run       key.Binding
	Analyze   key.Binding
	Compare   key.Binding
	Reports   key.Binding
	Demo      key.Binding
	NextPhase key.Binding
	PrevPhase key.Binding
}

// DashboardKeys is the global dashboard key binding set.
var DashboardKeys = DashboardKeyMap{
	Run:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
	Analyze:   key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "analyze")),
	Compare:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "compare")),
	Reports:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reports")),
	Demo:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "demo")),
	NextPhase: key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "next phase")),
	PrevPhase: key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "prev phase")),
}

// ShortHelp returns key bindings for the compact dashboard help view.
func (k DashboardKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Run, k.Analyze, k.Reports, k.NextPhase}
}

// FullHelp returns key bindings for the expanded dashboard help view.
func (k DashboardKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{Keys.Up, Keys.Down, k.NextPhase, k.PrevPhase},
		{k.Run, k.Analyze, k.Compare, k.Reports, k.Demo},
		{Keys.Quit, Keys.Help},
	}
}
