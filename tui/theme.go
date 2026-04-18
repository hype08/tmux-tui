package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme represents colors derived from the terminal
type Theme struct {
	Background  lipgloss.TerminalColor
	Foreground  lipgloss.TerminalColor
	Accent      lipgloss.Color
	Secondary   lipgloss.Color
	Transparent bool
}

// DefaultTheme returns a theme using terminal's default colors
func DefaultTheme() Theme {
	return Theme{
		Background:  lipgloss.NoColor{},  // Inherit terminal background
		Foreground:  lipgloss.NoColor{},  // Inherit terminal foreground
		Accent:      lipgloss.Color("2"), // Green
		Secondary:   lipgloss.Color("8"), // Bright black/gray
		Transparent: true,                // Start with transparent background
	}
}

// NewStyle creates a lipgloss style with theme colors
func (t Theme) NewStyle() lipgloss.Style {
	style := lipgloss.NewStyle()
	if !t.Transparent {
		style = style.Background(t.Background)
	}
	return style
}
