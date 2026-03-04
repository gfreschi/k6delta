package keys

import "github.com/charmbracelet/bubbles/key"

// AnalyzeKeyMap defines keys specific to the analyze view.
type AnalyzeKeyMap struct {
	Export key.Binding
}

// AnalyzeKeys is the global analyze-specific key binding set.
var AnalyzeKeys = AnalyzeKeyMap{
	Export: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export JSON")),
}
