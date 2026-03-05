package keys

import "github.com/charmbracelet/bubbles/key"

// RunKeyMap defines keys specific to the run view.
type RunKeyMap struct {
	Export   key.Binding
	OpenHTML key.Binding
	RawView  key.Binding
}

// RunKeys is the global run-specific key binding set.
var RunKeys = RunKeyMap{
	Export:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export JSON")),
	OpenHTML: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "open HTML")),
	RawView:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "raw view")),
}

// LiveKeyMap defines keys specific to the live dashboard mode.
type LiveKeyMap struct {
	ToggleGraphs key.Binding
	Abort        key.Binding
}

// LiveKeys is the global live-mode key binding set.
var LiveKeys = LiveKeyMap{
	ToggleGraphs: key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "graphs")),
	Abort:        key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "abort")),
}
