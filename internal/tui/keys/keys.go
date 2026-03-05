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
	Expand    key.Binding
	Enter     key.Binding
	Escape    key.Binding
	Jump1     key.Binding
	Jump2     key.Binding
	Jump3     key.Binding
	Jump4     key.Binding
}

// Keys is the global universal key binding set.
var Keys = KeyMap{
	Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	NextPanel: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
	PrevPanel: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev panel")),
	Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑", "up")),
	Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓", "down")),
	Expand:    key.NewBinding(key.WithKeys("+", "="), key.WithHelp("+", "expand")),
	Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
	Escape:    key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "collapse")),
	Jump1:     key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "panel 1")),
	Jump2:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "panel 2")),
	Jump3:     key.NewBinding(key.WithKeys("3"), key.WithHelp("3", "panel 3")),
	Jump4:     key.NewBinding(key.WithKeys("4"), key.WithHelp("4", "panel 4")),
}

// ShortHelp returns key bindings for the compact help view.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.NextPanel, k.Up, k.Down, k.Expand}
}

// FullHelp returns key bindings for the expanded help view.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Help, k.Escape},
		{k.NextPanel, k.PrevPanel, k.Up, k.Down},
		{k.Expand, k.Enter, k.Jump1, k.Jump2, k.Jump3, k.Jump4},
	}
}
