package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents colors derived from the terminal
type Theme struct {
	Background  lipgloss.Color
	Foreground  lipgloss.Color
	Accent      lipgloss.Color
	Secondary   lipgloss.Color
	Transparent bool
}

// DefaultTheme returns a theme using terminal's default colors
func DefaultTheme() Theme {
	return Theme{
		Background:  lipgloss.Color(""),  // Use terminal default
		Foreground:  lipgloss.Color(""),  // Use terminal default
		Accent:      lipgloss.Color("2"), // Green
		Secondary:   lipgloss.Color("8"), // Bright black/gray
		Transparent: true,                // Start with transparent background
	}
}

// NewStyle creates a lipgloss style with theme colors, only setting them if not empty
func (t Theme) NewStyle() lipgloss.Style {
	style := lipgloss.NewStyle()
	if !t.Transparent && t.Background != "" {
		style = style.Background(t.Background)
	}
	if t.Foreground != "" {
		style = style.Foreground(t.Foreground)
	}
	return style
}
