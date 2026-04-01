package tui

import tea "github.com/charmbracelet/bubbletea"

// Screen is the contract every top-level screen must satisfy.
// It is intentionally identical to tea.Model so any screen can also
// be passed directly to Bubble Tea.
type Screen interface {
	tea.Model
}

// Compile-time assertions: every concrete screen must satisfy Screen.
var (
	_ Screen = LoginScreen{}
	_ Screen = FilterScreen{}
	_ Screen = ResultsScreen{}
	_ Screen = DetailScreen{}
)
