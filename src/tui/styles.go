package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// ColorPrimary is the main accent colour.
	ColorPrimary = lipgloss.Color("#7C3AED")
	// ColorMuted is used for secondary / inactive text.
	ColorMuted = lipgloss.Color("#6B7280")
	// ColorError is used for error states and FATAL/ERROR severity.
	ColorError = lipgloss.Color("#EF4444")
	// ColorWarn is used for WARN severity.
	ColorWarn = lipgloss.Color("#F59E0B")
	// ColorSuccess is used for healthy DC indicators.
	ColorSuccess = lipgloss.Color("#10B981")
	// ColorInfo is used for INFO severity.
	ColorInfo = lipgloss.Color("#3B82F6")

	// StatusBar is the bottom status bar style.
	StatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(lipgloss.Color("#F9FAFB")).
			Padding(0, 1)

	// StatusBarKey highlights a key hint within the status bar.
	StatusBarKey = lipgloss.NewStyle().
			Background(lipgloss.Color("#374151")).
			Foreground(lipgloss.Color("#F9FAFB")).
			Padding(0, 1)

	// Title is the screen title style.
	Title = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true).
		Padding(0, 1)

	// FocusedField is the style for the currently focused form field.
	FocusedField = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	// BlurredField is the style for unfocused form fields.
	BlurredField = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted).
			Padding(0, 1)

	// SelectedRow is the style for the highlighted table row.
	SelectedRow = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)

	// SeverityStyles maps syslog severity names to their display style.
	SeverityStyles = map[string]lipgloss.Style{
		"Emergency": lipgloss.NewStyle().Foreground(ColorError).Bold(true),
		"Alert":     lipgloss.NewStyle().Foreground(ColorError).Bold(true),
		"Critical":  lipgloss.NewStyle().Foreground(ColorError).Bold(true),
		"Error":     lipgloss.NewStyle().Foreground(ColorError),
		"Warning":   lipgloss.NewStyle().Foreground(ColorWarn),
		"Notice":    lipgloss.NewStyle().Foreground(ColorInfo),
		"Info":      lipgloss.NewStyle().Foreground(ColorInfo),
		"Debug":     lipgloss.NewStyle().Foreground(ColorMuted),
	}

	// DCHealthOK is shown in the status bar for a responsive data center.
	DCHealthOK = lipgloss.NewStyle().Foreground(ColorSuccess)
	// DCHealthErr is shown in the status bar for a failed data center.
	DCHealthErr = lipgloss.NewStyle().Foreground(ColorError)
)

// SeverityStyle returns the style for the given severity label.
// It handles both plain names ("Error") and "Name (N)" display labels.
func SeverityStyle(level string) lipgloss.Style {
	name := level
	if i := strings.Index(level, " ("); i >= 0 {
		name = level[:i]
	}
	if s, ok := SeverityStyles[name]; ok {
		return s
	}
	return lipgloss.NewStyle()
}
