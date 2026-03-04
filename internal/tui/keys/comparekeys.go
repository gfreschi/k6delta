package keys

import "github.com/charmbracelet/bubbles/key"

// CompareKeyMap defines keys specific to the compare view.
type CompareKeyMap struct {
	Export key.Binding
	Sort   key.Binding
}

// CompareKeys is the global compare-specific key binding set.
var CompareKeys = CompareKeyMap{
	Export: key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export JSON")),
	Sort:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "sort")),
}
