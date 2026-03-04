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
