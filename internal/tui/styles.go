// Package tui provides shared Lip Gloss styles and step-tracking
// components for the k6delta terminal UI.
package tui

import "github.com/charmbracelet/lipgloss"

var (
	HeaderStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	WarnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	ErrorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	DimStyle     = lipgloss.NewStyle().Faint(true)
	BoldStyle    = lipgloss.NewStyle().Bold(true)
	TitleStyle   = lipgloss.NewStyle().Bold(true).Underline(true)
	LabelStyle   = lipgloss.NewStyle().Width(26)
)
