// Package keys defines all keyboard bindings for the k6delta TUI.
package keys

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines universal key bindings available in all views.
type KeyMap struct {
	Quit      key.Binding
	Help      key.Binding
	NextPanel key.Binding
	PrevPanel key.Binding
	Up        key.Binding
	Down      key.Binding
}

// Keys is the global universal key binding set.
var Keys = KeyMap{
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"), key.WithHelp("q", "quit")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	NextPanel: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
	PrevPanel: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
	Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
	Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
}

// ShortHelp returns key bindings for the compact help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.NextPanel, k.Up, k.Down}
}

// FullHelp returns key bindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Help},
		{k.NextPanel, k.PrevPanel, k.Up, k.Down},
	}
}
