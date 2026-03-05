// Package focus provides a simple panel focus cycling manager.
package focus

// Manager tracks which panel index is currently focused.
type Manager struct {
	current int
	count   int
}

// New creates a focus manager for n panels.
func New(count int) *Manager {
	return &Manager{count: count}
}

// Current returns the focused panel index.
func (m *Manager) Current() int { return m.current }

// Next moves focus to the next panel, wrapping around.
func (m *Manager) Next() {
	m.current = (m.current + 1) % m.count
}

// Prev moves focus to the previous panel, wrapping around.
func (m *Manager) Prev() {
	m.current = (m.current - 1 + m.count) % m.count
}

// SetFocus sets focus to the given index, wrapping out-of-range values.
func (m *Manager) SetFocus(index int) {
	m.current = ((index % m.count) + m.count) % m.count
}

// Count returns the number of focusable items.
func (m *Manager) Count() int { return m.count }

// IsFocused returns true if the given index is focused.
func (m *Manager) IsFocused(index int) bool {
	return m.current == index
}
